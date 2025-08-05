package test

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/input-output-hk/catalyst-forge/lib/foundry/client/users"
)

// generateTestEmail generates a unique test email
func generateTestEmail() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return fmt.Sprintf("test-user-%x@example.com", bytes)
}

// generateTestKid generates a unique test key ID
func generateTestKid() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return fmt.Sprintf("test-key-%x", bytes)
}

// generateTestPubKey generates a test public key
func generateTestPubKey() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return base64.StdEncoding.EncodeToString(bytes)
}

func TestUsersAPI(t *testing.T) {
	c := newTestClient()
	ctx, cancel := newTestContext()
	defer cancel()

	t.Run("UserManagement", func(t *testing.T) {
		testEmail := generateTestEmail()

		t.Run("CreateUser", func(t *testing.T) {
			req := &users.CreateUserRequest{
				Email:  testEmail,
				Status: "active",
			}

			createdUser, err := c.Users().Create(ctx, req)
			require.NoError(t, err)

			assert.NotZero(t, createdUser.ID)
			assert.Equal(t, testEmail, createdUser.Email)
			assert.Equal(t, "active", createdUser.Status)
			assert.NotZero(t, createdUser.CreatedAt)
			assert.NotZero(t, createdUser.UpdatedAt)

			userID := createdUser.ID
			t.Logf("Created user with ID: %d", userID)

			t.Run("GetUser", func(t *testing.T) {
				fetchedUser, err := c.Users().Get(ctx, userID)
				require.NoError(t, err)

				assert.Equal(t, userID, fetchedUser.ID)
				assert.Equal(t, testEmail, fetchedUser.Email)
				assert.Equal(t, "active", fetchedUser.Status)
			})

			t.Run("GetUserByEmail", func(t *testing.T) {
				fetchedUser, err := c.Users().GetByEmail(ctx, testEmail)
				require.NoError(t, err)

				assert.Equal(t, userID, fetchedUser.ID)
				assert.Equal(t, testEmail, fetchedUser.Email)
				assert.Equal(t, "active", fetchedUser.Status)
			})

			t.Run("UpdateUser", func(t *testing.T) {
				newEmail := generateTestEmail()
				updateReq := &users.UpdateUserRequest{
					Email:  newEmail,
					Status: "inactive",
				}

				updatedUser, err := c.Users().Update(ctx, userID, updateReq)
				require.NoError(t, err)

				assert.Equal(t, userID, updatedUser.ID)
				assert.Equal(t, newEmail, updatedUser.Email)
				assert.Equal(t, "inactive", updatedUser.Status)

				// Verify the update persisted
				fetchedUser, err := c.Users().Get(ctx, userID)
				require.NoError(t, err)
				assert.Equal(t, newEmail, fetchedUser.Email)
				assert.Equal(t, "inactive", fetchedUser.Status)
			})

			t.Run("ActivateUser", func(t *testing.T) {
				activatedUser, err := c.Users().Activate(ctx, userID)
				require.NoError(t, err)

				assert.Equal(t, userID, activatedUser.ID)
				assert.Equal(t, "active", activatedUser.Status)

				// Verify activation persisted
				fetchedUser, err := c.Users().Get(ctx, userID)
				require.NoError(t, err)
				assert.Equal(t, "active", fetchedUser.Status)
			})

			t.Run("DeactivateUser", func(t *testing.T) {
				deactivatedUser, err := c.Users().Deactivate(ctx, userID)
				require.NoError(t, err)

				assert.Equal(t, userID, deactivatedUser.ID)
				assert.Equal(t, "inactive", deactivatedUser.Status)

				// Verify deactivation persisted
				fetchedUser, err := c.Users().Get(ctx, userID)
				require.NoError(t, err)
				assert.Equal(t, "inactive", fetchedUser.Status)
			})

			t.Run("DeleteUser", func(t *testing.T) {
				err := c.Users().Delete(ctx, userID)
				require.NoError(t, err)

				// Verify user was deleted
				_, err = c.Users().Get(ctx, userID)
				assert.Error(t, err, "Expected error when getting deleted user")
			})
		})

		t.Run("RegisterUser", func(t *testing.T) {
			registerEmail := generateTestEmail()
			req := &users.RegisterUserRequest{
				Email: registerEmail,
			}

			registeredUser, err := c.Users().Register(ctx, req)
			require.NoError(t, err)

			assert.NotZero(t, registeredUser.ID)
			assert.Equal(t, registerEmail, registeredUser.Email)
			assert.Equal(t, "pending", registeredUser.Status)

			// Clean up
			err = c.Users().Delete(ctx, registeredUser.ID)
			require.NoError(t, err)
		})

		t.Run("ListUsers", func(t *testing.T) {
			// Create a few test users
			user1 := &users.CreateUserRequest{Email: generateTestEmail(), Status: "active"}
			user2 := &users.CreateUserRequest{Email: generateTestEmail(), Status: "inactive"}

			createdUser1, err := c.Users().Create(ctx, user1)
			require.NoError(t, err)

			createdUser2, err := c.Users().Create(ctx, user2)
			require.NoError(t, err)

			// List all users
			userList, err := c.Users().List(ctx)
			require.NoError(t, err)

			// Verify our test users are in the list
			found1, found2 := false, false
			for _, u := range userList {
				if u.ID == createdUser1.ID {
					found1 = true
					assert.Equal(t, user1.Email, u.Email)
				}
				if u.ID == createdUser2.ID {
					found2 = true
					assert.Equal(t, user2.Email, u.Email)
				}
			}

			assert.True(t, found1, "First test user not found in list")
			assert.True(t, found2, "Second test user not found in list")

			// Clean up
			c.Users().Delete(ctx, createdUser1.ID)
			c.Users().Delete(ctx, createdUser2.ID)
		})

		t.Run("GetPendingUsers", func(t *testing.T) {
			// Create a pending user
			registerReq := &users.RegisterUserRequest{Email: generateTestEmail()}
			pendingUser, err := c.Users().Register(ctx, registerReq)
			require.NoError(t, err)

			// Get pending users
			pendingUsers, err := c.Users().GetPending(ctx)
			require.NoError(t, err)

			// Verify our pending user is in the list
			found := false
			for _, u := range pendingUsers {
				if u.ID == pendingUser.ID {
					found = true
					assert.Equal(t, "pending", u.Status)
				}
			}

			assert.True(t, found, "Pending user not found in pending users list")

			// Clean up
			c.Users().Delete(ctx, pendingUser.ID)
		})
	})

	t.Run("RoleManagement", func(t *testing.T) {
		t.Run("CreateRole", func(t *testing.T) {
			roleName := fmt.Sprintf("test-role-%d", time.Now().Unix())
			req := &users.CreateRoleRequest{
				Name:        roleName,
				Permissions: []string{"read", "write"},
			}

			createdRole, err := c.Roles().Create(ctx, req)
			require.NoError(t, err)

			assert.NotZero(t, createdRole.ID)
			assert.Equal(t, roleName, createdRole.Name)
			assert.Equal(t, []string{"read", "write"}, createdRole.Permissions)
			assert.NotZero(t, createdRole.CreatedAt)
			assert.NotZero(t, createdRole.UpdatedAt)

			roleID := createdRole.ID
			t.Logf("Created role with ID: %d", roleID)

			t.Run("GetRole", func(t *testing.T) {
				fetchedRole, err := c.Roles().Get(ctx, roleID)
				require.NoError(t, err)

				assert.Equal(t, roleID, fetchedRole.ID)
				assert.Equal(t, roleName, fetchedRole.Name)
				assert.Equal(t, []string{"read", "write"}, fetchedRole.Permissions)
			})

			t.Run("GetRoleByName", func(t *testing.T) {
				fetchedRole, err := c.Roles().GetByName(ctx, roleName)
				require.NoError(t, err)

				assert.Equal(t, roleID, fetchedRole.ID)
				assert.Equal(t, roleName, fetchedRole.Name)
			})

			t.Run("UpdateRole", func(t *testing.T) {
				updateReq := &users.UpdateRoleRequest{
					Name:        roleName + "-updated",
					Permissions: []string{"read", "write", "delete"},
				}

				updatedRole, err := c.Roles().Update(ctx, roleID, updateReq)
				require.NoError(t, err)

				assert.Equal(t, roleID, updatedRole.ID)
				assert.Equal(t, roleName+"-updated", updatedRole.Name)
				assert.Equal(t, []string{"read", "write", "delete"}, updatedRole.Permissions)

				// Verify the update persisted
				fetchedRole, err := c.Roles().Get(ctx, roleID)
				require.NoError(t, err)
				assert.Equal(t, roleName+"-updated", fetchedRole.Name)
				assert.Equal(t, []string{"read", "write", "delete"}, fetchedRole.Permissions)
			})

			t.Run("ListRoles", func(t *testing.T) {
				roleList, err := c.Roles().List(ctx)
				require.NoError(t, err)

				found := false
				for _, r := range roleList {
					if r.ID == roleID {
						found = true
						assert.Equal(t, roleName+"-updated", r.Name)
					}
				}

				assert.True(t, found, "Created role not found in list")
			})

			t.Run("DeleteRole", func(t *testing.T) {
				err := c.Roles().Delete(ctx, roleID)
				require.NoError(t, err)

				// Verify role was deleted
				_, err = c.Roles().Get(ctx, roleID)
				assert.Error(t, err, "Expected error when getting deleted role")
			})
		})

		t.Run("CreateRoleWithAdmin", func(t *testing.T) {
			roleName := fmt.Sprintf("admin-role-%d", time.Now().Unix())
			req := &users.CreateRoleRequest{
				Name:        roleName,
				Permissions: []string{"read"},
			}

			createdRole, err := c.Roles().CreateWithAdmin(ctx, req)
			require.NoError(t, err)

			assert.NotZero(t, createdRole.ID)
			assert.Equal(t, roleName, createdRole.Name)
			// Admin roles typically get all permissions regardless of what's specified
			assert.NotEmpty(t, createdRole.Permissions)

			// Clean up
			c.Roles().Delete(ctx, createdRole.ID)
		})

		t.Run("UserRoleAssignment", func(t *testing.T) {
			// Create a user and role for testing
			user := &users.CreateUserRequest{Email: generateTestEmail(), Status: "active"}
			createdUser, err := c.Users().Create(ctx, user)
			require.NoError(t, err)

			role := &users.CreateRoleRequest{
				Name:        fmt.Sprintf("assignment-role-%d", time.Now().Unix()),
				Permissions: []string{"read"},
			}
			createdRole, err := c.Roles().Create(ctx, role)
			require.NoError(t, err)

			t.Run("AssignUserToRole", func(t *testing.T) {
				err := c.Roles().AssignUser(ctx, createdUser.ID, createdRole.ID)
				require.NoError(t, err)

				t.Run("GetUserRoles", func(t *testing.T) {
					userRoles, err := c.Roles().GetUserRoles(ctx, createdUser.ID)
					require.NoError(t, err)

					found := false
					for _, ur := range userRoles {
						if ur.RoleID == createdRole.ID {
							found = true
							assert.Equal(t, createdUser.ID, ur.UserID)
						}
					}

					assert.True(t, found, "User-role assignment not found")
				})

				t.Run("GetRoleUsers", func(t *testing.T) {
					roleUsers, err := c.Roles().GetRoleUsers(ctx, createdRole.ID)
					require.NoError(t, err)

					found := false
					for _, ur := range roleUsers {
						if ur.UserID == createdUser.ID {
							found = true
							assert.Equal(t, createdRole.ID, ur.RoleID)
						}
					}

					assert.True(t, found, "Role-user assignment not found")
				})

				t.Run("RemoveUserFromRole", func(t *testing.T) {
					err := c.Roles().RemoveUser(ctx, createdUser.ID, createdRole.ID)
					require.NoError(t, err)

					// Verify removal
					userRoles, err := c.Roles().GetUserRoles(ctx, createdUser.ID)
					require.NoError(t, err)

					found := false
					for _, ur := range userRoles {
						if ur.RoleID == createdRole.ID {
							found = true
						}
					}

					assert.False(t, found, "User-role assignment should have been removed")
				})
			})

			// Clean up
			c.Users().Delete(ctx, createdUser.ID)
			c.Roles().Delete(ctx, createdRole.ID)
		})
	})

	t.Run("UserKeyManagement", func(t *testing.T) {
		// Create a user for key testing
		user := &users.CreateUserRequest{Email: generateTestEmail(), Status: "active"}
		createdUser, err := c.Users().Create(ctx, user)
		require.NoError(t, err)

		t.Run("CreateUserKey", func(t *testing.T) {
			kid := generateTestKid()
			pubKey := generateTestPubKey()
			req := &users.CreateUserKeyRequest{
				UserID:    createdUser.ID,
				Kid:       kid,
				PubKeyB64: pubKey,
				Status:    "active",
			}

			createdKey, err := c.Keys().Create(ctx, req)
			require.NoError(t, err)

			assert.NotZero(t, createdKey.ID)
			assert.Equal(t, createdUser.ID, createdKey.UserID)
			assert.Equal(t, kid, createdKey.Kid)
			assert.Equal(t, pubKey, createdKey.PubKeyB64)
			assert.Equal(t, "active", createdKey.Status)
			assert.NotZero(t, createdKey.CreatedAt)
			assert.NotZero(t, createdKey.UpdatedAt)

			keyID := createdKey.ID
			t.Logf("Created user key with ID: %d", keyID)

			t.Run("GetUserKey", func(t *testing.T) {
				fetchedKey, err := c.Keys().Get(ctx, keyID)
				require.NoError(t, err)

				assert.Equal(t, keyID, fetchedKey.ID)
				assert.Equal(t, createdUser.ID, fetchedKey.UserID)
				assert.Equal(t, kid, fetchedKey.Kid)
				assert.Equal(t, pubKey, fetchedKey.PubKeyB64)
			})

			t.Run("GetUserKeyByKid", func(t *testing.T) {
				fetchedKey, err := c.Keys().GetByKid(ctx, kid)
				require.NoError(t, err)

				assert.Equal(t, keyID, fetchedKey.ID)
				assert.Equal(t, createdUser.ID, fetchedKey.UserID)
				assert.Equal(t, kid, fetchedKey.Kid)
			})

			t.Run("UpdateUserKey", func(t *testing.T) {
				newKid := generateTestKid()
				newPubKey := generateTestPubKey()
				updateReq := &users.UpdateUserKeyRequest{
					Kid:       &newKid,
					PubKeyB64: &newPubKey,
					Status:    stringPtr("inactive"),
				}

				updatedKey, err := c.Keys().Update(ctx, keyID, updateReq)
				require.NoError(t, err)

				assert.Equal(t, keyID, updatedKey.ID)
				assert.Equal(t, newKid, updatedKey.Kid)
				assert.Equal(t, newPubKey, updatedKey.PubKeyB64)
				assert.Equal(t, "inactive", updatedKey.Status)

				// Verify the update persisted
				fetchedKey, err := c.Keys().Get(ctx, keyID)
				require.NoError(t, err)
				assert.Equal(t, newKid, fetchedKey.Kid)
				assert.Equal(t, newPubKey, fetchedKey.PubKeyB64)
				assert.Equal(t, "inactive", fetchedKey.Status)
			})

			t.Run("RevokeUserKey", func(t *testing.T) {
				revokedKey, err := c.Keys().Revoke(ctx, keyID)
				require.NoError(t, err)

				assert.Equal(t, keyID, revokedKey.ID)
				assert.Equal(t, "revoked", revokedKey.Status)

				// Verify revocation persisted
				fetchedKey, err := c.Keys().Get(ctx, keyID)
				require.NoError(t, err)
				assert.Equal(t, "revoked", fetchedKey.Status)
			})

			t.Run("DeleteUserKey", func(t *testing.T) {
				err := c.Keys().Delete(ctx, keyID)
				require.NoError(t, err)

				// Verify key was deleted
				_, err = c.Keys().Get(ctx, keyID)
				assert.Error(t, err, "Expected error when getting deleted key")
			})
		})

		t.Run("RegisterUserKey", func(t *testing.T) {
			kid := generateTestKid()
			pubKey := generateTestPubKey()
			req := &users.RegisterUserKeyRequest{
				Email:     createdUser.Email,
				Kid:       kid,
				PubKeyB64: pubKey,
			}

			registeredKey, err := c.Keys().Register(ctx, req)
			require.NoError(t, err)

			assert.NotZero(t, registeredKey.ID)
			assert.Equal(t, createdUser.ID, registeredKey.UserID)
			assert.Equal(t, kid, registeredKey.Kid)
			assert.Equal(t, pubKey, registeredKey.PubKeyB64)
			assert.Equal(t, "active", registeredKey.Status)

			// Clean up
			c.Keys().Delete(ctx, registeredKey.ID)
		})

		t.Run("ListUserKeys", func(t *testing.T) {
			// Create multiple keys for the user
			key1 := &users.CreateUserKeyRequest{
				UserID:    createdUser.ID,
				Kid:       generateTestKid(),
				PubKeyB64: generateTestPubKey(),
				Status:    "active",
			}
			key2 := &users.CreateUserKeyRequest{
				UserID:    createdUser.ID,
				Kid:       generateTestKid(),
				PubKeyB64: generateTestPubKey(),
				Status:    "inactive",
			}

			createdKey1, err := c.Keys().Create(ctx, key1)
			require.NoError(t, err)

			createdKey2, err := c.Keys().Create(ctx, key2)
			require.NoError(t, err)

			t.Run("ListAllKeys", func(t *testing.T) {
				allKeys, err := c.Keys().List(ctx)
				require.NoError(t, err)

				found1, found2 := false, false
				for _, k := range allKeys {
					if k.ID == createdKey1.ID {
						found1 = true
					}
					if k.ID == createdKey2.ID {
						found2 = true
					}
				}

				assert.True(t, found1, "First test key not found in list")
				assert.True(t, found2, "Second test key not found in list")
			})

			t.Run("GetKeysByUserID", func(t *testing.T) {
				userKeys, err := c.Keys().GetByUserID(ctx, createdUser.ID)
				require.NoError(t, err)

				found1, found2 := false, false
				for _, k := range userKeys {
					if k.ID == createdKey1.ID {
						found1 = true
					}
					if k.ID == createdKey2.ID {
						found2 = true
					}
				}

				assert.True(t, found1, "First test key not found in user keys")
				assert.True(t, found2, "Second test key not found in user keys")
			})

			t.Run("GetActiveKeysByUserID", func(t *testing.T) {
				activeKeys, err := c.Keys().GetActiveByUserID(ctx, createdUser.ID)
				require.NoError(t, err)

				found1, found2 := false, false
				for _, k := range activeKeys {
					if k.ID == createdKey1.ID {
						found1 = true
						assert.Equal(t, "active", k.Status)
					}
					if k.ID == createdKey2.ID {
						found2 = true
					}
				}

				assert.True(t, found1, "Active key not found in active keys")
				assert.False(t, found2, "Inactive key should not be in active keys")
			})

			t.Run("GetInactiveKeysByUserID", func(t *testing.T) {
				inactiveKeys, err := c.Keys().GetInactiveByUserID(ctx, createdUser.ID)
				require.NoError(t, err)

				found1, found2 := false, false
				for _, k := range inactiveKeys {
					if k.ID == createdKey1.ID {
						found1 = true
					}
					if k.ID == createdKey2.ID {
						found2 = true
						assert.Equal(t, "inactive", k.Status)
					}
				}

				assert.False(t, found1, "Active key should not be in inactive keys")
				assert.True(t, found2, "Inactive key not found in inactive keys")
			})

			t.Run("GetInactiveKeys", func(t *testing.T) {
				inactiveKeys, err := c.Keys().GetInactive(ctx)
				require.NoError(t, err)

				found := false
				for _, k := range inactiveKeys {
					if k.ID == createdKey2.ID {
						found = true
						assert.Equal(t, "inactive", k.Status)
					}
				}

				assert.True(t, found, "Inactive key not found in inactive keys list")
			})

			// Clean up
			c.Keys().Delete(ctx, createdKey1.ID)
			c.Keys().Delete(ctx, createdKey2.ID)
		})

		// Clean up user
		c.Users().Delete(ctx, createdUser.ID)
	})
}

// stringPtr returns a pointer to a string
func stringPtr(s string) *string {
	return &s
}
