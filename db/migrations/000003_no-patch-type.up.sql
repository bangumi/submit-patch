alter table edit_suggestion
    alter column patch_type type varchar(64);

drop index if exists idx_episode_deleted_at;

drop index if exists idx_subject_deleted_at;
