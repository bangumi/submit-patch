package main

import (
	"context"

	"golang.org/x/sync/errgroup"

	"app/dal"
)

type PendingPatchCounts struct {
	SubjectPatchCount   int64
	EpisodePatchCount   int64
	CharacterPatchCount int64
	PersonPatchCount    int64
}

func CountPendingPatch(ctx context.Context, q *dal.Queries) (PendingPatchCounts, error) {
	var counts PendingPatchCounts

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		counts.SubjectPatchCount, err = q.CountPendingSubjectPatch(ctx)
		return err
	})

	g.Go(func() error {
		var err error
		counts.EpisodePatchCount, err = q.CountPendingEpisodePatch(ctx)
		return err
	})

	g.Go(func() error {
		var err error
		counts.CharacterPatchCount, err = q.CountPendingCharacterPatch(ctx)
		return err
	})

	g.Go(func() error {
		var err error
		counts.PersonPatchCount, err = q.CountPendingPersonPatch(ctx)
		return err
	})

	if err := g.Wait(); err != nil {
		return PendingPatchCounts{}, err
	}

	return counts, nil
}
