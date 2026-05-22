alter table subject_patch
drop column if exists meta_tags;

alter table subject_patch
drop column if exists original_meta_tags;
