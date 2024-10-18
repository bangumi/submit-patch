import asyncio
import contextvars
import time
from collections.abc import Mapping
from dataclasses import dataclass
from typing import Any, TypeAlias
from uuid import UUID

import asyncpg
import httpx
import litestar
from litestar.enums import ScopeType
from litestar.middleware import AbstractMiddleware
from litestar.types import Receive, Scope, Send
from redis.asyncio import Redis
from sslog import logger
from uuid_utils.compat import uuid7

from server.config import PG_DSN, REDIS_DSN


session_key_back_to = "backTo"


class RedirectException(Exception):
    location: str

    def __init__(self, location: str):
        self.location = location


@dataclass(frozen=True, slots=True, kw_only=True)
class User:
    user_id: int
    group_id: int

    time_offset: int

    access_token: str
    refresh_token: str

    access_token_created_at: int  # unix time stamp
    access_token_expires_in: int  # seconds

    def is_access_token_fresh(self) -> bool:
        if not self.access_token:
            return False

        if not self.access_token_created_at:
            return False

        if not self.access_token_expires_in:
            return False

        return self.access_token_created_at + self.access_token_expires_in > time.time() + 120

    @property
    def allow_edit(self) -> bool:
        return self.group_id in {1, 2, 9, 11}

    def super_user(self) -> bool:
        return self.user_id in {287622, 427613}


@dataclass(frozen=True, kw_only=True, slots=True)
class QueueItem:
    patch_id: UUID
    infobox: str


subject_infobox_queue = asyncio.Queue[QueueItem](maxsize=128)

redis_client = Redis.from_url(REDIS_DSN)

http_client = httpx.AsyncClient(
    follow_redirects=False,
    headers={"user-agent": "trim21/submit-patch"},
)
pg = asyncpg.create_pool(dsn=PG_DSN, server_settings={"application_name": "patch"})


async def pg_pool_startup() -> None:
    logger.info("init")
    await pg


Request = litestar.Request[None, User | None, Any]

AuthorizedRequest: TypeAlias = litestar.Request[None, User, Any]

patch_keys: Mapping[str, str] = {
    "name": "标题",
    "name_cn": "简体中文标题",
    "duration": "时长",
    "airdate": "放送日期",
    "description": "简介",
}

disable_cookies_opt = {"skip_session": True, "exclude_from_auth": True, "exclude_from_csrf": True}

CTX_REQUEST_ID: contextvars.ContextVar[str] = contextvars.ContextVar("request.id", default="")


class XRequestIdMiddleware(AbstractMiddleware):
    """If X-Request-Id is in the request headers, add it to the response headers.

    Reference: https://docs.litestar.dev/2/usage/middleware/creating-middleware.html

    """

    scopes = {ScopeType.HTTP}  # noqa: RUF012

    async def __call__(
        self,
        scope: Scope,
        receive: Receive,
        send: Send,
    ) -> None:
        async def send_wrapper(message: litestar.types.Message) -> None:
            if message["type"] != "http.response.start":
                reset = CTX_REQUEST_ID.set(str(uuid7()))
            else:
                request_id_header: str = litestar.Request(scope).headers.get("x-request-id") or str(
                    uuid7()
                )
                reset = CTX_REQUEST_ID.set(request_id_header)
            try:
                await send(message)
            finally:
                CTX_REQUEST_ID.reset(reset)

        await self.app(scope, receive, send_wrapper)
