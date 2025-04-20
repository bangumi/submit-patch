-- name: UpsertUser :exec
insert into patch_users (user_id, username, nickname)
VALUES ($1, $2, $3)
on conflict (user_id) do update set username = excluded.username,
                                    nickname = excluded.nickname;


-- name: ListSubjectPatchesByStates :many
select subject_patch.id,
       subject_patch.original_name,
       subject_patch.state,
       subject_patch.action,
       subject_patch.created_at,
       subject_patch.updated_at,
       subject_patch.comments_count,
       subject_patch.reason,
       subject_patch.subject_type,
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
  and action = 1
order by created_at desc
limit @size::int8 offset @skip::int8;

-- name: CountSubjectPatchesByStates :one
select count(1)
from subject_patch
where deleted_at is null
  and state = any ($1::int[])
  and action = 1;

-- name: ListEpisodePatchesByStates :many
select episode_patch.id,
       episode_patch.original_name,
       episode_patch.state,
       episode_patch.created_at,
       episode_patch.updated_at,
       episode_patch.comments_count,
       episode_patch.reason,
       author.user_id    as author_user_id,
       author.username   as author_username,
       author.nickname   as author_nickname,
       reviewer.user_id  as reviewer_user_id,
       reviewer.username as reviewer_username,
       reviewer.nickname as reviewer_nickname
from episode_patch
         inner join patch_users as author on author.user_id = episode_patch.from_user_id
         left outer join patch_users as reviewer on reviewer.user_id = episode_patch.wiki_user_id
where deleted_at is null
  and state = any (@state::int[])
order by created_at desc
limit @size::int8 offset @skip::int8;

-- name: CountEpisodePatchesByStates :one
select count(1)
from episode_patch
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

-- name: GetEpisodePatchByIDForUpdate :one
select *
from episode_patch
where deleted_at is null
  and id = $1
limit 1 for update;


-- name: GetEpisodePatchByID :one
select *
from episode_patch
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
set wiki_user_id  = $1,
    state         = $2,
    reject_reason = $3,
    updated_at    = current_timestamp
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

-- name: RejectEpisodePatch :exec
update episode_patch
set wiki_user_id  = $1,
    state         = $2,
    reject_reason = $3,
    updated_at    = current_timestamp
where id = $4
  and deleted_at is null
  and state = 0;

-- name: UpdateEpisodePatchCommentCount :exec
update episode_patch
set comments_count = (select count(1)
                      from edit_suggestion
                      where patch_type = 'episode'
                        and patch_id = $1
                        and edit_suggestion.from_user != 0)
where id = $1
  and deleted_at is null;

-- name: CreateSubjectEditPatch :exec
INSERT INTO subject_patch (id,
                           subject_id, from_user_id, reason, name, infobox,
                           summary, nsfw,
                           original_name, original_infobox,
                           original_summary, subject_type, patch_desc)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13);

-- name: CreateEpisodePatch :exec
insert into episode_patch (created_at, updated_at, id, episode_id, state, from_user_id,
                           wiki_user_id, reason,
                           original_name, name,
                           original_name_cn, name_cn,
                           original_duration, duration,
                           original_airdate, airdate,
                           original_description, description,
                           patch_desc, ep)
values (current_timestamp, current_timestamp,
        $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12,
        $13, $14, $15, $16, $17, $18);


-- name: UpdateSubjectPatch :exec
update subject_patch
set original_name    = $2,
    name             = $3,
    original_infobox = $4,
    infobox          = $5,
    original_summary = $6,
    summary          = $7,
    nsfw             = $8,
    reason           = $9,
    patch_desc       = $10,
    updated_at       = current_timestamp
where id = $1;

-- name: UpdateEpisodePatch :exec
update episode_patch
set reason               = $2,
    patch_desc           = $3,
    updated_at           = current_timestamp,
    original_name        = $4,
    name                 = $5,
    original_name_cn     = $6,
    name_cn              = $7,
    original_duration    = $8,
    duration             = $9,
    original_airdate     = $10,
    airdate              = $11,
    original_description = $12,
    description          = $13
where id = $1;


-- name: AcceptEpisodePatch :exec
update episode_patch
set wiki_user_id = $1,
    state        = $2,
    updated_at   = current_timestamp
where id = $3
  and deleted_at is null
  and state = 0;

-- name: DeleteSubjectPatch :exec
update subject_patch
set deleted_at = current_timestamp
where id = $1
  and deleted_at is null;

-- name: DeleteEpisodePatch :exec
update episode_patch
set deleted_at = current_timestamp
where id = $1
  and deleted_at is null;

-- name: TestDelete :exec
delete
from edit_suggestion
where id = $1;


-- name: CountPendingPatch :one
select (select count(1)
        from subject_patch
        where deleted_at is null
          and state = 0) as subject_patch_count,
       (select count(1)
        from episode_patch
        where deleted_at is null
          and state = 0) as episode_patch_count;

-- name: ListSubjectPatchesByStatesFromUser :many
select subject_patch.id,
       subject_patch.original_name,
       subject_patch.state,
       subject_patch.action,
       subject_patch.created_at,
       subject_patch.updated_at,
       subject_patch.comments_count,
       subject_patch.reason,
       subject_patch.subject_type,
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
  and from_user_id = $1
  and action = 1
order by created_at desc
limit @size::int8 offset @skip::int8;

-- name: CountSubjectPatchesByStatesFromUser :one
select count(1)
from subject_patch
where deleted_at is null
  and from_user_id = @user_id
  and state = any (@state::int[])
  and action = 1;


-- name: ListEpisodePatchesByStatesFromUser :many
select episode_patch.id,
       episode_patch.original_name,
       episode_patch.state,
       episode_patch.created_at,
       episode_patch.updated_at,
       episode_patch.comments_count,
       episode_patch.reason,
       author.user_id    as author_user_id,
       author.username   as author_username,
       author.nickname   as author_nickname,
       reviewer.user_id  as reviewer_user_id,
       reviewer.username as reviewer_username,
       reviewer.nickname as reviewer_nickname
from episode_patch
         inner join patch_users as author on author.user_id = episode_patch.from_user_id
         left outer join patch_users as reviewer on reviewer.user_id = episode_patch.wiki_user_id
where deleted_at is null
  and from_user_id = @user_id
  and state = any (@state::int[])
order by created_at desc
limit @size::int8 offset @skip::int8;

-- name: CountEpisodePatchesByStatesFromUser :one
select count(1)
from episode_patch
where deleted_at is null
  and from_user_id = @user_id
  and state = any (@state::int[]);


-- name: ListSubjectPatchesByStatesReviewedByUser :many
select subject_patch.id,
       subject_patch.original_name,
       subject_patch.state,
       subject_patch.action,
       subject_patch.created_at,
       subject_patch.updated_at,
       subject_patch.comments_count,
       subject_patch.reason,
       subject_patch.subject_type,
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
  and wiki_user_id = $1
  and action = 1
order by created_at desc
limit @size::int8 offset @skip::int8;

-- name: CountSubjectPatchesByStatesReviewedByUser :one
select count(1)
from subject_patch
where deleted_at is null
  and wiki_user_id = @user_id
  and state = any (@state::int[])
  and action = 1;


-- name: ListEpisodePatchesByStatesReviewedByUser :many
select episode_patch.id,
       episode_patch.original_name,
       episode_patch.state,
       episode_patch.created_at,
       episode_patch.updated_at,
       episode_patch.comments_count,
       episode_patch.reason,
       author.user_id    as author_user_id,
       author.username   as author_username,
       author.nickname   as author_nickname,
       reviewer.user_id  as reviewer_user_id,
       reviewer.username as reviewer_username,
       reviewer.nickname as reviewer_nickname
from episode_patch
         inner join patch_users as author on author.user_id = episode_patch.from_user_id
         left outer join patch_users as reviewer on reviewer.user_id = episode_patch.wiki_user_id
where deleted_at is null
  and wiki_user_id = @user_id
  and state = any (@state::int[])
order by created_at desc
limit @size::int8 offset @skip::int8;

-- name: CountEpisodePatchesByStatesReviewedByUser :one
select count(1)
from episode_patch
where deleted_at is null
  and wiki_user_id = @user_id
  and state = any (@state::int[]);


-- name: ListPendingEpisodePatches :many
select id, episode_id, created_at, updated_at, from_user_id
from episode_patch
where state = 0
  and deleted_at is not null;

-- name: ListPendingSubjectPatches :many
select id, subject_id, created_at, updated_at, from_user_id
from subject_patch
where state = 0
  and deleted_at is not null;
