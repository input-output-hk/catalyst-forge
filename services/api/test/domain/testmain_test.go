//go:build integration

package domain

import (
	"context"
	"os"
	"testing"
	"time"

	tu "github.com/input-output-hk/catalyst-forge/services/api/test/testutil"
)

var suite *tu.Suite

func TestMain(m *testing.M) {
	ctx := context.Background()

	s, err := tu.SuiteStart(ctx)
	if err != nil {
		panic(err)
	}
	suite = s

	// Enable test auth bypass for domain tests
	os.Setenv("TEST_AUTH_BYPASS", "1")

	if err := suite.SnapshotMigrations(ctx); err != nil {
		time.Sleep(500 * time.Millisecond)
		if err2 := suite.SnapshotMigrations(ctx); err2 != nil {
			panic(err2)
		}
	}

	code := m.Run()

	_ = suite.SuiteStop(context.Background())
	os.Exit(code)
}
