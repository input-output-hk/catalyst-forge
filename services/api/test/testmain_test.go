//go:build integration

package test

import (
	"context"
	"os"
	"testing"
	"time"

	"fmt"

	"github.com/input-output-hk/catalyst-forge/services/api/test/testutil"
)

var suite *testutil.Suite

func TestMain(m *testing.M) {
	ctx := context.Background()
	fmt.Println("[TestMain] Starting Postgres container...")
	// Start Postgres once
	s, err := testutil.SuiteStart(ctx)
	if err != nil {
		fmt.Println("[TestMain] SuiteStart error:", err)
		panic(err)
	}
	suite = s

	// Run migrations and snapshot for fast per-test restores
	fmt.Println("[TestMain] Running migrations and snapshot...")
	if err := suite.SnapshotMigrations(ctx); err != nil {
		fmt.Println("[TestMain] SnapshotMigrations error:", err)
		panic(err)
	}

	code := m.Run()

	// Stop container
	_ = suite.SuiteStop(context.Background())
	// Allow container to stop cleanly
	time.Sleep(100 * time.Millisecond)
	os.Exit(code)
}
