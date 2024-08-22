create table episode_patch
(
    id                   uuid                                               not null
        primary key,
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
