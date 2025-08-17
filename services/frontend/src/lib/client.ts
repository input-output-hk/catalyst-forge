// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-ignore - vendored ESM has no types here
import { ForgeClient } from "forge-client";
import { getApiBaseUrl } from "./api";

// Singleton Forge API client for the browser
// - Uses base URL from Vite env/window origin
// - AutoAuth enabled to handle refresh/CSRF via credentials: "include"
// - Adds request/response logging in development

const baseUrl = getApiBaseUrl();

export const forge = new ForgeClient({
  baseUrl,
  autoAuth: true,
  middleware: [],
});

export function getForgeBaseUrl(): string {
  return baseUrl;
}

export async function forgeFetch(path: string, init: RequestInit = {}): Promise<Response> {
  const url = new URL(path, getForgeBaseUrl()).toString();
  const http: typeof fetch = (forge as unknown as { getFetch: () => typeof fetch }).getFetch();
  return http(url, init);
}


