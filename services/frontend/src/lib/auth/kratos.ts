/**
 * Helpers for reading the Kratos public base URL from runtime configuration.
 * Falls back to build-time env or reasonable defaults for local dev.
 */

type RuntimeConfig = {
  apiBaseUrl?: string;
  kratosPublicUrl?: string;
};

function getRuntimeConfig(): RuntimeConfig {
  try {
    const w = window as unknown as { __CF_CONFIG__?: RuntimeConfig };
    return w.__CF_CONFIG__ ?? {};
  } catch {
    return {};
  }
}

/**
 * Returns the Kratos public base URL.
 * Priority:
 * 1) window.__CF_CONFIG__.kratosPublicUrl (runtime)
 * 2) import.meta.env.VITE_KRATOS_PUBLIC_URL (build-time)
 * 3) default "/.ory/kratos/public" (reverse-proxied path)
 */
export function getKratosPublicBaseUrl(): string {
  const runtime = getRuntimeConfig();
  if (runtime.kratosPublicUrl && runtime.kratosPublicUrl.length > 0) {
    return runtime.kratosPublicUrl;
  }

  const importMeta = import.meta as unknown as {
    env?: { VITE_KRATOS_PUBLIC_URL?: string };
  };
  const envUrl = importMeta.env?.VITE_KRATOS_PUBLIC_URL;
  if (envUrl && envUrl.length > 0) {
    return envUrl;
  }

  return "/.ory/kratos/public";
}


