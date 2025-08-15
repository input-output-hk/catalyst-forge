//go:build integration

package auth_test

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

	if err := suite.SnapshotMigrations(ctx); err != nil {
		// tolerate a single retry in case of race
		time.Sleep(500 * time.Millisecond)
		if err2 := suite.SnapshotMigrations(ctx); err2 != nil {
			panic(err2)
		}
	}

	code := m.Run()

	_ = suite.SuiteStop(context.Background())
	os.Exit(code)
}
