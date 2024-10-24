from __future__ import annotations

from typing import Any
from uuid import UUID

import asyncpg
from asyncpg.pool import PoolConnectionProxy

from server.base import pg
from server.model import PatchType


async def fetch_users(rows: list[asyncpg.Record]) -> dict[int, asyncpg.Record]:
    user_id = {x["from_user_id"] for x in rows} | {x["wiki_user_id"] for x in rows}
    user_id.discard(None)
    user_id.discard(0)

    users = {
        x["user_id"]: x
        for x in await pg.fetch("select * from patch_users where user_id = any($1)", user_id)
    }

    return users


async def create_edit_suggestion(
    conn: asyncpg.Connection[Any] | PoolConnectionProxy[Any] | asyncpg.Pool[Any],
    patch_id: UUID,
    type: PatchType,
    text: str,
    from_user: int,
) -> None:
    await conn.execute(
        """
            insert into edit_suggestion (id, patch_id, patch_type, text, from_user)
            VALUES (uuid_generate_v7(), $1, $2, $3, $4)
        """,
        patch_id,
        type,
        text,
        from_user,
    )
