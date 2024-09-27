"""A simple run_migration"""

import itertools
from pathlib import Path
from typing import NamedTuple

from asyncpg import UndefinedTableError

from server.base import pg


class Migrate(NamedTuple):
    version: int
    sql: str


async def run_migration() -> None:
    """Remember to fresh view after alter any table definition"""
    sql_dir = Path(__file__, "../sql").resolve()

    key_migration_version = "version"

    def load_sql(fn: str) -> str:
        return sql_dir.joinpath(fn).read_text(encoding="utf8")

    drop_view_sql = load_sql("000-drop-view.sql")
    create_or_update_view_sql = load_sql("004-deleted-view.sql")

    migrations: list[Migrate] = [
        Migrate(1, load_sql("001-init.sql")),
        Migrate(4, create_or_update_view_sql),
        Migrate(5, load_sql("005-edit-suggestion.sql")),
        Migrate(
            6,
            "alter table episode_patch add column subject_id int not null default 0;",
        ),
        Migrate(7, create_or_update_view_sql),
        Migrate(8, load_sql("008-create-index.sql")),
        Migrate(9, load_sql("009-show-suggestion-count.sql")),
        Migrate(10, create_or_update_view_sql),
        Migrate(11, load_sql("011-extra-description.sql")),
        Migrate(12, load_sql("012-show-ep-num.sql")),
        Migrate(13, load_sql("013-add-subject-platform.sql")),
    ]

    if not all(x <= y for x, y in itertools.pairwise(migrations)):
        raise Exception("migration list is not sorted")

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
        await pg.execute(drop_view_sql)
        await pg.execute(migrate.sql)
        await pg.execute(create_or_update_view_sql)
        await pg.execute(
            "update patch_db_migration set value = $1 where key = $2",
            str(migrate.version),
            key_migration_version,
        )
