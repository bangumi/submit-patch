from typing import Any

import aiohttp
import asyncpg
import litestar
from litestar.exceptions import ClientException
from litestar.status_codes import HTTP_400_BAD_REQUEST
from loguru import logger
from typing_extensions import Never

from config import PG_DSN
from server.model import User


http_client = aiohttp.ClientSession()
pg = asyncpg.create_pool(dsn=PG_DSN)


async def pg_pool_startup(*args, **kwargs):
    logger.info("init")
    await pg


Request = litestar.Request[Never, User | None, Any]

AuthorizedRequest = litestar.Request[Never, User, Any]


class BadRequestException(ClientException):
    """Server knows the request method, but the target resource doesn't support this method."""

    status_code = HTTP_400_BAD_REQUEST
