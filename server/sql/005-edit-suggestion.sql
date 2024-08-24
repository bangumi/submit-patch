CREATE TYPE patch_type AS ENUM ('subject', 'episode');

create table if not exists edit_suggestion
(
    id         uuid primary key not null,
    patch_id   uuid             not null,
    patch_type patch_type       not null,
    text       text             not null,
    from_user  int              not null
);

create index idx_edit_patch_lookup on edit_suggestion (patch_id, patch_type);
