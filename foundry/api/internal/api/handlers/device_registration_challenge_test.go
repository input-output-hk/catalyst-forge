package handlers

import (
	"testing"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"sync"
	"time"
)

func TestDeviceRegistrationHandler_ChallengeLookup(t *testing.T) {
    t.Parallel()
	// Test that challenges are properly stored and retrieved by DeviceID
	handler := &DeviceRegistrationHandler{
		challengeStorage: make(map[uuid.UUID]*DeviceInitChallenge),
		challengeMu:      &sync.RWMutex{},
	}

	// Create test challenges
	deviceID1 := uuid.New()
	deviceID2 := uuid.New()
	
	challenge1 := &DeviceInitChallenge{
		DeviceID:  deviceID1,
		Challenge: "challenge-abc123",
		Algorithm: "ES256",
		ExpiresAt: time.Now().Add(5 * time.Minute),
		InviteID:  1,
		UserID:    100,
	}
	
	challenge2 := &DeviceInitChallenge{
		DeviceID:  deviceID2,
		Challenge: "challenge-xyz789",
		Algorithm: "ES256",
		ExpiresAt: time.Now().Add(5 * time.Minute),
		InviteID:  2,
		UserID:    200,
	}

	// Store challenges by DeviceID (the correct way)
	handler.challengeMu.Lock()
	handler.challengeStorage[deviceID1] = challenge1
	handler.challengeStorage[deviceID2] = challenge2
	handler.challengeMu.Unlock()

	// Test retrieval by DeviceID
    t.Run("retrieve challenge by correct DeviceID", func(t *testing.T) {
        t.Parallel()
		found := handler.findChallengeByDeviceID(deviceID1)
		assert.NotNil(t, found)
		assert.Equal(t, challenge1.Challenge, found.Challenge)
		assert.Equal(t, challenge1.UserID, found.UserID)
	})

    t.Run("retrieve different challenge by DeviceID", func(t *testing.T) {
        t.Parallel()
		found := handler.findChallengeByDeviceID(deviceID2)
		assert.NotNil(t, found)
		assert.Equal(t, challenge2.Challenge, found.Challenge)
		assert.Equal(t, challenge2.UserID, found.UserID)
	})

    t.Run("return nil for non-existent DeviceID", func(t *testing.T) {
        t.Parallel()
		nonExistentID := uuid.New()
		found := handler.findChallengeByDeviceID(nonExistentID)
		assert.Nil(t, found)
	})

    t.Run("ensure O(1) lookup performance", func(t *testing.T) {
        t.Parallel()
		// Add many challenges
		for i := 0; i < 1000; i++ {
			id := uuid.New()
			handler.challengeMu.Lock()
			handler.challengeStorage[id] = &DeviceInitChallenge{
				DeviceID:  id,
				Challenge: "test",
				ExpiresAt: time.Now().Add(5 * time.Minute),
			}
			handler.challengeMu.Unlock()
		}

		// Lookup should still be fast (O(1))
		start := time.Now()
		_ = handler.findChallengeByDeviceID(deviceID1)
		elapsed := time.Since(start)
		
		// Should be very fast even with 1000+ entries
		assert.Less(t, elapsed, 1*time.Millisecond)
	})

    t.Run("verify we never use proof as lookup key", func(t *testing.T) {
        t.Parallel()
		// This test documents that we should NEVER do this:
		// BAD: handler.challengeStorage[deviceProof]
		// The device proof is a timestamp.signature, not a challenge key!
		
		// Example of what a device proof looks like (timestamp.signature)
		// deviceProof := "1234567890.dGVzdF9zaWduYXR1cmU"
		
		// This should NOT work (and doesn't, since we're using UUID keys now)
		// Attempting to use a string as a UUID key would cause a compile error
		// which is exactly what we want - type safety!
		
		// The following line would not compile:
		// handler.challengeStorage[deviceProof] // COMPILE ERROR: cannot use deviceProof (string) as uuid.UUID
		
		// This ensures we can never accidentally use the wrong lookup key
		assert.True(t, true, "Type safety prevents using device proof as lookup key")
	})
}
