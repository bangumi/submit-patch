create type patch_type as enum ('subject', 'episode');

create table patch_users
(
    user_id  integer      not null
        primary key,
    username varchar(255) not null,
    nickname varchar(255) not null
);


create table subject_patch
(
    id                uuid                                                   not null
        primary key,
    subject_id        integer                                                not null,
    state             integer                  default 0                     not null,
    from_user_id      integer                                                not null,
    wiki_user_id      integer                  default 0                     not null,
    reason            text                                                   not null,
    name              text,
    original_name     text                     default ''::text              not null,
    infobox           text,
    original_infobox  text,
    summary           text,
    original_summary  text,
    nsfw              boolean,
    created_at        timestamp with time zone default CURRENT_TIMESTAMP     not null,
    updated_at        timestamp with time zone default CURRENT_TIMESTAMP     not null,
    deleted_at        timestamp with time zone,
    reject_reason     text                     default ''::character varying not null,
    subject_type      bigint                   default 0                     not null,
    comments_count    integer                  default 0                     not null,
    patch_desc        text                     default ''::text              not null,
    original_platform integer,
    platform          integer,
    action            integer                  default 1
);

comment on column subject_patch.action is '1 for update 2 for create';

create index idx_subject_id
    on subject_patch (subject_id);

create index idx_subject_deleted_at
    on subject_patch (deleted_at);

create index idx_subject_patch_list
    on subject_patch (created_at, state, deleted_at);

create index idx_subject_patch_list2
    on subject_patch (updated_at, state, deleted_at);

create index idx_subject_count
    on subject_patch (state, deleted_at);

create index idx_subject_subject_id
    on subject_patch (subject_id, state);

create table episode_patch
(
    id                   uuid                                                   not null
        primary key,
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
    reject_reason        text                     default ''::character varying not null,
    subject_id           integer                  default 0                     not null,
    comments_count       integer                  default 0                     not null,
    patch_desc           text                     default ''::text              not null,
    ep                   integer
);

create index episode_patch_state_idx
    on episode_patch (state);

create index idx_episode_patch_list
    on episode_patch (created_at, state, deleted_at);

create index idx_episode_patch_list2
    on episode_patch (updated_at, state, deleted_at);

create index idx_episode_count
    on episode_patch (state, deleted_at);

create index idx_episode_subject_id
    on episode_patch (subject_id, state);

create index idx_episode_episode_id
    on episode_patch (episode_id, state);

create index idx_episode_deleted_at
    on episode_patch (deleted_at);

create table edit_suggestion
(
    id         uuid                                               not null
        primary key,
    patch_id   uuid                                               not null,
    patch_type patch_type                                         not null,
    text       text                                               not null,
    from_user  integer                                            not null,
    created_at timestamp with time zone default CURRENT_TIMESTAMP not null,
    deleted_at timestamp with time zone
);

create index idx_edit_patch_lookup
    on edit_suggestion (created_at, patch_id, patch_type);
