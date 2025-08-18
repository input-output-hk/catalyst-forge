//go:build integration

package rbac

import (
	"context"
	"os"
	"testing"

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

	// No special auth bypass; use real admin token
	if err := suite.SnapshotMigrations(ctx); err != nil {
		panic(err)
	}

	code := m.Run()

	_ = suite.SuiteStop(context.Background())
	os.Exit(code)
}
