alter table subject_patch
add column if not exists meta_tags text[];

alter table subject_patch
add column if not exists original_meta_tags text[];
