import createClient, { type Middleware, type Client } from 'openapi-fetch';
import { createAutoAuthFetch, InMemoryTokenStore, type TokenStore } from './autoAuth';
import type { paths } from './schema';

export type ForgeClientOptions = {
  baseUrl: string;
  headers?: Record<string, string>;
  fetch?: typeof fetch;
  middleware?: Middleware[];
  autoAuth?: boolean; // default true in browser
  tokenStore?: TokenStore; // used when autoAuth is enabled
};

export type AuthProvider = {
  middleware: Middleware;
};

/**
 * Bearer token authentication provider
 */
export class BearerTokenProvider implements AuthProvider {
  constructor(private token: string) { }

  get middleware(): Middleware {
    const token = this.token;
    return {
      async onRequest({ request }) {
        request.headers.set('Authorization', `Bearer ${token}`);
        return request;
      },
    };
  }

  /**
   * Create from environment variable
   */
  static fromEnv(envVar = 'Forge_API_TOKEN'): BearerTokenProvider {
    const token = process.env[envVar] || process.env.Forge_TOKEN;
    if (!token) {
      throw new Error(`Environment variable ${envVar} or Forge_TOKEN not set`);
    }
    return new BearerTokenProvider(token);
  }
}

/**
 * API key authentication provider
 */
export class ApiKeyProvider implements AuthProvider {
  constructor(
    private apiKey: string,
    private headerName = 'X-API-Key'
  ) { }

  get middleware(): Middleware {
    const headerName = this.headerName;
    const apiKey = this.apiKey;
    return {
      async onRequest({ request }) {
        request.headers.set(headerName, apiKey);
        return request;
      },
    };
  }

  /**
   * Create from environment variable
   */
  static fromEnv(envVar = 'Forge_API_KEY', headerName = 'X-API-Key'): ApiKeyProvider {
    const apiKey = process.env[envVar];
    if (!apiKey) {
      throw new Error(`Environment variable ${envVar} not set`);
    }
    return new ApiKeyProvider(apiKey, headerName);
  }
}

/**
 * Basic authentication provider
 */
export class BasicAuthProvider implements AuthProvider {
  private credentials: string;

  constructor(username: string, password: string) {
    this.credentials = btoa(`${username}:${password}`);
  }

  get middleware(): Middleware {
    const credentials = this.credentials;
    return {
      async onRequest({ request }) {
        request.headers.set('Authorization', `Basic ${credentials}`);
        return request;
      },
    };
  }
}

/**
 * Error handling middleware that provides better error messages
 */
export const errorHandlingMiddleware: Middleware = {
  async onResponse({ response }) {
    if (!response.ok) {
      const contentType = response.headers.get('content-type');
      let errorMessage = `HTTP ${response.status}: ${response.statusText}`;

      if (contentType?.includes('application/json')) {
        try {
          const errorBody = await response.clone().json();
          if (errorBody.error || errorBody.message) {
            errorMessage = errorBody.error || errorBody.message;
          }
        } catch {
          // If JSON parsing fails, use default message
        }
      }

      // Attach additional info to the response
      (response as any).errorMessage = errorMessage;
    }
    return response;
  },
};

/**
 * Logging middleware for debugging
 */
export const loggingMiddleware: Middleware = {
  async onRequest({ request }) {
    console.log(`[Forge API] ${request.method} ${request.url}`);
    return request;
  },
  async onResponse({ response, request }) {
    console.log(`[Forge API] ${request.method} ${request.url} -> ${response.status}`);
    return response;
  },
};

/**
 * Main Forge API client class
 */
export class ForgeClient {
  private client: Client<paths>;
  private tokenStore?: TokenStore;
  private httpFetch: typeof fetch;
  private baseUrl: string;

  constructor(options: ForgeClientOptions) {
    const middleware = options.middleware || [];

    // Always add error handling middleware
    middleware.push(errorHandlingMiddleware);

    // Auto-auth fetch wrapper (browser-only by default)
    const enableAutoAuth = options.autoAuth ?? (typeof window !== 'undefined');
    // Token store is optional now; when omitted we rely on cookies only
    const tokenStore = options.tokenStore;
    this.tokenStore = enableAutoAuth ? tokenStore : undefined;

    const wrappedFetch = enableAutoAuth
      ? createAutoAuthFetch(options.fetch ?? fetch, {
        baseUrl: options.baseUrl,
        tokenStore,
        credentials: 'include',
        // Refresh on any 401 (except the refresh call itself). This supports cookie-based auth
        // where no Authorization header is present but a refresh should still occur on expiry.
        refreshPolicy: 'always',
      })
      : (options.fetch ?? fetch);

    this.client = createClient<paths>({
      baseUrl: options.baseUrl,
      headers: options.headers,
      fetch: wrappedFetch,
      // @ts-ignore - openapi-fetch middleware types are a bit different
      middleware,
    });

    this.httpFetch = wrappedFetch;
    this.baseUrl = options.baseUrl;
  }

  /**
   * Get the underlying openapi-fetch client for direct use
   */
  get raw(): Client<paths> {
    return this.client;
  }

  /**
   * Get the fetch function used by this client (includes AutoAuth if enabled)
   */
  getFetch(): typeof fetch {
    return this.httpFetch;
  }

  /**
   * Get the base URL configured for this client
   */
  getBaseUrl(): string {
    return this.baseUrl;
  }

  /**
   * Create client with bearer token authentication
   */
  static withBearerToken(
    baseUrl: string,
    token: string,
    options?: Partial<ForgeClientOptions>
  ): ForgeClient {
    const authProvider = new BearerTokenProvider(token);
    return new ForgeClient({
      baseUrl,
      ...options,
      middleware: [authProvider.middleware, ...(options?.middleware || [])],
    });
  }

  /**
   * Create client with API key authentication
   */
  static withApiKey(
    baseUrl: string,
    apiKey: string,
    headerName = 'X-API-Key',
    options?: Partial<ForgeClientOptions>
  ): ForgeClient {
    const authProvider = new ApiKeyProvider(apiKey, headerName);
    return new ForgeClient({
      baseUrl,
      ...options,
      middleware: [authProvider.middleware, ...(options?.middleware || [])],
    });
  }

  /**
   * Create client with basic authentication
   */
  static withBasicAuth(
    baseUrl: string,
    username: string,
    password: string,
    options?: Partial<ForgeClientOptions>
  ): ForgeClient {
    const authProvider = new BasicAuthProvider(username, password);
    return new ForgeClient({
      baseUrl,
      ...options,
      middleware: [authProvider.middleware, ...(options?.middleware || [])],
    });
  }

  /**
   * Create client from environment variables
   */
  static fromEnv(options?: Partial<ForgeClientOptions>): ForgeClient {
    const baseUrl = process.env.Forge_API_URL || 'https://api.Forge.example.com';

    // Try different auth methods in order of preference
    if (process.env.Forge_API_TOKEN || process.env.Forge_TOKEN) {
      const authProvider = BearerTokenProvider.fromEnv();
      return new ForgeClient({
        baseUrl,
        ...options,
        middleware: [authProvider.middleware, ...(options?.middleware || [])],
      });
    }

    if (process.env.Forge_API_KEY) {
      const authProvider = ApiKeyProvider.fromEnv();
      return new ForgeClient({
        baseUrl,
        ...options,
        middleware: [authProvider.middleware, ...(options?.middleware || [])],
      });
    }

    // No auth configured
    return new ForgeClient({
      baseUrl,
      ...options,
    });
  }

  /**
   * Perform health check
   */
  async healthCheck() {
    const response = await this.client.GET('/healthz');
    if (!response.response.ok) {
      throw new Error(`Health check failed: ${response.response.status}`);
    }
    return response.data;
  }

  /**
   * Helper to check if an error response is a specific status code
   */
  static isStatus(error: unknown, status: number): boolean {
    if (error && typeof error === 'object' && 'response' in error) {
      const response = (error as any).response;
      return response?.status === status;
    }
    return false;
  }

  static isUnauthorized(error: unknown): boolean {
    return this.isStatus(error, 401);
  }

  static isForbidden(error: unknown): boolean {
    return this.isStatus(error, 403);
  }

  static isNotFound(error: unknown): boolean {
    return this.isStatus(error, 404);
  }

  static isConflict(error: unknown): boolean {
    return this.isStatus(error, 409);
  }

  static isServerError(error: unknown): boolean {
    if (error && typeof error === 'object' && 'response' in error) {
      const response = (error as any).response;
      return response?.status >= 500 && response?.status < 600;
    }
    return false;
  }

  /**
   * Set the current access token (used by AutoAuth wrapper)
   */
  setAccessToken(token: string) {
    this.tokenStore?.set(token);
  }

  /**
   * Get the current access token
   */
  getAccessToken(): string | null {
    return this.tokenStore?.get() ?? null;
  }
}
