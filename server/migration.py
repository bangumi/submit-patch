"""A simple run_migration"""

from pathlib import Path
from typing import NamedTuple

from asyncpg import UndefinedTableError

from server.base import pg


class Migrate(NamedTuple):
    version: int
    sql: str


async def run_migration() -> None:
    sql_dir = Path(__file__, "../sql").resolve()

    key_migration_version = "version"

    migrations: list[Migrate] = [
        Migrate(1, sql_dir.joinpath("001-init.sql").read_text(encoding="utf8")),
        Migrate(2, "select 1;"),  # noop
        Migrate(3, "alter table patch rename to subject_patch"),
        Migrate(4, sql_dir.joinpath("004-deleted-view.sql").read_text(encoding="utf-8")),
    ]

    v = None
    try:
        v = await pg.fetchval(
            "select value from patch_db_migration where key=$1", key_migration_version
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
            key_migration_version,
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
            key_migration_version,
        )
