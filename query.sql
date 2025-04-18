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

-- name: GetUserByID :one
select *
from patch_users
where user_id = $1;


-- name: GetComments :many
select edit_suggestion.*, author.*
from edit_suggestion
         inner join patch_users as author on author.user_id = edit_suggestion.from_user
where deleted_at is null
  and patch_id = $1
  and patch_type = $2
  and deleted_at is null
order by created_at
;
