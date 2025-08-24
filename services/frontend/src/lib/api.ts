

/**
 * Gets the base URL for API requests.
 * Uses VITE_API_URL environment variable if set, otherwise uses current origin.
 * @returns The API base URL
 */
export function getApiBaseUrl(): string {
  // 1) Runtime config from window.__CF_CONFIG__
  try {
    const w = window as unknown as { __CF_CONFIG__?: { apiBaseUrl?: string } };
    const runtimeUrl = w.__CF_CONFIG__?.apiBaseUrl;
    if (runtimeUrl && runtimeUrl.length > 0) return runtimeUrl;
  } catch {
    // ignore
  }

  // 2) Build-time Vite env
  const importMeta = import.meta as unknown as {
    env?: {
      VITE_API_URL?: string;
    };
  };
  const envUrl = importMeta.env?.VITE_API_URL;
  if (envUrl && envUrl.length > 0) return envUrl;

  // 3) Fallback to current origin
  return window.location.origin;
}

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
