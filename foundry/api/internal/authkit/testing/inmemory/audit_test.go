package inmemory

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAuditStore(t *testing.T) {
	t.Parallel()
	
	store := NewAuditStore()
	assert.NotNil(t, store)
}

func TestAuditStore_Record(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	userID := uuid.New()
	actorID := uuid.New()
	now := time.Now()
	
	tests := []struct {
		name    string
		event   domain.Event
		wantErr bool
		errMsg  string
	}{
		{
			name: "ok/basic_event",
			event: domain.Event{
				ID:        uuid.New(),
				Type:      domain.EventUserCreated,
				UserID:    &userID,
				ActorID:   &actorID,
				IPAddress: "192.168.1.1",
				UserAgent: "Mozilla/5.0",
				Metadata:  map[string]interface{}{"test": "value"},
				CreatedAt: now,
			},
			wantErr: false,
		},
		{
			name: "ok/minimal_event",
			event: domain.Event{
				ID:        uuid.New(),
				Type:      domain.EventUserDeleted,
				CreatedAt: now,
			},
			wantErr: false,
		},
		{
			name: "ok/nil_user_id",
			event: domain.Event{
				ID:        uuid.New(),
				Type:      domain.EventUserCreated,
				UserID:    nil,
				ActorID:   &actorID,
				IPAddress: "10.0.0.1",
				CreatedAt: now,
			},
			wantErr: false,
		},
		{
			name: "ok/empty_metadata",
			event: domain.Event{
				ID:        uuid.New(),
				Type:      domain.EventUserRolesUpdated,
				UserID:    &userID,
				Metadata:  map[string]interface{}{},
				CreatedAt: now,
			},
			wantErr: false,
		},
		{
			name: "ok/nil_metadata",
			event: domain.Event{
				ID:        uuid.New(),
				Type:      domain.EventUserSessionBumped,
				UserID:    &userID,
				Metadata:  nil,
				CreatedAt: now,
			},
			wantErr: false,
		},
		{
			name: "ok/complex_metadata",
			event: domain.Event{
				ID:     uuid.New(),
				Type:   domain.EventUserCreated,
				UserID: &userID,
				Metadata: map[string]interface{}{
					"string":  "value",
					"number":  42,
					"boolean": true,
					"array":   []string{"a", "b", "c"},
					"nested": map[string]interface{}{
						"key": "value",
					},
				},
				CreatedAt: now,
			},
			wantErr: false,
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			store := NewAuditStore()
			
			err := store.Record(ctx, tc.event)
			
			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
			} else {
				require.NoError(t, err)
				
				// Verify the event was stored
				events := store.GetEvents()
				require.Len(t, events, 1)
				assert.Equal(t, tc.event, events[0])
			}
		})
	}
}

func TestAuditStore_GetEvents(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	userID := uuid.New()
	now := time.Now()
	
	tests := []struct {
		name     string
		setup    func(*AuditStore) []domain.Event
		validate func(*testing.T, []domain.Event)
	}{
		{
			name: "ok/empty_store",
			setup: func(store *AuditStore) []domain.Event {
				return []domain.Event{}
			},
			validate: func(t *testing.T, events []domain.Event) {
				assert.Len(t, events, 0)
			},
		},
		{
			name: "ok/single_event",
			setup: func(store *AuditStore) []domain.Event {
				event := domain.Event{
					ID:        uuid.New(),
					Type:      domain.EventUserCreated,
					UserID:    &userID,
					CreatedAt: now,
				}
				err := store.Record(ctx, event)
				require.NoError(t, err)
				return []domain.Event{event}
			},
			validate: func(t *testing.T, events []domain.Event) {
				assert.Len(t, events, 1)
				assert.Equal(t, domain.EventUserCreated, events[0].Type)
				assert.Equal(t, userID, *events[0].UserID)
			},
		},
		{
			name: "ok/multiple_events",
			setup: func(store *AuditStore) []domain.Event {
				events := []domain.Event{
					{
						ID:        uuid.New(),
						Type:      domain.EventUserCreated,
						UserID:    &userID,
						CreatedAt: now,
					},
					{
						ID:        uuid.New(),
						Type:      domain.EventUserRolesUpdated,
						UserID:    &userID,
						CreatedAt: now.Add(time.Minute),
					},
					{
						ID:        uuid.New(),
						Type:      domain.EventUserDeleted,
						UserID:    &userID,
						CreatedAt: now.Add(2 * time.Minute),
					},
				}
				
				for _, event := range events {
					err := store.Record(ctx, event)
					require.NoError(t, err)
				}
				
				return events
			},
			validate: func(t *testing.T, events []domain.Event) {
				assert.Len(t, events, 3)
				
				// Verify events are returned in order
				assert.Equal(t, domain.EventUserCreated, events[0].Type)
				assert.Equal(t, domain.EventUserRolesUpdated, events[1].Type)
				assert.Equal(t, domain.EventUserDeleted, events[2].Type)
			},
		},
		{
			name: "ok/different_event_types",
			setup: func(store *AuditStore) []domain.Event {
				eventTypes := []domain.EventType{
					domain.EventUserCreated,
					domain.EventUserRolesUpdated,
					domain.EventUserSessionBumped,
					domain.EventUserDeleted,
					domain.EventUserSuspended,
					domain.EventUserReactivated,
				}
				
				var events []domain.Event
				for i, eventType := range eventTypes {
					event := domain.Event{
						ID:        uuid.New(),
						Type:      eventType,
						UserID:    &userID,
						CreatedAt: now.Add(time.Duration(i) * time.Minute),
					}
					err := store.Record(ctx, event)
					require.NoError(t, err)
					events = append(events, event)
				}
				
				return events
			},
			validate: func(t *testing.T, events []domain.Event) {
				assert.Len(t, events, 6)
				
				expectedTypes := []domain.EventType{
					domain.EventUserCreated,
					domain.EventUserRolesUpdated,
					domain.EventUserSessionBumped,
					domain.EventUserDeleted,
					domain.EventUserSuspended,
					domain.EventUserReactivated,
				}
				
				for i, event := range events {
					assert.Equal(t, expectedTypes[i], event.Type)
				}
			},
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			store := NewAuditStore()
			
			expectedEvents := tc.setup(store)
			actualEvents := store.GetEvents()
			
			if tc.validate != nil {
				tc.validate(t, actualEvents)
			}
			
			// Verify we got the expected events
			assert.Equal(t, expectedEvents, actualEvents)
		})
	}
}

func TestAuditStore_EventOrdering(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	store := NewAuditStore()
	userID := uuid.New()
	now := time.Now()
	
	// Record events in specific order
	events := []domain.Event{
		{
			ID:        uuid.New(),
			Type:      domain.EventUserCreated,
			UserID:    &userID,
			Metadata:  map[string]interface{}{"order": 1},
			CreatedAt: now,
		},
		{
			ID:        uuid.New(),
			Type:      domain.EventUserRolesUpdated,
			UserID:    &userID,
			Metadata:  map[string]interface{}{"order": 2},
			CreatedAt: now.Add(time.Minute),
		},
		{
			ID:        uuid.New(),
			Type:      domain.EventUserSessionBumped,
			UserID:    &userID,
			Metadata:  map[string]interface{}{"order": 3},
			CreatedAt: now.Add(2 * time.Minute),
		},
	}
	
	for _, event := range events {
		err := store.Record(ctx, event)
		require.NoError(t, err)
	}
	
	// Verify events are returned in the same order they were recorded
	retrievedEvents := store.GetEvents()
	require.Len(t, retrievedEvents, 3)
	
	for i, event := range retrievedEvents {
		expectedOrder := i + 1
		actualOrder := event.Metadata["order"]
		assert.Equal(t, expectedOrder, actualOrder)
	}
}

func TestAuditStore_Concurrency(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	
	t.Run("concurrent_records", func(t *testing.T) {
		t.Parallel()
		
		store := NewAuditStore()
		
		const numGoroutines = 20
		var wg sync.WaitGroup
		wg.Add(numGoroutines)
		
		errors := make(chan error, numGoroutines)
		
		for i := 0; i < numGoroutines; i++ {
			go func(idx int) {
				defer wg.Done()
				
				userID := uuid.New()
				event := domain.Event{
					ID:        uuid.New(),
					Type:      domain.EventUserCreated,
					UserID:    &userID,
					Metadata:  map[string]interface{}{"index": idx},
					CreatedAt: time.Now(),
				}
				
				if err := store.Record(ctx, event); err != nil {
					errors <- err
				}
			}(i)
		}
		
		wg.Wait()
		close(errors)
		
		// Check for errors
		for err := range errors {
			assert.NoError(t, err)
		}
		
		// Verify all events were recorded
		events := store.GetEvents()
		assert.Len(t, events, numGoroutines)
		
		// Verify all indices are present
		indices := make(map[int]bool)
		for _, event := range events {
			if idx, ok := event.Metadata["index"].(int); ok {
				indices[idx] = true
			}
		}
		assert.Len(t, indices, numGoroutines)
	})
	
	t.Run("concurrent_reads", func(t *testing.T) {
		t.Parallel()
		
		store := NewAuditStore()
		
		// Record some events first
		userID := uuid.New()
		for i := 0; i < 5; i++ {
			event := domain.Event{
				ID:        uuid.New(),
				Type:      domain.EventUserCreated,
				UserID:    &userID,
				CreatedAt: time.Now(),
			}
			err := store.Record(ctx, event)
			require.NoError(t, err)
		}
		
		const numGoroutines = 30
		var wg sync.WaitGroup
		wg.Add(numGoroutines)
		
		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()
				
				events := store.GetEvents()
				assert.GreaterOrEqual(t, len(events), 5) // At least 5 events from setup
			}()
		}
		
		wg.Wait()
	})
	
	t.Run("concurrent_record_and_read", func(t *testing.T) {
		t.Parallel()
		
		store := NewAuditStore()
		
		const numOperations = 50
		var wg sync.WaitGroup
		wg.Add(numOperations * 2) // Record and read operations
		
		for i := 0; i < numOperations; i++ {
			go func(idx int) {
				defer wg.Done()
				
				userID := uuid.New()
				event := domain.Event{
					ID:        uuid.New(),
					Type:      domain.EventUserCreated,
					UserID:    &userID,
					Metadata:  map[string]interface{}{"writer": idx},
					CreatedAt: time.Now(),
				}
				
				err := store.Record(ctx, event)
				assert.NoError(t, err)
			}(i)
			
			go func() {
				defer wg.Done()
				
				events := store.GetEvents()
				// Events may or may not be there depending on timing
				assert.GreaterOrEqual(t, len(events), 0)
			}()
		}
		
		wg.Wait()
		
		// Verify final state
		finalEvents := store.GetEvents()
		assert.Len(t, finalEvents, numOperations)
	})
}

func TestAuditStore_DataIsolation(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	
	t.Run("separate_stores_are_isolated", func(t *testing.T) {
		t.Parallel()
		
		store1 := NewAuditStore()
		store2 := NewAuditStore()
		
		userID := uuid.New()
		
		// Record event in store1
		event := domain.Event{
			ID:        uuid.New(),
			Type:      domain.EventUserCreated,
			UserID:    &userID,
			CreatedAt: time.Now(),
		}
		err := store1.Record(ctx, event)
		require.NoError(t, err)
		
		// Verify isolation
		events1 := store1.GetEvents()
		events2 := store2.GetEvents()
		
		assert.Len(t, events1, 1)
		assert.Len(t, events2, 0)
	})
	
	t.Run("modifications_dont_affect_returned_objects", func(t *testing.T) {
		t.Parallel()
		
		store := NewAuditStore()
		userID := uuid.New()
		
		// Record event
		originalEvent := domain.Event{
			ID:        uuid.New(),
			Type:      domain.EventUserCreated,
			UserID:    &userID,
			Metadata:  map[string]interface{}{"original": "value"},
			CreatedAt: time.Now(),
		}
		err := store.Record(ctx, originalEvent)
		require.NoError(t, err)
		
		// Get events
		retrievedEvents := store.GetEvents()
		require.Len(t, retrievedEvents, 1)
		
		// Modify retrieved event
		retrievedEvents[0].Type = domain.EventUserDeleted
		retrievedEvents[0].Metadata["original"] = "modified"
		retrievedEvents[0].Metadata["new"] = "added"
		
		// Get events again - should be unchanged
		unchangedEvents := store.GetEvents()
		require.Len(t, unchangedEvents, 1)
		
		assert.Equal(t, domain.EventUserCreated, unchangedEvents[0].Type)
		assert.Equal(t, "value", unchangedEvents[0].Metadata["original"])
		assert.NotContains(t, unchangedEvents[0].Metadata, "new")
	})
	
	t.Run("record_event_modifications_dont_affect_store", func(t *testing.T) {
		t.Parallel()
		
		store := NewAuditStore()
		userID := uuid.New()
		
		// Create event and record it
		recordEvent := domain.Event{
			ID:        uuid.New(),
			Type:      domain.EventUserCreated,
			UserID:    &userID,
			Metadata:  map[string]interface{}{"key": "original"},
			CreatedAt: time.Now(),
		}
		err := store.Record(ctx, recordEvent)
		require.NoError(t, err)
		
		// Modify original event after recording
		recordEvent.Type = domain.EventUserDeleted
		recordEvent.Metadata["key"] = "modified"
		recordEvent.Metadata["new"] = "added"
		
		// Retrieved event should be unchanged
		retrievedEvents := store.GetEvents()
		require.Len(t, retrievedEvents, 1)
		
		assert.Equal(t, domain.EventUserCreated, retrievedEvents[0].Type)
		assert.Equal(t, "original", retrievedEvents[0].Metadata["key"])
		assert.NotContains(t, retrievedEvents[0].Metadata, "new")
	})
}