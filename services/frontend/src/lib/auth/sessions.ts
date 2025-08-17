/**
 * Session management utilities for user authentication.
 *
 * This module provides functions to:
 * - List all active sessions
 * - Terminate individual sessions
 * - Terminate multiple sessions
 * - Log out from all devices
 */

import { forge } from "@/lib/client";
import { readResponseError } from "@/lib/api";
import type { components } from "forge-client";

// Type definitions
export type SessionsListResponse = components["schemas"]["auth.SessionsListResponse"];

/**
 * Fetches the list of all active sessions for the current user.
 * @returns Promise resolving to the sessions list response
 * @throws Error if the request fails
 */
export async function listSessions(): Promise<SessionsListResponse> {
  const response = await forge.raw.GET("/api/v1/auth/sessions");

  if (!response.response.ok) {
    const errorMessage = await readResponseError(response, "Failed to list sessions");
    throw new Error(errorMessage);
  }

  return response.data as SessionsListResponse;
}

/**
 * Terminates a specific session by its family ID.
 * @param familyId - The session family ID to terminate
 * @throws Error if the termination fails
 */
export async function deleteSession(familyId: string): Promise<void> {
  const requestParams = {
    params: {
      path: {
        family_id: familyId,
      },
    },
  };

  const response = await forge.raw.DELETE("/api/v1/auth/sessions/{family_id}", requestParams);

  const isSuccess = response.response.status === 204;

  if (!isSuccess) {
    const errorMessage = await readResponseError(response, "Failed to terminate session");
    throw new Error(errorMessage);
  }
}

/**
 * Terminates multiple sessions in parallel.
 * Uses Promise.allSettled to ensure all deletion attempts are made,
 * even if some fail.
 * @param familyIds - Array of session family IDs to terminate
 */
export async function deleteSessions(familyIds: string[]): Promise<void> {
  const deletionPromises = familyIds.map((id) => deleteSession(id));

  await Promise.allSettled(deletionPromises);
}

/**
 * Logs out all sessions across all devices.
 * This will invalidate all active sessions for the user.
 * @throws Error if the logout operation fails
 */
export async function logoutAllSessions(): Promise<void> {
  const response = await forge.raw.POST("/api/v1/auth/logout-all");

  if (!response.response.ok) {
    const errorMessage = await readResponseError(response, "Failed to log out of all devices");
    throw new Error(errorMessage);
  }
}
