create table character_patch
(
    id                uuid                                                   not null
        primary key,
    character_id      integer                                                not null,
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
    created_at        timestamp with time zone default CURRENT_TIMESTAMP     not null,
    updated_at        timestamp with time zone default CURRENT_TIMESTAMP     not null,
    deleted_at        timestamp with time zone,
    reject_reason     text                     default ''::character varying not null,
    comments_count    integer                  default 0                     not null,
    patch_desc        text                     default ''::text              not null,
    action            integer                  default 1,
    num_id            bigserial                                              not null
);

comment on column character_patch.action is '1 for update 2 for create';

create index idx_character_id
    on character_patch (character_id);

create index idx_character_patch_list
    on character_patch (created_at, state, deleted_at);

create index idx_character_patch_list2
    on character_patch (updated_at, state, deleted_at);

create index idx_character_patch_list3
    on character_patch (deleted_at, state, created_at);

create index idx_character_count
    on character_patch (state, deleted_at);

create index idx_character_character_id
    on character_patch (character_id, state);

create index idx_character_patch_num_id
    on character_patch (num_id);

create table person_patch
(
    id                uuid                                                   not null
        primary key,
    person_id         integer                                                not null,
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
    created_at        timestamp with time zone default CURRENT_TIMESTAMP     not null,
    updated_at        timestamp with time zone default CURRENT_TIMESTAMP     not null,
    deleted_at        timestamp with time zone,
    reject_reason     text                     default ''::character varying not null,
    comments_count    integer                  default 0                     not null,
    patch_desc        text                     default ''::text              not null,
    action            integer                  default 1,
    num_id            bigserial                                              not null
);

comment on column person_patch.action is '1 for update 2 for create';

create index idx_person_id
    on person_patch (person_id);

create index idx_person_patch_list
    on person_patch (created_at, state, deleted_at);

create index idx_person_patch_list2
    on person_patch (updated_at, state, deleted_at);

create index idx_person_patch_list3
    on person_patch (deleted_at, state, created_at);

create index idx_person_count
    on person_patch (state, deleted_at);

create index idx_person_person_id
    on person_patch (person_id, state);

create index idx_person_patch_num_id
    on person_patch (num_id);