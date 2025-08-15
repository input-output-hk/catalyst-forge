package rbac

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCondition_RequiresStepUp(t *testing.T) {
	t.Parallel()

	now := time.Date(2025, 1, 5, 10, 0, 0, 0, time.UTC)
	cond, ok := getCondition("requires_step_up")
	require.True(t, ok)

	// Pass: step-up performed within 5m and still valid
	ctx := EvalContext{Now: now, StepUpAt: now.Add(-4 * time.Minute), StepUpValidUntil: now.Add(10 * time.Minute)}
	pass, _, err := cond.Evaluate(ctx, map[string]any{"within": "5m"})
	require.NoError(t, err)
	assert.True(t, pass, "should pass when within window")

	// Fail: no step-up performed
	ctx2 := EvalContext{Now: now, StepUpValidUntil: now.Add(10 * time.Minute)}
	pass, _, err = cond.Evaluate(ctx2, map[string]any{"within": "5m"})
	require.NoError(t, err)
	assert.False(t, pass, "should fail when no step-up present")

	// Fail: step-up too old (performed 10m ago, window 5m)
	ctx3 := EvalContext{Now: now, StepUpAt: now.Add(-10 * time.Minute), StepUpValidUntil: now.Add(10 * time.Minute)}
	pass, _, err = cond.Evaluate(ctx3, map[string]any{"within": "5m"})
	require.NoError(t, err)
	assert.False(t, pass, "should fail when beyond window in current implementation")

	// Pass: exactly within the window (boundary)
	ctx4 := EvalContext{Now: now, StepUpAt: now.Add(-5 * time.Minute), StepUpValidUntil: now.Add(10 * time.Minute)}
	pass, _, err = cond.Evaluate(ctx4, map[string]any{"within": "5m"})
	require.NoError(t, err)
	assert.True(t, pass)
}

func TestCondition_AttrEquals(t *testing.T) {
	t.Parallel()

	cond, ok := getCondition("attr_equals")
	require.True(t, ok)

	ctx := EvalContext{
		Subject:  Subject{Type: SubjectUser, ID: "user-1"},
		Resource: ResourceRef{Attrs: map[string]any{"owner_id": "user-1"}},
	}
	pass, _, err := cond.Evaluate(ctx, map[string]any{"attr": "owner_id", "subject_field": "id"})
	require.NoError(t, err)
	assert.True(t, pass)

	ctx.Resource.Attrs["owner_id"] = "user-2"
	pass, _, err = cond.Evaluate(ctx, map[string]any{"attr": "owner_id", "subject_field": "id"})
	require.NoError(t, err)
	assert.False(t, pass)
}

func TestCondition_OrgMatches(t *testing.T) {
	t.Parallel()

	cond, ok := getCondition("org_matches")
	require.True(t, ok)

	org := uuid.New()
	ctx := EvalContext{
		Subject:  Subject{OrgID: &org},
		Resource: ResourceRef{OrgID: &org},
	}
	pass, _, err := cond.Evaluate(ctx, nil)
	require.NoError(t, err)
	assert.True(t, pass)

	org2 := uuid.New()
	ctx.Resource.OrgID = &org2
	pass, _, err = cond.Evaluate(ctx, nil)
	require.NoError(t, err)
	assert.False(t, pass)

	// If one side missing, passes (defensive default)
	ctx.Subject.OrgID = nil
	pass, _, err = cond.Evaluate(ctx, nil)
	require.NoError(t, err)
	assert.True(t, pass)
}

func TestCondition_TimeWindow(t *testing.T) {
	t.Parallel()

	cond, ok := getCondition("time_window")
	require.True(t, ok)

	now := time.Date(2025, 1, 6, 10, 0, 0, 0, time.UTC) // Monday
	ctx := EvalContext{Now: now}
	params := map[string]any{"start": "09:00Z", "end": "17:00Z", "days": []any{"Mon"}}
	pass, _, err := cond.Evaluate(ctx, params)
	require.NoError(t, err)
	assert.True(t, pass)

	params["days"] = []any{"Tue"}
	pass, _, err = cond.Evaluate(ctx, params)
	require.NoError(t, err)
	assert.False(t, pass)
}

func TestCondition_TimeWindow_Overnight(t *testing.T) {
	t.Parallel()

	cond, ok := getCondition("time_window")
	require.True(t, ok)

	// Window 23:00Z - 06:00Z, Monday
	params := map[string]any{"start": "23:00Z", "end": "06:00Z", "days": []any{"Mon"}}
	ctx := EvalContext{Now: time.Date(2025, 1, 6, 23, 30, 0, 0, time.UTC)} // Monday 23:30
	pass, _, err := cond.Evaluate(ctx, params)
	require.NoError(t, err)
	assert.True(t, pass)

	// 07:00Z should be outside overnight window
	ctx.Now = time.Date(2025, 1, 6, 7, 0, 0, 0, time.UTC)
	pass, _, err = cond.Evaluate(ctx, params)
	require.NoError(t, err)
	assert.False(t, pass)
}

func TestCondition_AttrEquals_MissingAttrs(t *testing.T) {
	t.Parallel()

	cond, ok := getCondition("attr_equals")
	require.True(t, ok)

	ctx := EvalContext{Subject: Subject{Type: SubjectUser, ID: "u1"}, Resource: ResourceRef{}}
	pass, _, err := cond.Evaluate(ctx, map[string]any{"attr": "owner_id", "subject_field": "id"})
	require.NoError(t, err)
	assert.False(t, pass)
}
