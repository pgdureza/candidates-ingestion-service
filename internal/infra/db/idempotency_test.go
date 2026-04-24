package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStoreAndRetrieveIdempotencyKey tests idempotency logic
func TestStoreAndRetrieveIdempotencyKey(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// NOTE: This test requires a running PostgreSQL instance
	dsn := "postgres://user:password@localhost:5432/candidates_test?sslmode=disable"
	db, err := New(dsn)
	if err != nil {
		t.Skipf("Could not connect to test DB: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	requestID := "test-request-123"
	appID := "app-456"

	// First call should not exist
	exists, retAppID, err := db.GetIdempotencyKey(ctx, requestID)
	require.NoError(t, err)
	assert.False(t, exists)

	// Store the key
	err = db.StoreIdempotencyKey(ctx, requestID, appID)
	require.NoError(t, err)

	// Second call should exist
	exists, retAppID, err = db.GetIdempotencyKey(ctx, requestID)
	require.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, appID, retAppID)
}

// TestIdempotencyKeyDeduplication tests duplicate keys are handled gracefully
func TestIdempotencyKeyDeduplication(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	dsn := "postgres://user:password@localhost:5432/candidates_test?sslmode=disable"
	db, err := New(dsn)
	if err != nil {
		t.Skipf("Could not connect to test DB: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	requestID := "dedup-test-123"
	appID1 := "app-111"
	appID2 := "app-222"

	// Store first
	err = db.StoreIdempotencyKey(ctx, requestID, appID1)
	require.NoError(t, err)

	// Try to store duplicate (should be ignored due to UNIQUE constraint)
	err = db.StoreIdempotencyKey(ctx, requestID, appID2)
	// Error or success depending on DB config (ON CONFLICT DO NOTHING)
	if err != nil {
		t.Logf("Expected behavior: duplicate key ignored or error: %v", err)
	}

	// Verify original value is still there
	exists, retAppID, err := db.GetIdempotencyKey(ctx, requestID)
	require.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, appID1, retAppID)
}
