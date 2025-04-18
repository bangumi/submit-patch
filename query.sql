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
select *
from subject_patch
where deleted_at is null
  and state = any (@state::int[])
order by created_at desc
limit @size::int8 offset @skip::int8;

-- name: CountSubjectPatchesByStates :one
select count(1)
from subject_patch
where deleted_at is null
  and state = any ($1::int[]);
