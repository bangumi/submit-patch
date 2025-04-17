-- name: GetEpisodePatch :one
SELECT *
FROM episode_patch
WHERE id = $1 LIMIT 1;


-- name: UpsertUser :exec
insert into patch_users (user_id, username, nickname)
VALUES ($1, $2, $3) on conflict (user_id) do
update set
    username = excluded.username,
    nickname = excluded.nickname;
