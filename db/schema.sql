create table public.patch_users
(
    user_id  integer      not null
        primary key,
    username varchar(255) not null,
    nickname varchar(255) not null
);

alter table public.patch_users
    owner to postgres;

create table public.subject_patch
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
    action            integer                  default 1,
    num_id            bigserial
);

comment on column public.subject_patch.action is '1 for update 2 for create';

alter table public.subject_patch
    owner to postgres;

create index idx_subject_id
    on public.subject_patch (subject_id);

create index idx_subject_patch_list
    on public.subject_patch (created_at, state, deleted_at);

create index idx_subject_patch_list2
    on public.subject_patch (updated_at, state, deleted_at);

create index idx_subject_patch_list3
    on public.subject_patch (deleted_at, state, created_at);

create index idx_subject_count
    on public.subject_patch (state, deleted_at);

create index idx_subject_subject_id
    on public.subject_patch (subject_id, state);

create index idx_subject_patch_num_id
    on public.subject_patch (num_id);

create table public.episode_patch
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
    ep                   integer,
    num_id               bigserial
);

alter table public.episode_patch
    owner to postgres;

create index episode_patch_state_idx
    on public.episode_patch (state);

create index idx_episode_patch_list
    on public.episode_patch (created_at, state, deleted_at);

create index idx_episode_patch_list2
    on public.episode_patch (updated_at, state, deleted_at);

create index idx_episode_patch_list3
    on public.episode_patch (deleted_at, state, created_at);

create index idx_episode_count
    on public.episode_patch (state, deleted_at);

create index idx_episode_subject_id
    on public.episode_patch (subject_id, state);

create index idx_episode_episode_id
    on public.episode_patch (episode_id, state);

create index idx_episode_patch_num_id
    on public.episode_patch (num_id);

create table public.edit_suggestion
(
    id         uuid                                               not null
        primary key,
    patch_id   uuid                                               not null,
    patch_type varchar(64)                                        not null,
    text       text                                               not null,
    from_user  integer                                            not null,
    created_at timestamp with time zone default CURRENT_TIMESTAMP not null,
    deleted_at timestamp with time zone
);

alter table public.edit_suggestion
    owner to postgres;

create index idx_edit_patch_lookup
    on public.edit_suggestion (created_at, patch_id, patch_type);

create table public.patch_tables_migrations
(
    version bigint  not null
        primary key,
    dirty   boolean not null
);

alter table public.patch_tables_migrations
    owner to postgres;
