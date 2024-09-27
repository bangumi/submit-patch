alter table subject_patch
    add column original_platform int default null;

alter table subject_patch
    add column platform int default null;


alter table subject_patch
    add column action int default 1;

COMMENT on column subject_patch.action IS '1 for update 2 for create';
