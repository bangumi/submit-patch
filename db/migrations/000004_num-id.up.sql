alter table subject_patch
    add column if not exists num_id bigserial not null;
alter table episode_patch
    add column if not exists num_id bigserial not null;

create index idx_subject_patch_num_id on subject_patch (num_id);
create index idx_episode_patch_num_id on episode_patch (num_id);
