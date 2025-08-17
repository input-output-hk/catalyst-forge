/**
 * Credential management utilities for WebAuthn passkeys.
 *
 * This module provides functions to:
 * - List all registered credentials/passkeys
 * - Rename device names for credentials
 * - Delete credentials from the account
 */

import { forge } from "@/lib/client";
import { readResponseError } from "@/lib/api";
import type { components } from "forge-client";

// Type definitions
export type CredentialsListResponse = components["schemas"]["auth.CredentialsListResponse"];

/**
 * Fetches the list of all credentials (passkeys) for the current user.
 * @returns Promise resolving to the credentials list response
 * @throws Error if the request fails
 */
export async function listCredentials(): Promise<CredentialsListResponse> {
  const response = await forge.raw.GET("/api/v1/auth/credentials");

  if (!response.response.ok) {
    const errorMessage = await readResponseError(response, "Failed to list credentials");
    throw new Error(errorMessage);
  }

  return response.data as CredentialsListResponse;
}

/**
 * Renames a credential by updating its device name.
 * @param id - The credential ID to rename
 * @param deviceName - The new device name to set
 * @throws Error if the rename operation fails
 */
export async function renameCredential(id: string, deviceName: string): Promise<void> {
  const requestBody = {
    device_name: deviceName,
  } as unknown as Record<string, never>;

  const requestParams = {
    params: {
      path: { id },
    },
    body: requestBody,
  };

  const response = await forge.raw.PATCH("/api/v1/auth/credentials/{id}", requestParams);

  const isSuccess = response.response.status === 204;

  if (!isSuccess) {
    const errorMessage = await readResponseError(response, "Failed to rename credential");
    throw new Error(errorMessage);
  }
}

/**
 * Deletes a credential from the user's account.
 * @param id - The credential ID to delete
 * @throws Error if the deletion fails
 */
export async function deleteCredential(id: string): Promise<void> {
  const requestParams = {
    params: {
      path: {
        credentialId: id,
      },
    },
  };

  const response = await forge.raw.DELETE("/api/v1/auth/credentials/{credentialId}", requestParams);

  const isSuccess = response.response.status === 204;

  if (!isSuccess) {
    const errorMessage = await readResponseError(response, "Failed to delete credential");
    throw new Error(errorMessage);
  }
}
