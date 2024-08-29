drop index idx_edit_patch_lookup;
create index idx_edit_patch_lookup on edit_suggestion (created_at, patch_id, patch_type);

create index idx_subject_patch_list on subject_patch (created_at, state, deleted_at);
create index idx_subject_patch_list2 on subject_patch (updated_at, state, deleted_at);

create index idx_episode_patch_list on episode_patch (created_at, state, deleted_at);
create index idx_episode_patch_list2 on episode_patch (updated_at, state, deleted_at);

create index idx_subject_count on subject_patch (state, deleted_at);
create index idx_episode_count on episode_patch (state, deleted_at);

create index idx_subject_subject_id on subject_patch (subject_id, state);

create index idx_episode_subject_id on episode_patch (subject_id, state);
create index idx_episode_episode_id on episode_patch (episode_id, state);
create index idx_episode_deleted_at on episode_patch (deleted_at);

alter index idx_deleted_at rename to idx_subject_deleted_at;
