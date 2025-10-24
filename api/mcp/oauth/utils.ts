import crypto from 'crypto';

/**
 * Generate a cryptographically secure random string
 * @param length - Length of the random string
 * @returns Random string using URL-safe characters
 */
export function generateRandomString(length: number): string {
  const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~';
  let result = '';
  const randomBytes = crypto.randomBytes(length);

  for (let i = 0; i < length; i++) {
    result += chars[randomBytes[i] % chars.length];
  }

  return result;
}

/**
 * Generate a PKCE code verifier
 * @returns Code verifier string (128 characters)
 */
export function generateCodeVerifier(): string {
  // PKCE code verifier: 43-128 characters, we use max length for security
  return generateRandomString(128);
}

/**
 * Generate a PKCE code challenge from a verifier
 * @param verifier - The code verifier
 * @returns Base64url-encoded SHA256 hash of the verifier
 */
export function generateCodeChallenge(verifier: string): string {
  // PKCE code challenge: SHA256 hash of verifier, base64url encoded
  const hash = crypto.createHash('sha256').update(verifier).digest();
  return base64UrlEncode(hash);
}

/**
 * Generate a random state parameter for CSRF protection
 * @returns Random hex string (64 characters)
 */
export function generateState(): string {
  // Generate random state for CSRF protection (32 bytes = 64 hex chars)
  return crypto.randomBytes(32).toString('hex');
}

/**
 * Base64url encode a buffer (RFC 7636)
 * @param buffer - Buffer to encode
 * @returns Base64url-encoded string
 */
export function base64UrlEncode(buffer: Buffer): string {
  return buffer
    .toString('base64')
    .replace(/\+/g, '-')
    .replace(/\//g, '_')
    .replace(/=/g, '');
}

/**
 * Constant-time string comparison to prevent timing attacks
 * @param a - First string
 * @param b - Second string
 * @returns true if strings are equal
 */
export function safeCompare(a: string, b: string): boolean {
  if (a.length !== b.length) {
    return false;
  }

  let result = 0;
  for (let i = 0; i < a.length; i++) {
    result |= a.charCodeAt(i) ^ b.charCodeAt(i);
  }

  return result === 0;
}

/**
 * Encrypt a token for secure storage
 * @param token - Token to encrypt
 * @param key - Encryption key (32 bytes for AES-256)
 * @returns Encrypted token with IV prepended
 */
export function encryptToken(token: string, key: string): string {
  // Ensure key is 32 bytes for AES-256
  const keyBuffer = crypto.createHash('sha256').update(key).digest();

  // Generate random IV
  const iv = crypto.randomBytes(16);

  // Create cipher
  const cipher = crypto.createCipheriv('aes-256-gcm', keyBuffer, iv);

  // Encrypt
  let encrypted = cipher.update(token, 'utf8', 'hex');
  encrypted += cipher.final('hex');

  // Get auth tag
  const authTag = cipher.getAuthTag();

  // Return IV + authTag + encrypted data (all hex-encoded)
  return `${iv.toString('hex')}:${authTag.toString('hex')}:${encrypted}`;
}

/**
 * Decrypt a token from storage
 * @param encryptedToken - Encrypted token with IV prepended
 * @param key - Decryption key (32 bytes for AES-256)
 * @returns Decrypted token
 */
export function decryptToken(encryptedToken: string, key: string): string {
  // Ensure key is 32 bytes for AES-256
  const keyBuffer = crypto.createHash('sha256').update(key).digest();

  // Split encrypted token
  const parts = encryptedToken.split(':');
  if (parts.length !== 3) {
    throw new Error('Invalid encrypted token format');
  }

  const iv = Buffer.from(parts[0], 'hex');
  const authTag = Buffer.from(parts[1], 'hex');
  const encrypted = parts[2];

  // Create decipher
  const decipher = crypto.createDecipheriv('aes-256-gcm', keyBuffer, iv);
  decipher.setAuthTag(authTag);

  // Decrypt
  let decrypted = decipher.update(encrypted, 'hex', 'utf8');
  decrypted += decipher.final('utf8');

  return decrypted;
}

/**
 * Validate URL format
 * @param url - URL to validate
 * @returns true if valid URL
 */
export function isValidUrl(url: string): boolean {
  try {
    new URL(url);
    return true;
  } catch {
    return false;
  }
}

/**
 * Validate redirect URI against allowed origins
 * @param redirectUri - The redirect URI to validate
 * @param allowedOrigins - List of allowed origins
 * @returns true if redirect URI is allowed
 */
export function isAllowedRedirectUri(
  redirectUri: string,
  allowedOrigins: string[]
): boolean {
  try {
    const url = new URL(redirectUri);
    const origin = url.origin;

    return allowedOrigins.some((allowed) => {
      // Support wildcards
      if (allowed.includes('*')) {
        const pattern = allowed.replace(/\*/g, '.*');
        const regex = new RegExp(`^${pattern}$`);
        return regex.test(origin);
      }
      return origin === allowed;
    });
  } catch {
    return false;
  }
}

/**
 * Build OAuth authorization URL
 * @param config - OAuth configuration
 * @returns Complete authorization URL
 */
export function buildAuthorizationUrl(config: {
  authEndpoint: string;
  clientId: string;
  redirectUri: string;
  state: string;
  codeChallenge: string;
  scopes: string[];
  additionalParams?: Record<string, string>;
}): string {
  const url = new URL(config.authEndpoint);

  url.searchParams.set('client_id', config.clientId);
  url.searchParams.set('redirect_uri', config.redirectUri);
  url.searchParams.set('response_type', 'code');
  url.searchParams.set('state', config.state);
  url.searchParams.set('scope', config.scopes.join(' '));
  url.searchParams.set('code_challenge', config.codeChallenge);
  url.searchParams.set('code_challenge_method', 'S256');

  // Add any additional provider-specific parameters
  if (config.additionalParams) {
    Object.entries(config.additionalParams).forEach(([key, value]) => {
      url.searchParams.set(key, value);
    });
  }

  return url.toString();
}

/**
 * Parse error response from OAuth provider
 * @param error - Error response
 * @returns Human-readable error message
 */
export function parseOAuthError(error: any): string {
  if (typeof error === 'string') {
    return error;
  }

  if (error?.error_description) {
    return error.error_description;
  }

  if (error?.error) {
    return error.error;
  }

  if (error?.message) {
    return error.message;
  }

  return 'Unknown OAuth error occurred';
}

/**
 * Sanitize error message for client response
 * @param error - Error message or object
 * @returns Safe error message
 */
export function sanitizeErrorMessage(error: any): string {
  // Don't expose internal errors to clients
  const safeErrors = [
    'Invalid or missing mcp_name',
    'Invalid or missing provider',
    'Invalid or missing redirect_uri',
    'Invalid redirect_uri format',
    'Unsupported provider',
    'Missing or invalid authorization token',
    'Invalid or expired token',
    'Provider is not properly configured',
  ];

  const errorMessage = parseOAuthError(error);

  if (safeErrors.some((safe) => errorMessage.includes(safe))) {
    return errorMessage;
  }

  // Return generic error for unknown/internal errors
  return 'An error occurred. Please try again later.';
}

/**
 * Create CORS headers for responses
 * @param additionalHeaders - Additional headers to include
 * @returns Headers object with CORS headers
 */
export function createCorsHeaders(
  additionalHeaders?: Record<string, string>
): Record<string, string> {
  return {
    'Access-Control-Allow-Origin': '*',
    'Access-Control-Allow-Methods': 'GET, POST, PUT, DELETE, OPTIONS',
    'Access-Control-Allow-Headers': 'Content-Type, Authorization',
    'Access-Control-Max-Age': '86400',
    ...additionalHeaders,
  };
}

/**
 * Create a JSON response with CORS headers
 * @param data - Response data
 * @param status - HTTP status code
 * @returns Response object
 */
export function createJsonResponse(data: any, status: number = 200): Response {
  return new Response(JSON.stringify(data), {
    status,
    headers: createCorsHeaders({ 'Content-Type': 'application/json' }),
  });
}

/**
 * Validate OAuth state expiration
 * @param expiresAt - Expiration timestamp
 * @returns true if state is still valid
 */
export function isStateValid(expiresAt: string): boolean {
  const expiration = new Date(expiresAt);
  const now = new Date();
  return expiration > now;
}

/**
 * Generate OAuth state expiration timestamp
 * @param minutesFromNow - Minutes until expiration (default: 10)
 * @returns ISO timestamp string
 */
export function generateStateExpiration(minutesFromNow: number = 10): string {
  const expiration = new Date(Date.now() + minutesFromNow * 60 * 1000);
  return expiration.toISOString();
}
