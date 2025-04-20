drop view if exists view_episode_patch;
drop view if exists view_subject_patch;

alter table subject_patch
    alter column reject_reason type text;


alter table episode_patch
    alter column reject_reason type text;
