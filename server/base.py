from http.cookies import BaseCookie
from typing import Any

import aiohttp
import asyncpg
import litestar
from aiohttp.abc import AbstractCookieJar, ClearCookiePredicate
from aiohttp.typedefs import LooseCookies
from litestar.exceptions import ClientException
from litestar.status_codes import HTTP_400_BAD_REQUEST
from loguru import logger
from typing_extensions import Never
from yarl import URL

from config import PG_DSN
from server.model import User


class DisableCookiesJar(AbstractCookieJar):
    """disable cookies on aiohttp client"""

    def clear(self, predicate: ClearCookiePredicate | None = None) -> None:
        return

    def clear_domain(self, domain: str) -> None:
        return

    def update_cookies(self, cookies: LooseCookies, response_url: URL = None) -> None:
        return

    def filter_cookies(self, request_url: URL) -> BaseCookie[str]:
        return BaseCookie()

    def __len__(self):
        return 0

    def __iter__(self):
        yield from ()


http_client = aiohttp.ClientSession(cookie_jar=DisableCookiesJar())
pg = asyncpg.create_pool(dsn=PG_DSN)


async def pg_pool_startup(*args, **kwargs):
    logger.info("init")
    await pg


Request = litestar.Request[Never, User | None, Any]

AuthorizedRequest = litestar.Request[Never, User, Any]


class BadRequestException(ClientException):
    """Server knows the request method, but the target resource doesn't support this method."""

    status_code = HTTP_400_BAD_REQUEST
