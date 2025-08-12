/**
 * Client-side accessor for API base URL.
 * Reads VITE_API_URL first; falls back to http://127.0.0.1:5050 for local dev.
 */
export function getApiBaseUrlClient(): string {
  // Vite exposes import.meta.env at build time on the client
  // @ts-ignore - typing for Vite env
  const fromEnv = (import.meta as any).env?.VITE_API_URL as string | undefined;
  // If no explicit API base is provided, prefer relative same-origin calls during dev
  if (!fromEnv && typeof window !== 'undefined') return '';
  const base = (fromEnv ?? 'http://localhost:5050').trim();
  return base.endsWith('/') ? base.slice(0, -1) : base;
}


