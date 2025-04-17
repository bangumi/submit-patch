-- name: GetEpisodePatch :one
SELECT * FROM episode_patch WHERE id = $1 LIMIT 1;
