create view view_episode_patch AS
select *
from episode_patch
where deleted_at IS NULL;

create view view_subject_patch AS
select *
from subject_patch
where deleted_at IS NULL;
