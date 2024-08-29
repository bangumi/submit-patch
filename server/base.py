import time
from dataclasses import dataclass
from typing import Any

import asyncpg
import httpx
import litestar
from frozendict import frozendict
from litestar.exceptions import ClientException
from litestar.status_codes import HTTP_400_BAD_REQUEST
from loguru import logger
from redis.asyncio import Redis
from typing_extensions import Never

from config import PG_DSN, REDIS_DSN


session_key_back_to = "backTo"


@dataclass(frozen=True, slots=True, kw_only=True)
class User:
    user_id: int
    group_id: int

    access_token: str
    refresh_token: str

    access_token_created_at: int  # unix time stamp
    access_token_expires_in: int  # seconds

    def is_access_token_fresh(self) -> bool:
        if not self.access_token:
            return False

        if self.access_token_created_at + self.access_token_expires_in <= time.time() + 120:
            return False

        return True

    @property
    def allow_edit(self) -> bool:
        return self.group_id in {1, 2, 9, 11}

    def allow_bypass_captcha(self) -> bool:
        return self.user_id == 287622


redis_client = Redis.from_url(REDIS_DSN)

http_client = httpx.AsyncClient(
    follow_redirects=False, headers={"user-agent": "trim21/submit-patch"}
)
pg = asyncpg.create_pool(dsn=PG_DSN)


async def pg_pool_startup(*args: Any, **kwargs: Any) -> None:
    logger.info("init")
    await pg


Request = litestar.Request[Never, User | None, Any]

AuthorizedRequest = litestar.Request[Never, User, Any]


class BadRequestException(ClientException):
    """Server knows the request method, but the target resource doesn't support this method."""

    status_code = HTTP_400_BAD_REQUEST


patch_keys: frozendict[str, str] = frozendict(
    {
        "name": "标题",
        "name_cn": "简体中文标题",
        "duration": "时长",
        "airdate": "放送日期",
        "description": "简介",
    }
)
