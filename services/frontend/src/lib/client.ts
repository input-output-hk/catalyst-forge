/**
 * Forge API client singleton for the frontend application.
 *
 * This module provides:
 * - A configured ForgeClient instance for API communication
 * - Helper functions for making raw HTTP requests
 * - Automatic authentication handling via cookies
 *
 * Features:
 * - Uses base URL from Vite environment or window origin
 * - AutoAuth enabled to handle refresh/CSRF via credentials: "include"
 * - Middleware support for request/response interceptors
 */

// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-ignore - vendored ESM has no types here
import { ForgeClient } from "forge-client";
import { getApiBaseUrl } from "./api";

// Type definitions for internal Forge client methods
interface ForgeClientWithInternals {
  getFetch: () => typeof fetch;
}

// Initialize base URL once at module load
const baseUrl = getApiBaseUrl();

/**
 * Singleton Forge API client instance.
 * Configured with automatic authentication and CSRF handling.
 */
export const forge = new ForgeClient({
  baseUrl,
  autoAuth: true,
  middleware: [],
});

/**
 * Gets the base URL used by the Forge client.
 * @returns The API base URL string
 */
export function getForgeBaseUrl(): string {
  return baseUrl;
}

/**
 * Makes a raw HTTP request using the Forge client's fetch implementation.
 * This bypasses the typed API methods but still uses the client's auth handling.
 * @param path - The API path to request
 * @param init - Standard fetch RequestInit options
 * @returns Promise resolving to the Response object
 */
export async function forgeFetch(path: string, init: RequestInit = {}): Promise<Response> {
  // Build the full URL from path and base
  const fullUrl = new URL(path, getForgeBaseUrl());
  const urlString = fullUrl.toString();

  // Extract the internal fetch function from Forge client
  const forgeWithInternals = forge as unknown as ForgeClientWithInternals;
  const httpClient = forgeWithInternals.getFetch();

  // Make the request using Forge's configured fetch
  return httpClient(urlString, init);
}
