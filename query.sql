-- name: GetEpisodePatch :one
SELECT *
FROM episode_patch
WHERE id = $1
LIMIT 1;


-- name: UpsertUser :exec
insert into patch_users (user_id, username, nickname)
VALUES ($1, $2, $3)
on conflict (user_id) do update set username = excluded.username,
                                    nickname = excluded.nickname;


-- name: ListSubjectPatchesByStates :many
select subject_patch.*,
       author.user_id    as author_user_id,
       author.username   as author_username,
       author.nickname   as author_nickname,
       reviewer.user_id  as reviewer_user_id,
       reviewer.username as reviewer_username,
       reviewer.nickname as reviewer_nickname
from subject_patch
         inner join patch_users as author on author.user_id = subject_patch.from_user_id
         left outer join patch_users as reviewer on reviewer.user_id = subject_patch.wiki_user_id
where deleted_at is null
  and state = any (@state::int[])
order by created_at desc
limit @size::int8 offset @skip::int8;

-- name: CountSubjectPatchesByStates :one
select count(1)
from subject_patch
where deleted_at is null
  and state = any ($1::int[]);


-- name: GetSubjectPatchByID :one
select *
from subject_patch
where deleted_at is null
  and id = $1
limit 1;

-- name: GetSubjectPatchByIDForUpdate :one
select *
from subject_patch
where deleted_at is null
  and id = $1
limit 1 for update;


-- name: GetUserByID :one
select *
from patch_users
where user_id = $1;


-- name: GetComments :many
select edit_suggestion.*, author.*
from edit_suggestion
         left join patch_users as author on author.user_id = edit_suggestion.from_user
where deleted_at is null
  and patch_id = $1
  and patch_type = $2
  and deleted_at is null
order by created_at;

-- name: CreateComment :exec
insert into edit_suggestion (id,
                             patch_id,
                             patch_type,
                             text,
                             from_user,
                             created_at,
                             deleted_at)
values ($1, $2, $3, $4, $5, current_timestamp, null);


-- name: RejectSubjectPatch :exec
update subject_patch
set wiki_user_id = $1,
    state        = $2,
    reject_reason = $3,
    updated_at   = current_timestamp
where id = $4
  and deleted_at is null
  and state = 0;

-- name: AcceptSubjectPatch :exec
update subject_patch
set wiki_user_id = $1,
    state        = $2,
    updated_at   = current_timestamp
where id = $3
  and deleted_at is null
  and state = 0;


-- name: UpdateSubjectPatchCommentCount :exec
update subject_patch
set comments_count = (select count(1)
                      from edit_suggestion
                      where patch_type = 'subject'
                        and patch_id = $1
                        and edit_suggestion.from_user != 0)
where id = $1
  and deleted_at is null;


-- name: CreateSubjectEditPatch :exec
INSERT INTO subject_patch
(id, subject_id, from_user_id, reason, name, infobox, summary, nsfw,
 original_name, original_infobox, original_summary, subject_type, patch_desc)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13);
