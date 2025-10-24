/**
 * OAuth provider types
 */
export type OAuthProvider = 'github' | 'google' | 'azure' | 'auth0';

/**
 * Request body for OAuth initialization endpoint
 */
export interface OAuthInitRequest {
  mcp_name: string;
  provider: OAuthProvider;
  redirect_uri: string;
}

/**
 * Response from OAuth initialization endpoint
 */
export interface OAuthInitResponse {
  success: boolean;
  auth_url?: string;
  error?: string;
}

/**
 * OAuth provider configuration
 */
export interface ProviderConfig {
  client_id: string;
  client_secret: string;
  auth_endpoint: string;
  token_endpoint: string;
  scopes: string[];
  additional_params?: Record<string, string>;
}

/**
 * OAuth state stored in database
 */
export interface OAuthState {
  id: string;
  user_id: string;
  state: string;
  code_verifier: string;
  provider: OAuthProvider;
  mcp_name: string;
  redirect_uri: string;
  created_at: string;
  expires_at: string;
  used: boolean;
}

/**
 * Request body for OAuth callback endpoint
 */
export interface OAuthCallbackRequest {
  code: string;
  state: string;
}

/**
 * Response from OAuth callback endpoint
 */
export interface OAuthCallbackResponse {
  success: boolean;
  mcp_config_id?: string;
  error?: string;
}

/**
 * OAuth token response from provider
 */
export interface OAuthTokenResponse {
  access_token: string;
  token_type: string;
  expires_in?: number;
  refresh_token?: string;
  scope?: string;
  id_token?: string;
}

/**
 * MCP configuration with OAuth credentials
 */
export interface MCPConfig {
  id: string;
  name: string;
  type: 'http' | 'sse' | 'stdio';
  auth_type: 'bearer' | 'apikey' | 'oauth' | 'none';
  endpoint: string;
  config: {
    access_token?: string;
    refresh_token?: string;
    token_expires_at?: string;
    provider?: OAuthProvider;
    [key: string]: any;
  };
  scope: 'global' | 'org' | 'user';
  org_id?: string;
  user_id: string;
  status: 'pending' | 'approved' | 'rejected' | 'active';
  created_by: string;
  created_at: string;
  updated_at: string;
}

/**
 * Error types for OAuth flows
 */
export enum OAuthErrorType {
  INVALID_REQUEST = 'invalid_request',
  INVALID_STATE = 'invalid_state',
  EXPIRED_STATE = 'expired_state',
  INVALID_CODE = 'invalid_code',
  TOKEN_EXCHANGE_FAILED = 'token_exchange_failed',
  PROVIDER_ERROR = 'provider_error',
  DATABASE_ERROR = 'database_error',
  MISSING_CONFIG = 'missing_config',
  UNAUTHORIZED = 'unauthorized',
}

/**
 * OAuth error response
 */
export interface OAuthError {
  type: OAuthErrorType;
  message: string;
  details?: any;
}

/**
 * Token refresh request
 */
export interface TokenRefreshRequest {
  mcp_config_id: string;
}

/**
 * Token refresh response
 */
export interface TokenRefreshResponse {
  success: boolean;
  expires_at?: string;
  error?: string;
}

/**
 * Token revocation request
 */
export interface TokenRevocationRequest {
  mcp_config_id: string;
}

/**
 * Token revocation response
 */
export interface TokenRevocationResponse {
  success: boolean;
  error?: string;
}
