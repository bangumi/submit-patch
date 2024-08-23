"""A simple run_migration"""

from typing import NamedTuple

from server.base import pg


class Migrate(NamedTuple):
    version: int
    sql: str


migrations: list[Migrate] = [Migrate(1, "select 1")]


async def run_migration():
    v = await pg.fetchval("select value from patch_db_migration where key=$1", "version")
    if v is None:
        await pg.execute(
            "insert into patch_db_migration (key, value) VALUES ($1,$2)", "version", "0"
        )
        return

    current_version = int(v)
    for migrate in migrations:
        if migrate.version < current_version:
            continue
        await pg.execute(migrate.sql)
        await pg.execute(
            "update patch_db_migration set value = $1 where key = 'version'", str(migrate.version)
        )
