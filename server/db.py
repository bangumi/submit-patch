import asyncpg

from server.base import pg


async def fetch_users(rows: list[asyncpg.Record]) -> dict[int, asyncpg.Record]:
    user_id = {x["from_user_id"] for x in rows} | {x["wiki_user_id"] for x in rows}
    user_id.discard(None)
    user_id.discard(0)

    users = {
        x["user_id"]: x
        for x in await pg.fetch("select * from patch_users where user_id = any($1)", user_id)
    }

    return users
