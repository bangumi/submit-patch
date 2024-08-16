create table if not exists patch
(
    id               uuid PRIMARY KEY     DEFAULT gen_random_uuid(),
    subject_id       int         not null,
    -- 0 pending
    -- 1 approved
    -- 2 rejected
    -- 3 out-dated
    state            int         not null default 0,
    from_user_id     int         not null,           -- from user XXX
    wiki_user_id     int         not null default 0, -- approved/rejected by user XXX
    -- why we should make this change
    description      text        not null,
    name             TEXT                 default null,
    original_name    TEXT                 default null,
    infobox          TEXT                 default null,
    original_infobox TEXT                 default null,
    summary          TEXT                 default null,
    original_summary TEXT                 default null,
    nsfw             bool                 default null,
    created_at       timestamptz not null default current_timestamp,
    updated_at       timestamptz not null default current_timestamp,
    deleted_at       timestamptz          default null
);

create index idx_subject_id on patch (subject_id);

create index idx_deleted_at on patch (deleted_at);
