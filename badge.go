package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

func (h *handler) badge(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context() // Use request context for cancellation propagation
	cacheKey := "patch:rest:pending"

	cachedBadge, err := h.r.Do(ctx, h.r.B().Get().Key(cacheKey).Build()).AsBytes()
	if err == nil {
		writeBadgeResponse(w, cachedBadge)
		return
	}

	// 2. Query database concurrently to get pending counts
	var wg errgroup.Group
	var countSubject, countEpisode int64

	wg.Go(func() error {
		return h.db.QueryRow(ctx, "SELECT count(1) FROM subject_patch WHERE state = $1 and deleted_at is null", PatchStatePending).Scan(&countSubject)
	})

	wg.Go(func() error {
		return h.db.QueryRow(ctx, "SELECT count(1) FROM episode_patch WHERE state = $1 and deleted_at is null", PatchStatePending).Scan(&countEpisode)
	})

	err = wg.Wait()
	if err != nil {
		log.Err(err).Msg("failed to get pending count from database")
		http.Error(w, "failed to get pending count from database", http.StatusInternalServerError)
		return
	}

	totalCount := countSubject + countEpisode

	badge, err := h.getBadge(ctx, totalCount)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate badge: %v", err), http.StatusInternalServerError)
		return
	}

	err = h.r.Do(ctx, h.r.B().Set().Key(cacheKey).Value(string(badge)).Ex(10*time.Second).Build()).NonRedisError()
	if err != nil {
		log.Warn().
			Str("cacheKey", cacheKey).
			Err(err).
			Msg("failed to cache pending count badge in redis")
	}

	writeBadgeResponse(w, badge)
}

func (h *handler) getBadge(ctx context.Context, count int64) ([]byte, error) {
	cachePrefix := "badge:count"
	var cacheKey string
	displayCountStr := strconv.FormatInt(count, 10)

	// Determine the specific cache key and potentially adjust the display string
	// based on the count, rounding down for counts >= 100.
	if count >= 100 {
		roundedCount := (count / 100) * 100                        // e.g., 123 -> 100, 250 -> 200
		cacheKey = fmt.Sprintf("%s:%d", cachePrefix, roundedCount) // Cache key uses rounded value
		displayCountStr = fmt.Sprintf(">%d", roundedCount)         // Display string shows ">100", ">200", etc.
	} else {
		cacheKey = fmt.Sprintf("%s:%d", cachePrefix, count) // Cache key uses exact count
	}

	// 1. Check Redis cache for this specific count/range badge
	badge, err := h.r.Do(ctx, h.r.B().Get().Key(cacheKey).Build()).AsBytes()
	if err == nil {
		return badge, nil
	}

	// 2. Determine badge color based on the count
	var color string
	if count >= 100 {
		color = "dc3545" // Red
	} else if count >= 50 {
		color = "ffc107" // Yellow
	} else {
		color = "green" // Green
	}

	// 3. Construct the shields.io URL and fetch the badge
	// Note: Ensure the label "待审核" is properly URL-encoded if needed, though shields.io often handles this.
	url := fmt.Sprintf("https://img.shields.io/badge/待审核-%s-%s", displayCountStr, color)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create shields.io request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch badge from shields.io (%s): %w", url, err)
	}
	defer resp.Body.Close()

	// Check if the request to shields.io was successful
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body) // Read body for more context on failure
		return nil, fmt.Errorf("shields.io request failed (%s) with status %d: %s", url, resp.StatusCode, string(bodyBytes))
	}

	// Read the SVG badge content from the response body
	badge, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read shields.io response body: %w", err)
	}

	// 4. Cache the newly fetched badge in Redis (long expiry: 7 days)
	err = h.r.Do(ctx, h.r.B().Set().Key(cacheKey).Value(string(badge)).Ex(7*24*time.Hour).Build()).NonRedisError()
	if err != nil {
		// Log caching errors but return the successfully fetched badge
		fmt.Printf("Warning: failed to cache generated badge in redis (key: %s): %v\n", cacheKey, err)
	}

	return badge, nil
}

// writeBadgeResponse is a helper to set common response headers and write the badge data.
func writeBadgeResponse(w http.ResponseWriter, badgeData []byte) {
	w.Header().Set("Content-Type", "image/svg+xml")
	// Set cache control header as defined in the Python code
	w.Header().Set("Cache-Control", "public, max-age=5")
	// Consider adding ETag or Last-Modified headers for more sophisticated caching
	w.WriteHeader(http.StatusOK)
	w.Write(badgeData)
}
