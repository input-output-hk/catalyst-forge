/**
 * Lightweight API client utilities for the frontend.
 *
 * Features:
 * - Uses base URL from VITE_API_URL (defaults to window.origin)
 * - Keeps access token in memory only for security
 * - Sends credentials to include cookies for refresh/CSRF flows
 * - Provides error handling utilities
 */

// Access token storage (in-memory only for security)
let accessToken: string | null = null;

/**
 * Sets the access token for API requests.
 * Token is stored in memory only, not persisted to storage.
 * @param token - The access token to set, or null to clear
 */
export function setAccessToken(token: string | null): void {
  accessToken = token;
}

/**
 * Gets the current access token.
 * @returns The current access token or null if not set
 */
export function getAccessToken(): string | null {
  return accessToken;
}

/**
 * Gets the base URL for API requests.
 * Uses VITE_API_URL environment variable if set, otherwise uses current origin.
 * @returns The API base URL
 */
export function getApiBaseUrl(): string {
  const importMeta = import.meta as unknown as {
    env?: {
      VITE_API_URL?: string;
    };
  };

  const envUrl = importMeta.env?.VITE_API_URL;
  const hasEnvUrl = envUrl && envUrl.length > 0;

  return hasEnvUrl ? envUrl : window.location.origin;
}

/**
 * Enhanced fetch function for API requests.
 * Automatically includes authorization header and credentials.
 * @param input - The resource to fetch
 * @param init - Request initialization options
 * @returns Promise resolving to the Response
 */
export async function apiFetch(
  input: RequestInfo | URL,
  init: RequestInit = {}
): Promise<Response> {
  const headers = new Headers(init.headers);

  // Add authorization header if token is available
  if (accessToken) {
    headers.set("Authorization", `Bearer ${accessToken}`);
  }

  // Always include credentials for cookie-based auth
  const requestOptions: RequestInit = {
    ...init,
    headers,
    credentials: "include",
  };

  return fetch(input, requestOptions);
}

/**
 * Reads a cookie value by name.
 * @param name - The name of the cookie to read
 * @returns The cookie value or null if not found
 */
export function readCookie(name: string): string | null {
  const cookieString = document.cookie;
  const cookies = cookieString ? cookieString.split("; ") : [];

  for (const cookie of cookies) {
    const equalIndex = cookie.indexOf("=");
    const hasValue = equalIndex > -1;

    const cookieName = hasValue ? cookie.substring(0, equalIndex) : cookie;

    const decodedName = decodeURIComponent(cookieName);

    if (decodedName === name) {
      const cookieValue = hasValue ? cookie.substring(equalIndex + 1) : "";

      return decodeURIComponent(cookieValue);
    }
  }

  return null;
}

/**
 * Refreshes the access token using the refresh token cookie.
 * Handles CSRF token requirement and retries briefly if cookie is not yet visible.
 * @returns The new access token if successful, null otherwise
 */
export async function refreshAccessToken(): Promise<string | null> {
  try {
    // Wait briefly for CSRF cookie to be visible if it was just set by the server
    const csrfToken = await waitForCsrfToken();

    // Build headers with CSRF token if available
    const headers: HeadersInit = {
      "content-type": "application/json",
    };

    if (csrfToken) {
      headers["X-CSRF-Token"] = csrfToken;
    }

    // Send refresh request
    const refreshUrl = new URL("/api/v1/auth/refresh", getApiBaseUrl()).toString();
    const response = await apiFetch(refreshUrl, {
      method: "POST",
      headers,
      body: "{}",
    });

    if (!response.ok) {
      return null;
    }

    // Parse response and extract token
    const responseData = (await response.json().catch(() => null)) as {
      access_token?: string;
    } | null;

    if (!responseData || typeof responseData.access_token !== "string") {
      return null;
    }

    // Store and return the new token
    const newToken = responseData.access_token;
    setAccessToken(newToken);

    return newToken;
  } catch {
    return null;
  }
}

/**
 * Waits for CSRF token to be available in cookies.
 * Retries a few times with short delays for race conditions.
 * @returns The CSRF token if found, null otherwise
 */
async function waitForCsrfToken(): Promise<string | null> {
  const maxRetries = 3;
  const retryDelay = 100; // milliseconds

  for (let attempt = 0; attempt < maxRetries; attempt++) {
    const csrfToken = readCookie("__Host-csrf_token");

    if (csrfToken) {
      return csrfToken;
    }

    // Wait before next attempt (except on last attempt)
    if (attempt < maxRetries - 1) {
      await new Promise((resolve) => setTimeout(resolve, retryDelay));
    }
  }

  return null;
}

/**
 * Logs out the user everywhere by invalidating all sessions.
 * Clears the local access token and attempts to invalidate server-side sessions.
 */
export async function logoutEverywhere(): Promise<void> {
  const logoutUrl = new URL("/api/v1/auth/logout", getApiBaseUrl()).toString();

  // Best-effort invalidate refresh cookie on server
  try {
    await apiFetch(logoutUrl, {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: "{}",
    });
  } catch {
    // Ignore errors - logout should always clear local state
  }

  // Clear local access token
  setAccessToken(null);
}

// Error handling utilities

/**
 * Extracts a user-friendly error message from various error types.
 * @param error - The error to extract message from (can be Error, string, or unknown)
 * @param defaultMessage - Fallback message if no message can be extracted
 * @returns A user-friendly error message
 */
export function extractErrorMessage(
  error: unknown,
  defaultMessage = "An unexpected error occurred."
): string {
  // Check for Error object with message property
  if (error && typeof error === "object" && "message" in error) {
    const errorWithMessage = error as { message?: string };
    const message = errorWithMessage.message;

    if (message && typeof message === "string" && message.length > 0) {
      return message;
    }
  }

  // Check for string error
  if (typeof error === "string" && error.length > 0) {
    return error;
  }

  // Try to convert to string
  try {
    const stringified = String(error);
    return stringified || defaultMessage;
  } catch {
    return defaultMessage;
  }
}

/**
 * Reads and extracts error message from an HTTP response.
 * Attempts to read response body as text, falling back to status code.
 * @param res - Response object or object containing a response property
 * @param defaultMessage - Fallback message if response body is empty
 * @returns Error message from response body or status code
 */
export async function readResponseError(
  res: Response | { response: Response },
  defaultMessage = ""
): Promise<string> {
  // Extract Response object from various formats
  const response = res instanceof Response ? res : res.response;

  try {
    const responseText = await response.text();

    // Return response text if available, otherwise use default or status
    if (responseText && responseText.length > 0) {
      return responseText;
    }

    return defaultMessage || `${response.status}`;
  } catch {
    // If reading response fails, return default or status
    return defaultMessage || `${response.status}`;
  }
}
