"""A simple run_migration"""

from typing import NamedTuple

from server.base import pg


class Migrate(NamedTuple):
    version: int
    sql: str


migrations: list[Migrate] = [
    Migrate(1, "select 1"),
    Migrate(
        2,
        """
create table episode_patch
(
    id                   uuid primary key                                   not null,
    episode_id           integer                                            not null,
    state                integer                  default 0                 not null,
    from_user_id         integer                                            not null,
    wiki_user_id         integer                  default 0                 not null,
    reason               text                                               not null,

    original_name        text,
    name                 text,

    original_name_cn     text,
    name_cn              text,

    original_duration    varchar(255),
    duration             varchar(255),

    original_airdate     varchar(64),
    airdate              varchar(64),

    original_description text,
    description          text,

    created_at           timestamp with time zone default CURRENT_TIMESTAMP not null,
    updated_at           timestamp with time zone default CURRENT_TIMESTAMP not null,
    deleted_at           timestamp with time zone,

    reject_reason        varchar(255)             default ''                not null
);


create index on episode_patch (state);
""",
    ),
]


async def run_migration() -> None:
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
