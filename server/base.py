from typing import Any

import aiohttp
import asyncpg
import litestar
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
