import html
import json
import time
from dataclasses import dataclass
from typing import Any, TypedDict
from urllib.parse import urlencode

import litestar
from litestar.connection import ASGIConnection
from litestar.exceptions import InternalServerException, NotAuthorizedException
from litestar.middleware import AuthenticationResult
from litestar.middleware.session.server_side import (
    ServerSideSessionBackend,
    ServerSideSessionConfig,
)
from litestar.response import Redirect
from litestar.security.session_auth import SessionAuth, SessionAuthMiddleware
from litestar.types import Empty
from sslog import logger

from server.base import (
    CTX_REQUEST_ID,
    AuthorizedRequest,
    RedirectException,
    Request,
    User,
    external_http_client,
    http_client,
    pg,
    session_key_back_to,
)
from server.config import (
    BGM_TV_APP_ID,
    BGM_TV_APP_SECRET,
    HEADER_KEY_API,
    SERVER_BASE_URL,
    SUPER_USERS,
)
from server.router import Router


CALLBACK_URL = f"{SERVER_BASE_URL}/callback"

router = Router()


@dataclass(frozen=True, slots=True, kw_only=True)
class SuperUser(User):
    def is_access_token_fresh(self) -> bool:
        return True

    @property
    def allow_edit(self) -> bool:
        return True

    def super_user(self) -> bool:
        return True


class SessionDict(TypedDict):
    user_id: int
    group_id: int
    access_token: str
    refresh_token: str
    access_token_created_at: int
    access_token_expires_in: int
    tz: int


async def retrieve_user_from_session(
    session: dict[str, Any], _: ASGIConnection[Any, Any, Any, Any]
) -> User | None:
    try:
        return __user_from_session(session)  # type: ignore
    except KeyError:
        return None


def __user_from_session(session: SessionDict) -> User:
    return User(
        user_id=session["user_id"],
        group_id=session["group_id"],
        access_token=session["access_token"],
        refresh_token=session["refresh_token"],
        access_token_created_at=session["access_token_created_at"],
        access_token_expires_in=session["access_token_expires_in"],
        time_offset=session["tz"],
    )


class OAuthResponse(TypedDict):
    user_id: int
    expires_in: int
    access_token: str
    refresh_token: str
    token_type: str


class MyAuthenticationMiddleware(SessionAuthMiddleware):
    async def authenticate_request(
        self, connection: ASGIConnection[Any, Any, Any, Any]
    ) -> AuthenticationResult:
        api_token = connection.headers.get(HEADER_KEY_API)
        if api_token:
            if su := SUPER_USERS.get(api_token):
                return AuthenticationResult(
                    user=None,
                    auth=SuperUser(
                        user_id=su["user_id"],
                        group_id=su["user_id"],
                        access_token="",
                        refresh_token="",
                        access_token_created_at=0,
                        access_token_expires_in=0,
                        time_offset=8,
                    ),
                )

        if not connection.session or connection.scope["session"] is Empty:
            # the assignment of 'Empty' forces the session middleware to clear session data.
            return AuthenticationResult(user=None, auth=None)

        user = await retrieve_user_from_session(connection.session, connection)

        return AuthenticationResult(user=user, auth=user)


session_auth_config = SessionAuth[User, ServerSideSessionBackend](
    retrieve_user_handler=retrieve_user_from_session,
    session_backend_config=ServerSideSessionConfig(),
    authentication_middleware_class=MyAuthenticationMiddleware,
)


@router
@litestar.get("/login", sync_to_thread=False)
def login() -> Redirect:
    return Redirect(
        "https://next.bgm.tv/oauth/authorize?"
        + urlencode(
            {
                "client_id": BGM_TV_APP_ID,
                "response_type": "code",
                "redirect_uri": CALLBACK_URL,
            }
        )
    )


@router
@litestar.get("/callback")
async def callback(code: str, request: Request) -> Redirect:
    res = await http_client.post(
        "https://next.bgm.tv/oauth/access_token",
        headers={
            "cf-ray": CTX_REQUEST_ID.get(),
        },
        data={
            "code": code,
            "client_id": BGM_TV_APP_ID,
            "grant_type": "authorization_code",
            "redirect_uri": CALLBACK_URL,
            "client_secret": BGM_TV_APP_SECRET,
        },
    )
    if res.status_code >= 300:
        raise InternalServerException("api request error")
    data: OAuthResponse = res.json()

    user_id = int(data["user_id"])

    access_token = data["access_token"]

    res = await external_http_client.get(
        "https://api.bgm.tv/v0/me", headers={"Authorization": f"Bearer {access_token}"}
    )

    if res.status_code >= 300:
        raise InternalServerException("api request error")
    user = res.json()

    group_id = user["user_group"]

    await pg.execute(
        """
        insert into patch_users (user_id, username, nickname) VALUES ($1,$2,$3)
        on conflict (user_id) do update set
            username = excluded.username,
            nickname = excluded.nickname
    """,
        user_id,
        user["username"],
        html.unescape(user["nickname"]),
    )
    back_to = request.session.get(session_key_back_to, "/")

    request.set_session(
        SessionDict(
            user_id=user_id,
            group_id=group_id,
            access_token=access_token,
            tz=user["time_offset"],
            refresh_token=data["refresh_token"],
            access_token_created_at=int(time.time()),
            access_token_expires_in=int(data["expires_in"]),
        )
    )

    return Redirect(back_to)


def require_user_login(connection: ASGIConnection[Any, Any, Any, Any], _: Any) -> None:
    if not connection.auth:
        raise NotAuthorizedException("require user to login before this action")


def require_user_editor(connection: ASGIConnection[Any, Any, Any, Any], _: Any = None) -> None:
    if not connection.auth:
        raise NotAuthorizedException("require user to login before this action")
    if not connection.auth.allow_edit:
        raise NotAuthorizedException("you don't have wiki edit permission")


async def refresh_access_token(request: AuthorizedRequest, back_to: str) -> None:
    logger.debug("refresh_access_token")
    if not request.auth.access_token:
        logger.debug("no access token")
    elif not request.auth.access_token_created_at:
        logger.debug("no access_token_created_at")
    elif not request.auth.access_token_expires_in:
        logger.debug("no access_token_expires_in")
    else:
        logger.debug(
            "token create at {},expires in {}, now {}",
            request.auth.access_token_created_at,
            request.auth.access_token_expires_in,
            time.time(),
        )

    auth: User = request.auth
    res = await http_client.post(
        "https://next.bgm.tv/oauth/access_token",
        headers={"cf-ray": CTX_REQUEST_ID.get()},
        timeout=10,
        data={
            "grant_type": "refresh_token",
            "refresh_token": auth.refresh_token,
            "client_id": BGM_TV_APP_ID,
            "redirect_uri": CALLBACK_URL,
            "client_secret": BGM_TV_APP_SECRET,
        },
    )
    if res.status_code >= 300:
        try:
            err_data = res.json()
            if err_data.get("code") == "INVALID_REFRESH_TOKEN":
                request.set_session({session_key_back_to: back_to})
                raise RedirectException("/login")
        except json.JSONDecodeError:
            pass
        logger.error("failed to refresh token")
        raise InternalServerException("oauth server error")

    data: OAuthResponse = res.json()

    user_id = int(data["user_id"])

    access_token = data["access_token"]

    res = await external_http_client.get(
        "https://api.bgm.tv/v0/me", headers={"Authorization": f"Bearer {access_token}"}
    )

    if res.status_code >= 300:
        raise InternalServerException("api request error")
    user = res.json()

    group_id = user["user_group"]

    await pg.execute(
        """
        insert into patch_users (user_id, username, nickname) VALUES ($1,$2,$3)
        on conflict (user_id) do update set
            username = excluded.username,
            nickname = excluded.nickname
    """,
        user_id,
        user["username"],
        html.unescape(user["nickname"]),
    )

    s = SessionDict(
        user_id=user_id,
        group_id=group_id,
        access_token=access_token,
        tz=user["time_offset"],
        refresh_token=data["refresh_token"],
        access_token_created_at=int(time.time()),
        access_token_expires_in=int(data["expires_in"]),
    )

    request.set_session(s)

    request.scope["auth"] = __user_from_session(s)
