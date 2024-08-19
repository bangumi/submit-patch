import time
from typing import Any
from urllib.parse import urlencode

import litestar
import orjson
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

from config import BGM_TV_APP_ID, BGM_TV_APP_SECRET, SERVER_BASE_URL
from server.base import Request, http_client
from server.model import User


CALLBACK_URL = f"{SERVER_BASE_URL}/oauth_callback"


async def retrieve_user_from_session(
    session: dict[str, Any], req: ASGIConnection[Any, Any, Any, Any]
) -> User | None:
    try:
        return __user_from_session(session)
    except KeyError:
        return None


def __user_from_session(session: dict[str, Any]) -> User:
    return User(
        user_id=session["user_id"],
        group_id=session["group_id"],
        access_token=session.get("access_token", ""),
        refresh_token=session.get("refresh_token", ""),
        access_token_created_at=session.get("access_token_created_at", 0),
        access_token_expires_in=session.get("access_token_expires_in", 0),
    )


async def refresh(refresh_token: str) -> dict[str, Any]:
    res = await http_client.post(
        "https://bgm.tv/oauth/access_token",
        data={
            "refresh_token": refresh_token,
            "client_id": BGM_TV_APP_ID,
            "grant_type": "refresh_token",
            "client_secret": BGM_TV_APP_SECRET,
        },
    )
    if res.status_code >= 300:
        raise InternalServerException("api request error")
    return orjson.loads(res.content)


class MyAuthenticationMiddleware(SessionAuthMiddleware):
    async def authenticate_request(
        self, connection: ASGIConnection[Any, Any, Any, Any]
    ) -> AuthenticationResult:
        if not connection.session or connection.scope["session"] is Empty:
            # the assignment of 'Empty' forces the session middleware to clear session data.
            connection.scope["session"] = Empty
            return AuthenticationResult(user=None, auth=None)

        user = await retrieve_user_from_session(connection.session, connection)

        return AuthenticationResult(user=user, auth=user)


session_auth_config = SessionAuth[User, ServerSideSessionBackend](
    retrieve_user_handler=retrieve_user_from_session,
    session_backend_config=ServerSideSessionConfig(),
    authentication_middleware_class=MyAuthenticationMiddleware,
)


@litestar.get("/login", sync_to_thread=False)
def login() -> Redirect:
    return Redirect(
        "https://bgm.tv/oauth/authorize?"
        + urlencode(
            {
                "client_id": BGM_TV_APP_ID,
                "response_type": "code",
                "redirect_uri": CALLBACK_URL,
            }
        )
    )


@litestar.get("/oauth_callback")
async def callback(code: str, request: Request) -> Redirect:
    res = await http_client.post(
        "https://bgm.tv/oauth/access_token",
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
    data = res.json()

    user_id = data["user_id"]
    access_token = data["access_token"]

    res = await http_client.get(
        "https://api.bgm.tv/v0/me", headers={"Authorization": f"Bearer {access_token}"}
    )
    if res.status_code >= 300:
        raise InternalServerException("api request error")
    user = res.json()

    group_id = user["user_group"]

    # litestar type this as dict[str, Any], but it maybe Empty
    if isinstance(request.session, dict):
        back_to = request.session.get("backTo", "/")
    else:
        back_to = "/"

    request.set_session(
        {
            "user_id": user_id,
            "group_id": group_id,
            "access_token": access_token,
            "refresh_token": data["refresh_token"],
            "access_token_created_at": time.time(),
            "access_token_expires_in": int(data["expires_in"]),
        }
    )

    return Redirect(back_to)


def require_user_editor(connection: ASGIConnection[Any, Any, Any, Any], _: Any) -> None:
    if not connection.auth:
        raise NotAuthorizedException
    if not connection.auth.allow_edit:
        raise NotAuthorizedException
