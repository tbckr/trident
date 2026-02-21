package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWait_NoDelay(t *testing.T) {
	// Large burst means tokens are immediately available — Wait should return fast.
	l := New(100, 100)
	start := time.Now()
	err := l.Wait(context.Background())
	require.NoError(t, err)
	assert.Less(t, time.Since(start), 50*time.Millisecond)
}

func TestWait_ContextCancelled(t *testing.T) {
	// 1 RPS limiter — second call must wait ~1s; cancelling should unblock it.
	l := New(1, 1)
	// Consume the only available token.
	require.NoError(t, l.Wait(context.Background()))

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	err := l.Wait(ctx)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestWait_JitterWithinBounds(t *testing.T) {
	// Run several waits and verify jitter stays within ±20%.
	// Each iteration uses a fresh limiter so there is no drift between runs.
	// 2 RPS → expected delay ~500ms for the second token after burst exhausted.
	const runs = 5
	const expectedDelay = 500 * time.Millisecond
	const tolerance = 0.25 // allow slightly more than 20% for timer resolution

	for range runs {
		l := New(2, 1)
		// Consume burst token instantly.
		require.NoError(t, l.Wait(context.Background()))

		start := time.Now()
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		err := l.Wait(ctx)
		cancel()
		require.NoError(t, err)

		elapsed := time.Since(start)
		margin := time.Duration(float64(expectedDelay) * tolerance)
		assert.GreaterOrEqual(t, elapsed, expectedDelay-margin, "wait was too short (jitter > ±20%%)")
		assert.LessOrEqual(t, elapsed, expectedDelay+margin, "wait was too long (jitter > ±20%%)")
	}
}
