create table patch_users
(
    user_id  integer      not null primary key,
    username varchar(255) not null,
    nickname varchar(255) not null
);


create table subject_patch
(
    id               uuid primary key                                       not null,
    subject_id       integer                                                not null,
    state            integer                  default 0                     not null,
    from_user_id     integer                                                not null,
    wiki_user_id     integer                  default 0                     not null,
    reason           text                                                   not null,
    name             text,
    original_name    text                     default ''::text              not null,
    infobox          text,
    original_infobox text,
    summary          text,
    original_summary text,
    nsfw             boolean,
    created_at       timestamp with time zone default CURRENT_TIMESTAMP     not null,
    updated_at       timestamp with time zone default CURRENT_TIMESTAMP     not null,
    deleted_at       timestamp with time zone,
    reject_reason    varchar(255)             default ''::character varying not null,
    subject_type     bigint                   default 0                     not null
);

create index idx_subject_id on subject_patch (subject_id);

create index idx_deleted_at on subject_patch (deleted_at);

create table episode_patch
(
    id                   uuid primary key                                       not null,
    episode_id           integer                                                not null,
    state                integer                  default 0                     not null,
    from_user_id         integer                                                not null,
    wiki_user_id         integer                  default 0                     not null,
    reason               text                                                   not null,
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
    created_at           timestamp with time zone default CURRENT_TIMESTAMP     not null,
    updated_at           timestamp with time zone default CURRENT_TIMESTAMP     not null,
    deleted_at           timestamp with time zone,
    reject_reason        varchar(255)             default ''::character varying not null
);

create index episode_patch_state_idx
    on episode_patch (state);
