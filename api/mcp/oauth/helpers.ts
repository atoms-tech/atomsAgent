/**
 * OAuth Helper Functions
 *
 * Shared utilities for OAuth operations including encryption,
 * token refresh, and provider-specific configurations.
 */

import crypto from 'crypto';
import { createClient } from '@supabase/supabase-js';

// Environment Variables
const SUPABASE_URL = process.env.SUPABASE_URL!;
const SUPABASE_SERVICE_ROLE_KEY = process.env.SUPABASE_SERVICE_ROLE_KEY!;
const ENCRYPTION_KEY = process.env.OAUTH_ENCRYPTION_KEY!;

// Types
export interface OAuthTokens {
  access_token: string;
  refresh_token?: string;
  expires_at: string;
  token_type: string;
  scope: string;
}

export interface ProviderConfig {
  tokenUrl: string;
  refreshUrl?: string;
  clientId: string;
  clientSecret: string;
  scopes?: string[];
}

export interface RefreshTokenResponse {
  access_token: string;
  refresh_token?: string;
  expires_in?: number;
  token_type?: string;
  scope?: string;
}

// Provider Configurations
export const OAUTH_PROVIDERS: Record<string, ProviderConfig> = {
  github: {
    tokenUrl: 'https://github.com/login/oauth/access_token',
    refreshUrl: 'https://github.com/login/oauth/access_token',
    clientId: process.env.GITHUB_CLIENT_ID!,
    clientSecret: process.env.GITHUB_CLIENT_SECRET!,
  },
  google: {
    tokenUrl: 'https://oauth2.googleapis.com/token',
    refreshUrl: 'https://oauth2.googleapis.com/token',
    clientId: process.env.GOOGLE_CLIENT_ID!,
    clientSecret: process.env.GOOGLE_CLIENT_SECRET!,
  },
  azure: {
    tokenUrl: process.env.AZURE_TOKEN_ENDPOINT!,
    refreshUrl: process.env.AZURE_TOKEN_ENDPOINT!,
    clientId: process.env.AZURE_CLIENT_ID!,
    clientSecret: process.env.AZURE_CLIENT_SECRET!,
  },
  auth0: {
    tokenUrl: process.env.AUTH0_TOKEN_ENDPOINT!,
    refreshUrl: process.env.AUTH0_TOKEN_ENDPOINT!,
    clientId: process.env.AUTH0_CLIENT_ID!,
    clientSecret: process.env.AUTH0_CLIENT_SECRET!,
  },
};

// Encryption/Decryption Functions
export function encrypt(text: string): string {
  if (!ENCRYPTION_KEY || ENCRYPTION_KEY.length !== 64) {
    throw new Error('Invalid encryption key. Must be 32 bytes (64 hex chars)');
  }

  const iv = crypto.randomBytes(16);
  const key = Buffer.from(ENCRYPTION_KEY, 'hex');
  const cipher = crypto.createCipheriv('aes-256-gcm', key, iv);

  let encrypted = cipher.update(text, 'utf8', 'hex');
  encrypted += cipher.final('hex');

  const authTag = cipher.getAuthTag();

  return iv.toString('hex') + ':' + authTag.toString('hex') + ':' + encrypted;
}

export function decrypt(encryptedData: string): string {
  if (!ENCRYPTION_KEY || ENCRYPTION_KEY.length !== 64) {
    throw new Error('Invalid encryption key. Must be 32 bytes (64 hex chars)');
  }

  const parts = encryptedData.split(':');
  if (parts.length !== 3) {
    throw new Error('Invalid encrypted data format');
  }

  const iv = Buffer.from(parts[0], 'hex');
  const authTag = Buffer.from(parts[1], 'hex');
  const encrypted = parts[2];

  const key = Buffer.from(ENCRYPTION_KEY, 'hex');
  const decipher = crypto.createDecipheriv('aes-256-gcm', key, iv);
  decipher.setAuthTag(authTag);

  let decrypted = decipher.update(encrypted, 'hex', 'utf8');
  decrypted += decipher.final('utf8');

  return decrypted;
}

// Token Refresh Functions
export async function refreshAccessToken(
  userId: string,
  mcpName: string,
  provider: string
): Promise<OAuthTokens> {
  const supabase = createClient(SUPABASE_URL, SUPABASE_SERVICE_ROLE_KEY);

  // Get current tokens
  const { data: tokenData, error: fetchError } = await supabase
    .from('mcp_oauth_tokens')
    .select('*')
    .eq('user_id', userId)
    .eq('mcp_name', mcpName)
    .single();

  if (fetchError || !tokenData) {
    throw new Error('OAuth tokens not found');
  }

  // Decrypt refresh token
  if (!tokenData.refresh_token) {
    throw new Error('No refresh token available');
  }

  const refreshToken = decrypt(tokenData.refresh_token);

  // Get provider config
  const config = OAUTH_PROVIDERS[provider];
  if (!config || !config.refreshUrl) {
    throw new Error(`Provider ${provider} does not support token refresh`);
  }

  // Refresh the token
  const params = new URLSearchParams({
    grant_type: 'refresh_token',
    refresh_token: refreshToken,
    client_id: config.clientId,
    client_secret: config.clientSecret,
  });

  const headers: Record<string, string> = {
    'Content-Type': 'application/x-www-form-urlencoded',
  };

  if (provider === 'github') {
    headers['Accept'] = 'application/json';
  }

  const response = await fetch(config.refreshUrl, {
    method: 'POST',
    headers,
    body: params.toString(),
  });

  if (!response.ok) {
    const errorText = await response.text();
    throw new Error(`Token refresh failed: ${response.status} - ${errorText}`);
  }

  const data: RefreshTokenResponse = await response.json();

  if (!data.access_token) {
    throw new Error('No access_token in refresh response');
  }

  // Calculate new expiration
  const expiresAt = data.expires_in
    ? new Date(Date.now() + data.expires_in * 1000)
    : new Date(Date.now() + 3600 * 1000);

  // Encrypt new tokens
  const encryptedAccessToken = encrypt(data.access_token);
  const encryptedRefreshToken = data.refresh_token
    ? encrypt(data.refresh_token)
    : tokenData.refresh_token; // Keep old refresh token if not provided

  // Update tokens in database
  const { error: updateError } = await supabase
    .from('mcp_oauth_tokens')
    .update({
      access_token: encryptedAccessToken,
      refresh_token: encryptedRefreshToken,
      expires_at: expiresAt.toISOString(),
      token_type: data.token_type || tokenData.token_type,
      scope: data.scope || tokenData.scope,
      updated_at: new Date().toISOString(),
    })
    .eq('user_id', userId)
    .eq('mcp_name', mcpName);

  if (updateError) {
    throw new Error(`Failed to update tokens: ${updateError.message}`);
  }

  return {
    access_token: data.access_token,
    refresh_token: data.refresh_token,
    expires_at: expiresAt.toISOString(),
    token_type: data.token_type || 'Bearer',
    scope: data.scope || tokenData.scope,
  };
}

// Get valid access token (refresh if expired)
export async function getValidAccessToken(
  userId: string,
  mcpName: string,
  provider: string
): Promise<string> {
  const supabase = createClient(SUPABASE_URL, SUPABASE_SERVICE_ROLE_KEY);

  // Get current tokens
  const { data: tokenData, error } = await supabase
    .from('mcp_oauth_tokens')
    .select('*')
    .eq('user_id', userId)
    .eq('mcp_name', mcpName)
    .single();

  if (error || !tokenData) {
    throw new Error('OAuth tokens not found');
  }

  // Check if token is expired (with 5-minute buffer)
  const expiresAt = new Date(tokenData.expires_at);
  const now = new Date();
  const bufferTime = 5 * 60 * 1000; // 5 minutes

  if (expiresAt.getTime() - bufferTime < now.getTime()) {
    // Token is expired or about to expire, refresh it
    const refreshedTokens = await refreshAccessToken(userId, mcpName, provider);
    return refreshedTokens.access_token;
  }

  // Token is still valid, decrypt and return
  return decrypt(tokenData.access_token);
}

// Generate PKCE code verifier and challenge
export function generatePKCE(): {
  codeVerifier: string;
  codeChallenge: string;
} {
  // Generate code verifier (43-128 characters)
  const codeVerifier = crypto.randomBytes(32).toString('base64url');

  // Generate code challenge (SHA256 hash of verifier)
  const hash = crypto.createHash('sha256').update(codeVerifier).digest();
  const codeChallenge = hash.toString('base64url');

  return { codeVerifier, codeChallenge };
}

// Generate random state for CSRF protection
export function generateState(): string {
  return crypto.randomBytes(32).toString('base64url');
}

// Store OAuth state in database
export async function storeOAuthState(
  userId: string,
  provider: string,
  mcpName: string,
  state: string,
  codeVerifier: string,
  redirectUri: string
): Promise<void> {
  const supabase = createClient(SUPABASE_URL, SUPABASE_SERVICE_ROLE_KEY);

  // Encrypt code_verifier before storing
  const encryptedCodeVerifier = encrypt(codeVerifier);

  const { error } = await supabase.from('oauth_states').insert({
    user_id: userId,
    provider,
    mcp_name: mcpName,
    state,
    code_verifier: encryptedCodeVerifier,
    redirect_uri: redirectUri,
  });

  if (error) {
    throw new Error(`Failed to store OAuth state: ${error.message}`);
  }
}

// Revoke OAuth tokens
export async function revokeOAuthTokens(
  userId: string,
  mcpName: string,
  provider: string
): Promise<void> {
  const supabase = createClient(SUPABASE_URL, SUPABASE_SERVICE_ROLE_KEY);

  // Get current tokens
  const { data: tokenData, error: fetchError } = await supabase
    .from('mcp_oauth_tokens')
    .select('*')
    .eq('user_id', userId)
    .eq('mcp_name', mcpName)
    .single();

  if (fetchError || !tokenData) {
    // Already revoked or not found
    return;
  }

  // Provider-specific revocation
  const config = OAUTH_PROVIDERS[provider];
  if (!config) {
    throw new Error(`Unsupported provider: ${provider}`);
  }

  // Decrypt access token
  const accessToken = decrypt(tokenData.access_token);

  // Attempt to revoke token with provider (if supported)
  try {
    switch (provider) {
      case 'google':
        await fetch(
          `https://oauth2.googleapis.com/revoke?token=${accessToken}`,
          { method: 'POST' }
        );
        break;
      case 'github':
        // GitHub uses DELETE to revoke
        await fetch(
          `https://api.github.com/applications/${config.clientId}/token`,
          {
            method: 'DELETE',
            headers: {
              Authorization: `Basic ${Buffer.from(
                `${config.clientId}:${config.clientSecret}`
              ).toString('base64')}`,
            },
            body: JSON.stringify({ access_token: accessToken }),
          }
        );
        break;
      // Add other providers as needed
    }
  } catch (err) {
    console.error('Provider revocation failed:', err);
    // Continue to delete from database even if provider revocation fails
  }

  // Delete tokens from database
  const { error: deleteError } = await supabase
    .from('mcp_oauth_tokens')
    .delete()
    .eq('user_id', userId)
    .eq('mcp_name', mcpName);

  if (deleteError) {
    throw new Error(`Failed to delete tokens: ${deleteError.message}`);
  }
}

// Build OAuth authorization URL
export function buildAuthorizationUrl(
  provider: string,
  state: string,
  codeChallenge: string,
  redirectUri: string,
  scopes?: string[]
): string {
  const config = OAUTH_PROVIDERS[provider];
  if (!config) {
    throw new Error(`Unsupported provider: ${provider}`);
  }

  const params = new URLSearchParams({
    client_id: config.clientId,
    redirect_uri: redirectUri,
    state,
    response_type: 'code',
    code_challenge: codeChallenge,
    code_challenge_method: 'S256',
  });

  if (scopes && scopes.length > 0) {
    params.append('scope', scopes.join(' '));
  } else if (config.scopes) {
    params.append('scope', config.scopes.join(' '));
  }

  // Provider-specific authorization URLs
  const authUrls: Record<string, string> = {
    github: 'https://github.com/login/oauth/authorize',
    google: 'https://accounts.google.com/o/oauth2/v2/auth',
    azure: process.env.AZURE_AUTH_ENDPOINT!,
    auth0: process.env.AUTH0_AUTH_ENDPOINT!,
  };

  const authUrl = authUrls[provider];
  if (!authUrl) {
    throw new Error(`No authorization URL for provider: ${provider}`);
  }

  return `${authUrl}?${params.toString()}`;
}
