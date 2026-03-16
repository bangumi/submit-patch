package main

import (
	"context"
	"sync"

	"app/dal"
)

type PendingPatchCounts struct {
	SubjectPatchCount   int64
	EpisodePatchCount   int64
	CharacterPatchCount int64
	PersonPatchCount    int64
}

func CountPendingPatch(ctx context.Context, q *dal.Queries) (PendingPatchCounts, error) {
	var wg sync.WaitGroup
	var counts PendingPatchCounts
	var countErr error

	wg.Add(4)
	go func() {
		defer wg.Done()
		counts.SubjectPatchCount, countErr = q.CountPendingSubjectPatch(ctx)
	}()
	go func() {
		defer wg.Done()
		counts.EpisodePatchCount, countErr = q.CountPendingEpisodePatch(ctx)
	}()
	go func() {
		defer wg.Done()
		counts.CharacterPatchCount, countErr = q.CountPendingCharacterPatch(ctx)
	}()
	go func() {
		defer wg.Done()
		counts.PersonPatchCount, countErr = q.CountPendingPersonPatch(ctx)
	}()
	wg.Wait()

	return counts, countErr
}
