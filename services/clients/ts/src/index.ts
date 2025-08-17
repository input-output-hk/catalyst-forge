/**
 * Forge TypeScript Client
 *
 * A fully-typed TypeScript client for the Forge API, auto-generated from OpenAPI specifications.
 */

// Main client and auth providers
export {
  ForgeClient,
  BearerTokenProvider,
  ApiKeyProvider,
  BasicAuthProvider,
  errorHandlingMiddleware,
  loggingMiddleware,
} from './api/client';

// Types
export type { ForgeClientOptions as ForgeClientOptions, AuthProvider } from './api/client';

// Export all OpenAPI types
export type { paths, components, operations } from './api/schema';

// Re-export openapi-fetch types for advanced usage
export type { Client, Middleware } from 'openapi-fetch';

// Auto-auth exports
export { createAutoAuthFetch, InMemoryTokenStore } from './api/autoAuth';
export type { TokenStore } from './api/autoAuth';

// WebAuthn helpers
export * as webauthn from './api/webauthn';
