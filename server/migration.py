"""A simple run_migration"""

from pathlib import Path
from typing import NamedTuple

from asyncpg import UndefinedTableError

from server.base import pg


sql_dir = Path(__file__, "../sql").resolve()

KEY_MIGRATION_VERSION = "version"


class Migrate(NamedTuple):
    version: int
    sql: str


migrations: list[Migrate] = [
    Migrate(1, sql_dir.joinpath("001-init.sql").read_text(encoding="utf8")),
    Migrate(2, "select 1;"),  # noop
]


async def run_migration() -> None:
    v = None
    try:
        v = await pg.fetchval(
            "select value from patch_db_migration where key=$1", KEY_MIGRATION_VERSION
        )
    except UndefinedTableError:
        # do init
        await pg.execute(
            """
            create table if not exists patch_db_migration(
                key text primary key not null,
                value text not null
            )
            """
        )

    if v is None:
        await pg.execute(
            "insert into patch_db_migration (key, value) VALUES ($1, $2)",
            KEY_MIGRATION_VERSION,
            "0",
        )
        v = "0"

    current_version = int(v)
    for migrate in migrations:
        if migrate.version <= current_version:
            continue
        await pg.execute(migrate.sql)
        await pg.execute(
            "update patch_db_migration set value = $1 where key = $2",
            str(migrate.version),
            KEY_MIGRATION_VERSION,
        )
