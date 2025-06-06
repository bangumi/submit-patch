// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.29.0
// source: query.sql

package dal

import (
	"context"

	uuid "github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const acceptEpisodePatch = `-- name: AcceptEpisodePatch :exec
update episode_patch
set wiki_user_id = $1,
    state        = $2,
    updated_at   = current_timestamp
where id = $3
  and deleted_at is null
  and state = 0
`

type AcceptEpisodePatchParams struct {
	WikiUserID int32
	State      int32
	ID         uuid.UUID
}

func (q *Queries) AcceptEpisodePatch(ctx context.Context, arg AcceptEpisodePatchParams) error {
	_, err := q.db.Exec(ctx, acceptEpisodePatch, arg.WikiUserID, arg.State, arg.ID)
	return err
}

const acceptSubjectPatch = `-- name: AcceptSubjectPatch :exec
update subject_patch
set wiki_user_id = $1,
    state        = $2,
    updated_at   = current_timestamp
where id = $3
  and deleted_at is null
  and state = 0
`

type AcceptSubjectPatchParams struct {
	WikiUserID int32
	State      int32
	ID         uuid.UUID
}

func (q *Queries) AcceptSubjectPatch(ctx context.Context, arg AcceptSubjectPatchParams) error {
	_, err := q.db.Exec(ctx, acceptSubjectPatch, arg.WikiUserID, arg.State, arg.ID)
	return err
}

const countEpisodePatches = `-- name: CountEpisodePatches :one
select count(1)
from episode_patch
where deleted_at is null
  and ((from_user_id = $1 and $1 != 0) or $1 = 0)
  and ((wiki_user_id = $2 and $2 != 0) or $2 = 0)
  and state = any ($3::int[])
`

type CountEpisodePatchesParams struct {
	FromUserID int32
	WikiUserID int32
	State      []int32
}

func (q *Queries) CountEpisodePatches(ctx context.Context, arg CountEpisodePatchesParams) (int64, error) {
	row := q.db.QueryRow(ctx, countEpisodePatches, arg.FromUserID, arg.WikiUserID, arg.State)
	var count int64
	err := row.Scan(&count)
	return count, err
}

const countPendingPatch = `-- name: CountPendingPatch :one
select (select count(1)
        from subject_patch
        where deleted_at is null
          and state = 0) as subject_patch_count,
       (select count(1)
        from episode_patch
        where deleted_at is null
          and state = 0) as episode_patch_count
`

type CountPendingPatchRow struct {
	SubjectPatchCount int64
	EpisodePatchCount int64
}

func (q *Queries) CountPendingPatch(ctx context.Context) (CountPendingPatchRow, error) {
	row := q.db.QueryRow(ctx, countPendingPatch)
	var i CountPendingPatchRow
	err := row.Scan(&i.SubjectPatchCount, &i.EpisodePatchCount)
	return i, err
}

const countSubjectPatches = `-- name: CountSubjectPatches :one
select count(1)
from subject_patch
where deleted_at is null
  and state = any ($1::int[])
  and ((from_user_id = $2 and $2 != 0) or $2 = 0)
  and ((wiki_user_id = $3 and $3 != 0) or $3 = 0)
  and action = 1
`

type CountSubjectPatchesParams struct {
	State      []int32
	FromUserID int32
	WikiUserID int32
}

func (q *Queries) CountSubjectPatches(ctx context.Context, arg CountSubjectPatchesParams) (int64, error) {
	row := q.db.QueryRow(ctx, countSubjectPatches, arg.State, arg.FromUserID, arg.WikiUserID)
	var count int64
	err := row.Scan(&count)
	return count, err
}

const createComment = `-- name: CreateComment :exec
insert into edit_suggestion (id,
                             patch_id,
                             patch_type,
                             text,
                             from_user,
                             created_at,
                             deleted_at)
values ($1, $2, $3, $4, $5, current_timestamp, null)
`

type CreateCommentParams struct {
	ID        uuid.UUID
	PatchID   uuid.UUID
	PatchType string
	Text      string
	FromUser  int32
}

func (q *Queries) CreateComment(ctx context.Context, arg CreateCommentParams) error {
	_, err := q.db.Exec(ctx, createComment,
		arg.ID,
		arg.PatchID,
		arg.PatchType,
		arg.Text,
		arg.FromUser,
	)
	return err
}

const createEpisodePatch = `-- name: CreateEpisodePatch :exec
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
        $13, $14, $15, $16, $17, $18)
`

type CreateEpisodePatchParams struct {
	ID                  uuid.UUID
	EpisodeID           int32
	State               int32
	FromUserID          int32
	WikiUserID          int32
	Reason              string
	OriginalName        pgtype.Text
	Name                pgtype.Text
	OriginalNameCn      pgtype.Text
	NameCn              pgtype.Text
	OriginalDuration    pgtype.Text
	Duration            pgtype.Text
	OriginalAirdate     pgtype.Text
	Airdate             pgtype.Text
	OriginalDescription pgtype.Text
	Description         pgtype.Text
	PatchDesc           string
	Ep                  pgtype.Int4
}

func (q *Queries) CreateEpisodePatch(ctx context.Context, arg CreateEpisodePatchParams) error {
	_, err := q.db.Exec(ctx, createEpisodePatch,
		arg.ID,
		arg.EpisodeID,
		arg.State,
		arg.FromUserID,
		arg.WikiUserID,
		arg.Reason,
		arg.OriginalName,
		arg.Name,
		arg.OriginalNameCn,
		arg.NameCn,
		arg.OriginalDuration,
		arg.Duration,
		arg.OriginalAirdate,
		arg.Airdate,
		arg.OriginalDescription,
		arg.Description,
		arg.PatchDesc,
		arg.Ep,
	)
	return err
}

const createSubjectEditPatch = `-- name: CreateSubjectEditPatch :exec
INSERT INTO subject_patch (id,
                           subject_id, from_user_id, reason, name, infobox,
                           summary, nsfw,
                           original_name, original_infobox,
                           original_summary, subject_type, patch_desc)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
`

type CreateSubjectEditPatchParams struct {
	ID              uuid.UUID
	SubjectID       int32
	FromUserID      int32
	Reason          string
	Name            pgtype.Text
	Infobox         pgtype.Text
	Summary         pgtype.Text
	Nsfw            pgtype.Bool
	OriginalName    string
	OriginalInfobox pgtype.Text
	OriginalSummary pgtype.Text
	SubjectType     int64
	PatchDesc       string
}

func (q *Queries) CreateSubjectEditPatch(ctx context.Context, arg CreateSubjectEditPatchParams) error {
	_, err := q.db.Exec(ctx, createSubjectEditPatch,
		arg.ID,
		arg.SubjectID,
		arg.FromUserID,
		arg.Reason,
		arg.Name,
		arg.Infobox,
		arg.Summary,
		arg.Nsfw,
		arg.OriginalName,
		arg.OriginalInfobox,
		arg.OriginalSummary,
		arg.SubjectType,
		arg.PatchDesc,
	)
	return err
}

const deleteEpisodePatch = `-- name: DeleteEpisodePatch :exec
update episode_patch
set deleted_at = current_timestamp
where id = $1
  and deleted_at is null
`

func (q *Queries) DeleteEpisodePatch(ctx context.Context, id uuid.UUID) error {
	_, err := q.db.Exec(ctx, deleteEpisodePatch, id)
	return err
}

const deleteSubjectPatch = `-- name: DeleteSubjectPatch :exec
update subject_patch
set deleted_at = current_timestamp
where id = $1
  and deleted_at is null
`

func (q *Queries) DeleteSubjectPatch(ctx context.Context, id uuid.UUID) error {
	_, err := q.db.Exec(ctx, deleteSubjectPatch, id)
	return err
}

const getComments = `-- name: GetComments :many
select edit_suggestion.id, edit_suggestion.patch_id, edit_suggestion.patch_type, edit_suggestion.text, edit_suggestion.from_user, edit_suggestion.created_at, edit_suggestion.deleted_at, author.user_id, author.username, author.nickname
from edit_suggestion
         left join patch_users as author on author.user_id = edit_suggestion.from_user
where deleted_at is null
  and patch_id = $1
  and patch_type = $2
  and deleted_at is null
order by created_at
`

type GetCommentsParams struct {
	PatchID   uuid.UUID
	PatchType string
}

type GetCommentsRow struct {
	ID        uuid.UUID
	PatchID   uuid.UUID
	PatchType string
	Text      string
	FromUser  int32
	CreatedAt pgtype.Timestamptz
	DeletedAt pgtype.Timestamptz
	UserID    pgtype.Int4
	Username  pgtype.Text
	Nickname  pgtype.Text
}

func (q *Queries) GetComments(ctx context.Context, arg GetCommentsParams) ([]GetCommentsRow, error) {
	rows, err := q.db.Query(ctx, getComments, arg.PatchID, arg.PatchType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetCommentsRow
	for rows.Next() {
		var i GetCommentsRow
		if err := rows.Scan(
			&i.ID,
			&i.PatchID,
			&i.PatchType,
			&i.Text,
			&i.FromUser,
			&i.CreatedAt,
			&i.DeletedAt,
			&i.UserID,
			&i.Username,
			&i.Nickname,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getEpisodePatchByID = `-- name: GetEpisodePatchByID :one
select id, episode_id, state, from_user_id, wiki_user_id, reason, original_name, name, original_name_cn, name_cn, original_duration, duration, original_airdate, airdate, original_description, description, created_at, updated_at, deleted_at, reject_reason, subject_id, comments_count, patch_desc, ep
from episode_patch
where deleted_at is null
  and id = $1
limit 1
`

func (q *Queries) GetEpisodePatchByID(ctx context.Context, id uuid.UUID) (EpisodePatch, error) {
	row := q.db.QueryRow(ctx, getEpisodePatchByID, id)
	var i EpisodePatch
	err := row.Scan(
		&i.ID,
		&i.EpisodeID,
		&i.State,
		&i.FromUserID,
		&i.WikiUserID,
		&i.Reason,
		&i.OriginalName,
		&i.Name,
		&i.OriginalNameCn,
		&i.NameCn,
		&i.OriginalDuration,
		&i.Duration,
		&i.OriginalAirdate,
		&i.Airdate,
		&i.OriginalDescription,
		&i.Description,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.DeletedAt,
		&i.RejectReason,
		&i.SubjectID,
		&i.CommentsCount,
		&i.PatchDesc,
		&i.Ep,
	)
	return i, err
}

const getEpisodePatchByIDForUpdate = `-- name: GetEpisodePatchByIDForUpdate :one
select id, episode_id, state, from_user_id, wiki_user_id, reason, original_name, name, original_name_cn, name_cn, original_duration, duration, original_airdate, airdate, original_description, description, created_at, updated_at, deleted_at, reject_reason, subject_id, comments_count, patch_desc, ep
from episode_patch
where deleted_at is null
  and id = $1
limit 1 for update
`

func (q *Queries) GetEpisodePatchByIDForUpdate(ctx context.Context, id uuid.UUID) (EpisodePatch, error) {
	row := q.db.QueryRow(ctx, getEpisodePatchByIDForUpdate, id)
	var i EpisodePatch
	err := row.Scan(
		&i.ID,
		&i.EpisodeID,
		&i.State,
		&i.FromUserID,
		&i.WikiUserID,
		&i.Reason,
		&i.OriginalName,
		&i.Name,
		&i.OriginalNameCn,
		&i.NameCn,
		&i.OriginalDuration,
		&i.Duration,
		&i.OriginalAirdate,
		&i.Airdate,
		&i.OriginalDescription,
		&i.Description,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.DeletedAt,
		&i.RejectReason,
		&i.SubjectID,
		&i.CommentsCount,
		&i.PatchDesc,
		&i.Ep,
	)
	return i, err
}

const getSubjectPatchByID = `-- name: GetSubjectPatchByID :one
select id, subject_id, state, from_user_id, wiki_user_id, reason, name, original_name, infobox, original_infobox, summary, original_summary, nsfw, created_at, updated_at, deleted_at, reject_reason, subject_type, comments_count, patch_desc, original_platform, platform, action
from subject_patch
where deleted_at is null
  and id = $1
limit 1
`

func (q *Queries) GetSubjectPatchByID(ctx context.Context, id uuid.UUID) (SubjectPatch, error) {
	row := q.db.QueryRow(ctx, getSubjectPatchByID, id)
	var i SubjectPatch
	err := row.Scan(
		&i.ID,
		&i.SubjectID,
		&i.State,
		&i.FromUserID,
		&i.WikiUserID,
		&i.Reason,
		&i.Name,
		&i.OriginalName,
		&i.Infobox,
		&i.OriginalInfobox,
		&i.Summary,
		&i.OriginalSummary,
		&i.Nsfw,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.DeletedAt,
		&i.RejectReason,
		&i.SubjectType,
		&i.CommentsCount,
		&i.PatchDesc,
		&i.OriginalPlatform,
		&i.Platform,
		&i.Action,
	)
	return i, err
}

const getSubjectPatchByIDForUpdate = `-- name: GetSubjectPatchByIDForUpdate :one
select id, subject_id, state, from_user_id, wiki_user_id, reason, name, original_name, infobox, original_infobox, summary, original_summary, nsfw, created_at, updated_at, deleted_at, reject_reason, subject_type, comments_count, patch_desc, original_platform, platform, action
from subject_patch
where deleted_at is null
  and id = $1
limit 1 for update
`

func (q *Queries) GetSubjectPatchByIDForUpdate(ctx context.Context, id uuid.UUID) (SubjectPatch, error) {
	row := q.db.QueryRow(ctx, getSubjectPatchByIDForUpdate, id)
	var i SubjectPatch
	err := row.Scan(
		&i.ID,
		&i.SubjectID,
		&i.State,
		&i.FromUserID,
		&i.WikiUserID,
		&i.Reason,
		&i.Name,
		&i.OriginalName,
		&i.Infobox,
		&i.OriginalInfobox,
		&i.Summary,
		&i.OriginalSummary,
		&i.Nsfw,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.DeletedAt,
		&i.RejectReason,
		&i.SubjectType,
		&i.CommentsCount,
		&i.PatchDesc,
		&i.OriginalPlatform,
		&i.Platform,
		&i.Action,
	)
	return i, err
}

const getUserByID = `-- name: GetUserByID :one
select user_id, username, nickname
from patch_users
where user_id = $1
`

func (q *Queries) GetUserByID(ctx context.Context, userID int32) (PatchUser, error) {
	row := q.db.QueryRow(ctx, getUserByID, userID)
	var i PatchUser
	err := row.Scan(&i.UserID, &i.Username, &i.Nickname)
	return i, err
}

const listEpisodePatches = `-- name: ListEpisodePatches :many
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
  and ((from_user_id = $1 and $1 != 0) or $1 = 0)
  and ((wiki_user_id = $2 and $2 != 0) or $2 = 0)
  and state = any ($3::int[])
order by case when $4::text = 'created_at' then created_at end desc,
         case when $4 = 'updated_at' then updated_at end desc,
         case when $4 = '' then created_at end desc
limit $6::int8 offset $5::int8
`

type ListEpisodePatchesParams struct {
	FromUserID int32
	WikiUserID int32
	State      []int32
	OrderBy    string
	Skip       int64
	Size       int64
}

type ListEpisodePatchesRow struct {
	ID               uuid.UUID
	OriginalName     pgtype.Text
	State            int32
	CreatedAt        pgtype.Timestamptz
	UpdatedAt        pgtype.Timestamptz
	CommentsCount    int32
	Reason           string
	AuthorUserID     int32
	AuthorUsername   string
	AuthorNickname   string
	ReviewerUserID   pgtype.Int4
	ReviewerUsername pgtype.Text
	ReviewerNickname pgtype.Text
}

func (q *Queries) ListEpisodePatches(ctx context.Context, arg ListEpisodePatchesParams) ([]ListEpisodePatchesRow, error) {
	rows, err := q.db.Query(ctx, listEpisodePatches,
		arg.FromUserID,
		arg.WikiUserID,
		arg.State,
		arg.OrderBy,
		arg.Skip,
		arg.Size,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ListEpisodePatchesRow
	for rows.Next() {
		var i ListEpisodePatchesRow
		if err := rows.Scan(
			&i.ID,
			&i.OriginalName,
			&i.State,
			&i.CreatedAt,
			&i.UpdatedAt,
			&i.CommentsCount,
			&i.Reason,
			&i.AuthorUserID,
			&i.AuthorUsername,
			&i.AuthorNickname,
			&i.ReviewerUserID,
			&i.ReviewerUsername,
			&i.ReviewerNickname,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const listPendingEpisodePatches = `-- name: ListPendingEpisodePatches :many
select id, episode_id, created_at, updated_at, from_user_id
from episode_patch
where state = 0
  and deleted_at is null
`

type ListPendingEpisodePatchesRow struct {
	ID         uuid.UUID
	EpisodeID  int32
	CreatedAt  pgtype.Timestamptz
	UpdatedAt  pgtype.Timestamptz
	FromUserID int32
}

func (q *Queries) ListPendingEpisodePatches(ctx context.Context) ([]ListPendingEpisodePatchesRow, error) {
	rows, err := q.db.Query(ctx, listPendingEpisodePatches)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ListPendingEpisodePatchesRow
	for rows.Next() {
		var i ListPendingEpisodePatchesRow
		if err := rows.Scan(
			&i.ID,
			&i.EpisodeID,
			&i.CreatedAt,
			&i.UpdatedAt,
			&i.FromUserID,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const listPendingSubjectPatches = `-- name: ListPendingSubjectPatches :many
select id, subject_id, created_at, updated_at, from_user_id
from subject_patch
where state = 0
  and deleted_at is null
`

type ListPendingSubjectPatchesRow struct {
	ID         uuid.UUID
	SubjectID  int32
	CreatedAt  pgtype.Timestamptz
	UpdatedAt  pgtype.Timestamptz
	FromUserID int32
}

func (q *Queries) ListPendingSubjectPatches(ctx context.Context) ([]ListPendingSubjectPatchesRow, error) {
	rows, err := q.db.Query(ctx, listPendingSubjectPatches)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ListPendingSubjectPatchesRow
	for rows.Next() {
		var i ListPendingSubjectPatchesRow
		if err := rows.Scan(
			&i.ID,
			&i.SubjectID,
			&i.CreatedAt,
			&i.UpdatedAt,
			&i.FromUserID,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const listSubjectPatches = `-- name: ListSubjectPatches :many
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
  and state = any ($1::int[])
  and ((from_user_id = $2 and $2 != 0) or $2 = 0)
  and ((wiki_user_id = $3 and $3 != 0) or $3 = 0)
  and action = 1
order by case when $4::text = 'created_at' then created_at end desc,
         case when $4 = 'updated_at' then updated_at end desc,
         case when $4 = '' then created_at end desc
limit $6::int8 offset $5::int8
`

type ListSubjectPatchesParams struct {
	State      []int32
	FromUserID int32
	WikiUserID int32
	OrderBy    string
	Skip       int64
	Size       int64
}

type ListSubjectPatchesRow struct {
	ID               uuid.UUID
	OriginalName     string
	State            int32
	Action           pgtype.Int4
	CreatedAt        pgtype.Timestamptz
	UpdatedAt        pgtype.Timestamptz
	CommentsCount    int32
	Reason           string
	SubjectType      int64
	AuthorUserID     int32
	AuthorUsername   string
	AuthorNickname   string
	ReviewerUserID   pgtype.Int4
	ReviewerUsername pgtype.Text
	ReviewerNickname pgtype.Text
}

func (q *Queries) ListSubjectPatches(ctx context.Context, arg ListSubjectPatchesParams) ([]ListSubjectPatchesRow, error) {
	rows, err := q.db.Query(ctx, listSubjectPatches,
		arg.State,
		arg.FromUserID,
		arg.WikiUserID,
		arg.OrderBy,
		arg.Skip,
		arg.Size,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ListSubjectPatchesRow
	for rows.Next() {
		var i ListSubjectPatchesRow
		if err := rows.Scan(
			&i.ID,
			&i.OriginalName,
			&i.State,
			&i.Action,
			&i.CreatedAt,
			&i.UpdatedAt,
			&i.CommentsCount,
			&i.Reason,
			&i.SubjectType,
			&i.AuthorUserID,
			&i.AuthorUsername,
			&i.AuthorNickname,
			&i.ReviewerUserID,
			&i.ReviewerUsername,
			&i.ReviewerNickname,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const nextPendingEpisodePatch = `-- name: NextPendingEpisodePatch :one
select id
from episode_patch
where state = 0
  and deleted_at is null
  and id < $1
order by id desc
limit 1
`

func (q *Queries) NextPendingEpisodePatch(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
	row := q.db.QueryRow(ctx, nextPendingEpisodePatch, id)
	err := row.Scan(&id)
	return id, err
}

const nextPendingSubjectPatch = `-- name: NextPendingSubjectPatch :one
select id
from subject_patch
where state = 0
  and deleted_at is null
  and id < $1
order by id desc
limit 1
`

func (q *Queries) NextPendingSubjectPatch(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
	row := q.db.QueryRow(ctx, nextPendingSubjectPatch, id)
	err := row.Scan(&id)
	return id, err
}

const rejectEpisodePatch = `-- name: RejectEpisodePatch :exec
update episode_patch
set wiki_user_id  = $1,
    state         = $2,
    reject_reason = $3,
    updated_at    = current_timestamp
where id = $4
  and deleted_at is null
  and state = 0
`

type RejectEpisodePatchParams struct {
	WikiUserID   int32
	State        int32
	RejectReason string
	ID           uuid.UUID
}

func (q *Queries) RejectEpisodePatch(ctx context.Context, arg RejectEpisodePatchParams) error {
	_, err := q.db.Exec(ctx, rejectEpisodePatch,
		arg.WikiUserID,
		arg.State,
		arg.RejectReason,
		arg.ID,
	)
	return err
}

const rejectSubjectPatch = `-- name: RejectSubjectPatch :exec
update subject_patch
set wiki_user_id  = $1,
    state         = $2,
    reject_reason = $3,
    updated_at    = current_timestamp
where id = $4
  and deleted_at is null
  and state = 0
`

type RejectSubjectPatchParams struct {
	WikiUserID   int32
	State        int32
	RejectReason string
	ID           uuid.UUID
}

func (q *Queries) RejectSubjectPatch(ctx context.Context, arg RejectSubjectPatchParams) error {
	_, err := q.db.Exec(ctx, rejectSubjectPatch,
		arg.WikiUserID,
		arg.State,
		arg.RejectReason,
		arg.ID,
	)
	return err
}

const testDelete = `-- name: TestDelete :exec
delete
from edit_suggestion
where id = $1
`

func (q *Queries) TestDelete(ctx context.Context, id uuid.UUID) error {
	_, err := q.db.Exec(ctx, testDelete, id)
	return err
}

const updateEpisodePatch = `-- name: UpdateEpisodePatch :exec
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
where id = $1
`

type UpdateEpisodePatchParams struct {
	ID                  uuid.UUID
	Reason              string
	PatchDesc           string
	OriginalName        pgtype.Text
	Name                pgtype.Text
	OriginalNameCn      pgtype.Text
	NameCn              pgtype.Text
	OriginalDuration    pgtype.Text
	Duration            pgtype.Text
	OriginalAirdate     pgtype.Text
	Airdate             pgtype.Text
	OriginalDescription pgtype.Text
	Description         pgtype.Text
}

func (q *Queries) UpdateEpisodePatch(ctx context.Context, arg UpdateEpisodePatchParams) error {
	_, err := q.db.Exec(ctx, updateEpisodePatch,
		arg.ID,
		arg.Reason,
		arg.PatchDesc,
		arg.OriginalName,
		arg.Name,
		arg.OriginalNameCn,
		arg.NameCn,
		arg.OriginalDuration,
		arg.Duration,
		arg.OriginalAirdate,
		arg.Airdate,
		arg.OriginalDescription,
		arg.Description,
	)
	return err
}

const updateEpisodePatchCommentCount = `-- name: UpdateEpisodePatchCommentCount :exec
update episode_patch
set comments_count = (select count(1)
                      from edit_suggestion
                      where patch_type = 'episode'
                        and patch_id = $1
                        and edit_suggestion.from_user != 0)
where id = $1
  and deleted_at is null
`

func (q *Queries) UpdateEpisodePatchCommentCount(ctx context.Context, patchID uuid.UUID) error {
	_, err := q.db.Exec(ctx, updateEpisodePatchCommentCount, patchID)
	return err
}

const updateSubjectPatch = `-- name: UpdateSubjectPatch :exec
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
where id = $1
`

type UpdateSubjectPatchParams struct {
	ID              uuid.UUID
	OriginalName    string
	Name            pgtype.Text
	OriginalInfobox pgtype.Text
	Infobox         pgtype.Text
	OriginalSummary pgtype.Text
	Summary         pgtype.Text
	Nsfw            pgtype.Bool
	Reason          string
	PatchDesc       string
}

func (q *Queries) UpdateSubjectPatch(ctx context.Context, arg UpdateSubjectPatchParams) error {
	_, err := q.db.Exec(ctx, updateSubjectPatch,
		arg.ID,
		arg.OriginalName,
		arg.Name,
		arg.OriginalInfobox,
		arg.Infobox,
		arg.OriginalSummary,
		arg.Summary,
		arg.Nsfw,
		arg.Reason,
		arg.PatchDesc,
	)
	return err
}

const updateSubjectPatchCommentCount = `-- name: UpdateSubjectPatchCommentCount :exec
update subject_patch
set comments_count = (select count(1)
                      from edit_suggestion
                      where patch_type = 'subject'
                        and patch_id = $1
                        and edit_suggestion.from_user != 0)
where id = $1
  and deleted_at is null
`

func (q *Queries) UpdateSubjectPatchCommentCount(ctx context.Context, patchID uuid.UUID) error {
	_, err := q.db.Exec(ctx, updateSubjectPatchCommentCount, patchID)
	return err
}

const upsertUser = `-- name: UpsertUser :exec
insert into patch_users (user_id, username, nickname)
VALUES ($1, $2, $3)
on conflict (user_id) do update set username = excluded.username,
                                    nickname = excluded.nickname
`

type UpsertUserParams struct {
	UserID   int32
	Username string
	Nickname string
}

func (q *Queries) UpsertUser(ctx context.Context, arg UpsertUserParams) error {
	_, err := q.db.Exec(ctx, upsertUser, arg.UserID, arg.Username, arg.Nickname)
	return err
}
