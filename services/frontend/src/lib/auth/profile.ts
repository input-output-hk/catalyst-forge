/**
 * User profile management utilities.
 *
 * This module provides functions to update user profile information
 * such as display name and other account details.
 */

import { forge } from "@/lib/client";
import { readResponseError } from "@/lib/api";

/**
 * Updates the user's full name/display name.
 * @param fullName - The new full name to set for the user
 * @throws Error if the update operation fails
 */
export async function updateFullName(fullName: string): Promise<void> {
  // Prepare the request body with the new name
  const requestBody = {
    full_name: fullName,
  } as unknown as Record<string, never>;

  // Send the update request
  const response = await forge.raw.PATCH("/api/v1/auth/me", {
    body: requestBody,
  });

  // Check if the update was successful
  const isSuccess = response.response.status === 204;

  if (!isSuccess) {
    const errorMessage = await readResponseError(response, "Failed to update profile");
    throw new Error(errorMessage);
  }
}
