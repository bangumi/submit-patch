-- name: UpsertUser :exec
insert into patch_users (user_id, username, nickname)
VALUES ($1, $2, $3)
on conflict (user_id) do update set username = excluded.username,
                                    nickname = excluded.nickname;


-- name: CountSubjectPatches :one
select count(1)
from subject_patch
where deleted_at is null
  and state = any (@state::int[])
  and ((from_user_id = @from_user_id and @from_user_id != 0) or @from_user_id = 0)
  and ((wiki_user_id = @wiki_user_id and @wiki_user_id != 0) or @wiki_user_id = 0)
  and action = 1;

-- name: ListSubjectPatches :many
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
  and ((from_user_id = @from_user_id and @from_user_id != 0) or @from_user_id = 0)
  and ((wiki_user_id = @wiki_user_id and @wiki_user_id != 0) or @wiki_user_id = 0)
  and action = 1
order by case when @order_by::text = 'created_at' then created_at end desc,
         case when @order_by = 'updated_at' then updated_at end desc,
         case when @order_by = '' then created_at end desc
limit @size offset @skip;


-- name: CountEpisodePatches :one
select count(1)
from episode_patch
where deleted_at is null
  and ((from_user_id = @from_user_id and @from_user_id != 0) or @from_user_id = 0)
  and ((wiki_user_id = @wiki_user_id and @wiki_user_id != 0) or @wiki_user_id = 0)
  and state = any (@state::int[]);

-- name: ListEpisodePatches :many
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
  and ((from_user_id = @from_user_id and @from_user_id != 0) or @from_user_id = 0)
  and ((wiki_user_id = @wiki_user_id and @wiki_user_id != 0) or @wiki_user_id = 0)
  and state = any (@state::int[])
order by case when @order_by::text = 'created_at' then created_at end desc,
         case when @order_by = 'updated_at' then updated_at end desc,
         case when @order_by = '' then created_at end desc
limit @size offset @skip;


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

-- name: CountPendingSubjectPatch :one
select count(1)
from subject_patch
where deleted_at is null
  and state = 0;

-- name: CountPendingEpisodePatch :one
select count(1)
from episode_patch
where deleted_at is null
  and state = 0;

-- name: CountPendingCharacterPatch :one
select count(1)
from character_patch
where deleted_at is null
  and state = 0;

-- name: CountPendingPersonPatch :one
select count(1)
from person_patch
where deleted_at is null
  and state = 0;

-- name: ListPendingEpisodePatches :many
select id, episode_id, created_at, updated_at, from_user_id
from episode_patch
where state = 0
  and deleted_at is null;

-- name: ListPendingSubjectPatches :many
select id, subject_id, created_at, updated_at, from_user_id
from subject_patch
where state = 0
  and deleted_at is null;


-- name: NextPendingSubjectPatch :one
select id
from subject_patch
where state = 0
  and deleted_at is null
  and id < $1
order by id desc
limit 1;

-- name: NextPendingEpisodePatch :one
select id
from episode_patch
where state = 0
  and deleted_at is null
  and id < $1
order by id desc
limit 1;

-- name: CountCharacterPatches :one
select count(1)
from character_patch
where deleted_at is null
  and state = any (@state::int[])
  and ((from_user_id = @from_user_id and @from_user_id != 0) or @from_user_id = 0)
  and ((wiki_user_id = @wiki_user_id and @wiki_user_id != 0) or @wiki_user_id = 0)
  and action = 1;

-- name: ListCharacterPatches :many
select character_patch.id,
       character_patch.original_name,
       character_patch.state,
       character_patch.action,
       character_patch.created_at,
       character_patch.updated_at,
       character_patch.comments_count,
       character_patch.reason,
       author.user_id    as author_user_id,
       author.username   as author_username,
       author.nickname   as author_nickname,
       reviewer.user_id  as reviewer_user_id,
       reviewer.username as reviewer_username,
       reviewer.nickname as reviewer_nickname
from character_patch
         inner join patch_users as author on author.user_id = character_patch.from_user_id
         left outer join patch_users as reviewer on reviewer.user_id = character_patch.wiki_user_id
where deleted_at is null
  and state = any (@state::int[])
  and ((from_user_id = @from_user_id and @from_user_id != 0) or @from_user_id = 0)
  and ((wiki_user_id = @wiki_user_id and @wiki_user_id != 0) or @wiki_user_id = 0)
  and action = 1
order by case when @order_by::text = 'created_at' then created_at end desc,
         case when @order_by = 'updated_at' then updated_at end desc,
         case when @order_by = '' then created_at end desc
limit @size offset @skip;

-- name: CountPersonPatches :one
select count(1)
from person_patch
where deleted_at is null
  and state = any (@state::int[])
  and ((from_user_id = @from_user_id and @from_user_id != 0) or @from_user_id = 0)
  and ((wiki_user_id = @wiki_user_id and @wiki_user_id != 0) or @wiki_user_id = 0)
  and action = 1;

-- name: ListPersonPatches :many
select person_patch.id,
       person_patch.original_name,
       person_patch.state,
       person_patch.action,
       person_patch.created_at,
       person_patch.updated_at,
       person_patch.comments_count,
       person_patch.reason,
       author.user_id    as author_user_id,
       author.username   as author_username,
       author.nickname   as author_nickname,
       reviewer.user_id  as reviewer_user_id,
       reviewer.username as reviewer_username,
       reviewer.nickname as reviewer_nickname
from person_patch
         inner join patch_users as author on author.user_id = person_patch.from_user_id
         left outer join patch_users as reviewer on reviewer.user_id = person_patch.wiki_user_id
where deleted_at is null
  and state = any (@state::int[])
  and ((from_user_id = @from_user_id and @from_user_id != 0) or @from_user_id = 0)
  and ((wiki_user_id = @wiki_user_id and @wiki_user_id != 0) or @wiki_user_id = 0)
  and action = 1
order by case when @order_by::text = 'created_at' then created_at end desc,
         case when @order_by = 'updated_at' then updated_at end desc,
         case when @order_by = '' then created_at end desc
limit @size offset @skip;

-- name: GetCharacterPatchByID :one
select *
from character_patch
where deleted_at is null
  and id = $1
limit 1;

-- name: GetCharacterPatchByIDForUpdate :one
select *
from character_patch
where deleted_at is null
  and id = $1
limit 1 for update;

-- name: GetPersonPatchByID :one
select *
from person_patch
where deleted_at is null
  and id = $1
limit 1;

-- name: GetPersonPatchByIDForUpdate :one
select *
from person_patch
where deleted_at is null
  and id = $1
limit 1 for update;

-- name: CreateCharacterEditPatch :exec
INSERT INTO character_patch (id,
                             character_id, from_user_id, reason, name, infobox,
                             summary,
                             original_name, original_infobox,
                             original_summary, patch_desc)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);

-- name: CreatePersonEditPatch :exec
INSERT INTO person_patch (id,
                           person_id, from_user_id, reason, name, infobox,
                           summary,
                           original_name, original_infobox,
                           original_summary, patch_desc)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);

-- name: UpdateCharacterPatch :exec
update character_patch
set original_name    = $2,
    name             = $3,
    original_infobox = $4,
    infobox          = $5,
    original_summary = $6,
    summary          = $7,
    reason           = $8,
    patch_desc       = $9,
    updated_at       = current_timestamp
where id = $1;

-- name: UpdatePersonPatch :exec
update person_patch
set original_name    = $2,
    name             = $3,
    original_infobox = $4,
    infobox          = $5,
    original_summary = $6,
    summary          = $7,
    reason           = $8,
    patch_desc       = $9,
    updated_at       = current_timestamp
where id = $1;

-- name: RejectCharacterPatch :exec
update character_patch
set wiki_user_id  = $1,
    state         = $2,
    reject_reason = $3,
    updated_at    = current_timestamp
where id = $4
  and deleted_at is null
  and state = 0;

-- name: RejectPersonPatch :exec
update person_patch
set wiki_user_id  = $1,
    state         = $2,
    reject_reason = $3,
    updated_at    = current_timestamp
where id = $4
  and deleted_at is null
  and state = 0;

-- name: AcceptCharacterPatch :exec
update character_patch
set wiki_user_id = $1,
    state        = $2,
    updated_at   = current_timestamp
where id = $3
  and deleted_at is null
  and state = 0;

-- name: AcceptPersonPatch :exec
update person_patch
set wiki_user_id = $1,
    state        = $2,
    updated_at   = current_timestamp
where id = $3
  and deleted_at is null
  and state = 0;

-- name: UpdateCharacterPatchCommentCount :exec
update character_patch
set comments_count = (select count(1)
                      from edit_suggestion
                      where patch_type = 'character'
                        and patch_id = $1
                        and edit_suggestion.from_user != 0)
where id = $1
  and deleted_at is null;

-- name: UpdatePersonPatchCommentCount :exec
update person_patch
set comments_count = (select count(1)
                      from edit_suggestion
                      where patch_type = 'person'
                        and patch_id = $1
                        and edit_suggestion.from_user != 0)
where id = $1
  and deleted_at is null;

-- name: DeleteCharacterPatch :exec
update character_patch
set deleted_at = current_timestamp
where id = $1
  and deleted_at is null;

-- name: DeletePersonPatch :exec
update person_patch
set deleted_at = current_timestamp
where id = $1
  and deleted_at is null;

-- name: ListPendingCharacterPatches :many
select id, character_id, created_at, updated_at, from_user_id
from character_patch
where state = 0
  and deleted_at is null;

-- name: ListPendingPersonPatches :many
select id, person_id, created_at, updated_at, from_user_id
from person_patch
where state = 0
  and deleted_at is null;

-- name: NextPendingCharacterPatch :one
select id
from character_patch
where state = 0
  and deleted_at is null
  and id < $1
order by id desc
limit 1;

-- name: NextPendingPersonPatch :one
select id
from person_patch
where state = 0
  and deleted_at is null
  and id < $1
order by id desc
limit 1;

-- name: GetPendingSubjectPatchesBySubjectID :many
select id, from_user_id, num_id, name, original_name, infobox, original_infobox, summary, original_summary
from subject_patch
where subject_id = $1
  and state = 0
  and deleted_at is null;

-- name: GetPendingCharacterPatchesByCharacterID :many
select id, from_user_id, num_id, name, original_name, infobox, original_infobox, summary, original_summary
from character_patch
where character_id = $1
  and state = 0
  and deleted_at is null;

-- name: GetPendingPersonPatchesByPersonID :many
select id, from_user_id, num_id, name, original_name, infobox, original_infobox, summary, original_summary
from person_patch
where person_id = $1
  and state = 0
  and deleted_at is null;
