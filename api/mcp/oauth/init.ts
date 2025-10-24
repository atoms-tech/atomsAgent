import { createClient } from '@supabase/supabase-js';
import crypto from 'crypto';

// Types
interface OAuthInitRequest {
  mcp_name: string;
  provider: string;
  redirect_uri: string;
}

interface OAuthInitResponse {
  success: boolean;
  auth_url?: string;
  error?: string;
}

interface ProviderConfig {
  client_id: string;
  client_secret: string;
  auth_endpoint: string;
  token_endpoint: string;
  scopes: string[];
}

// Provider configurations
const PROVIDER_CONFIGS: Record<string, ProviderConfig> = {
  github: {
    client_id: process.env.GITHUB_CLIENT_ID || '',
    client_secret: process.env.GITHUB_CLIENT_SECRET || '',
    auth_endpoint: 'https://github.com/login/oauth/authorize',
    token_endpoint: 'https://github.com/login/oauth/access_token',
    scopes: ['read:user', 'user:email'],
  },
  google: {
    client_id: process.env.GOOGLE_CLIENT_ID || '',
    client_secret: process.env.GOOGLE_CLIENT_SECRET || '',
    auth_endpoint: 'https://accounts.google.com/o/oauth2/v2/auth',
    token_endpoint: 'https://oauth2.googleapis.com/token',
    scopes: ['openid', 'email', 'profile'],
  },
  azure: {
    client_id: process.env.AZURE_CLIENT_ID || '',
    client_secret: process.env.AZURE_CLIENT_SECRET || '',
    auth_endpoint: `https://login.microsoftonline.com/${process.env.AZURE_TENANT_ID || 'common'}/oauth2/v2.0/authorize`,
    token_endpoint: `https://login.microsoftonline.com/${process.env.AZURE_TENANT_ID || 'common'}/oauth2/v2.0/token`,
    scopes: ['openid', 'email', 'profile'],
  },
  auth0: {
    client_id: process.env.AUTH0_CLIENT_ID || '',
    client_secret: process.env.AUTH0_CLIENT_SECRET || '',
    auth_endpoint: `https://${process.env.AUTH0_DOMAIN || ''}/authorize`,
    token_endpoint: `https://${process.env.AUTH0_DOMAIN || ''}/oauth/token`,
    scopes: ['openid', 'email', 'profile'],
  },
};

// PKCE utility functions
function generateRandomString(length: number): string {
  const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~';
  let result = '';
  const randomBytes = crypto.randomBytes(length);

  for (let i = 0; i < length; i++) {
    result += chars[randomBytes[i] % chars.length];
  }

  return result;
}

function generateCodeVerifier(): string {
  // PKCE code verifier: 43-128 characters
  return generateRandomString(128);
}

function generateCodeChallenge(verifier: string): string {
  // PKCE code challenge: SHA256 hash of verifier, base64url encoded
  const hash = crypto.createHash('sha256').update(verifier).digest();
  return hash
    .toString('base64')
    .replace(/\+/g, '-')
    .replace(/\//g, '_')
    .replace(/=/g, '');
}

function generateState(): string {
  // Generate random state for CSRF protection (32 bytes = 64 hex chars)
  return crypto.randomBytes(32).toString('hex');
}

// Supabase client initialization
function getSupabaseClient() {
  const supabaseUrl = process.env.SUPABASE_URL;
  const supabaseServiceKey = process.env.SUPABASE_SERVICE_ROLE_KEY;

  if (!supabaseUrl || !supabaseServiceKey) {
    throw new Error('Missing Supabase configuration');
  }

  return createClient(supabaseUrl, supabaseServiceKey);
}

// Validation functions
function validateRequest(body: any): { valid: boolean; error?: string } {
  if (!body.mcp_name || typeof body.mcp_name !== 'string') {
    return { valid: false, error: 'Invalid or missing mcp_name' };
  }

  if (!body.provider || typeof body.provider !== 'string') {
    return { valid: false, error: 'Invalid or missing provider' };
  }

  if (!body.redirect_uri || typeof body.redirect_uri !== 'string') {
    return { valid: false, error: 'Invalid or missing redirect_uri' };
  }

  // Validate redirect_uri format
  try {
    new URL(body.redirect_uri);
  } catch {
    return { valid: false, error: 'Invalid redirect_uri format' };
  }

  return { valid: true };
}

// Main handler function
export default async function handler(req: Request): Promise<Response> {
  // Enable CORS
  const corsHeaders = {
    'Access-Control-Allow-Origin': '*',
    'Access-Control-Allow-Methods': 'POST, OPTIONS',
    'Access-Control-Allow-Headers': 'Content-Type, Authorization',
  };

  // Handle OPTIONS request
  if (req.method === 'OPTIONS') {
    return new Response(null, { status: 204, headers: corsHeaders });
  }

  // Only allow POST
  if (req.method !== 'POST') {
    return new Response(
      JSON.stringify({ success: false, error: 'Method not allowed' }),
      {
        status: 405,
        headers: { ...corsHeaders, 'Content-Type': 'application/json' }
      }
    );
  }

  try {
    // Parse request body
    let body: OAuthInitRequest;
    try {
      body = await req.json();
    } catch {
      return new Response(
        JSON.stringify({ success: false, error: 'Invalid JSON body' }),
        {
          status: 400,
          headers: { ...corsHeaders, 'Content-Type': 'application/json' }
        }
      );
    }

    // Validate request
    const validation = validateRequest(body);
    if (!validation.valid) {
      return new Response(
        JSON.stringify({ success: false, error: validation.error }),
        {
          status: 400,
          headers: { ...corsHeaders, 'Content-Type': 'application/json' }
        }
      );
    }

    const { mcp_name, provider, redirect_uri } = body;

    // Validate provider
    if (!PROVIDER_CONFIGS[provider]) {
      return new Response(
        JSON.stringify({
          success: false,
          error: `Unsupported provider: ${provider}. Supported providers: ${Object.keys(PROVIDER_CONFIGS).join(', ')}`
        }),
        {
          status: 400,
          headers: { ...corsHeaders, 'Content-Type': 'application/json' }
        }
      );
    }

    const providerConfig = PROVIDER_CONFIGS[provider];

    // Validate provider environment variables
    if (!providerConfig.client_id || !providerConfig.client_secret) {
      return new Response(
        JSON.stringify({
          success: false,
          error: `Provider ${provider} is not properly configured. Missing client credentials.`
        }),
        {
          status: 500,
          headers: { ...corsHeaders, 'Content-Type': 'application/json' }
        }
      );
    }

    // Get user ID from auth header
    const authHeader = req.headers.get('Authorization');
    if (!authHeader || !authHeader.startsWith('Bearer ')) {
      return new Response(
        JSON.stringify({ success: false, error: 'Missing or invalid authorization token' }),
        {
          status: 401,
          headers: { ...corsHeaders, 'Content-Type': 'application/json' }
        }
      );
    }

    const token = authHeader.substring(7);

    // Initialize Supabase client
    const supabase = getSupabaseClient();

    // Verify user token and get user ID
    const { data: { user }, error: authError } = await supabase.auth.getUser(token);

    if (authError || !user) {
      return new Response(
        JSON.stringify({ success: false, error: 'Invalid or expired token' }),
        {
          status: 401,
          headers: { ...corsHeaders, 'Content-Type': 'application/json' }
        }
      );
    }

    const user_id = user.id;

    // Generate PKCE parameters
    const codeVerifier = generateCodeVerifier();
    const codeChallenge = generateCodeChallenge(codeVerifier);
    const state = generateState();

    // Store state and code verifier in Supabase
    // First, create oauth_state table if it doesn't exist (this should be in migrations)
    const { error: insertError } = await supabase
      .from('oauth_state')
      .insert({
        user_id,
        state,
        code_verifier: codeVerifier,
        provider,
        mcp_name,
        redirect_uri,
        created_at: new Date().toISOString(),
        expires_at: new Date(Date.now() + 10 * 60 * 1000).toISOString(), // 10 minutes
      });

    if (insertError) {
      console.error('Failed to store OAuth state:', insertError);
      return new Response(
        JSON.stringify({
          success: false,
          error: 'Failed to initialize OAuth flow. Please try again.'
        }),
        {
          status: 500,
          headers: { ...corsHeaders, 'Content-Type': 'application/json' }
        }
      );
    }

    // Build authorization URL
    const authUrl = new URL(providerConfig.auth_endpoint);
    authUrl.searchParams.set('client_id', providerConfig.client_id);
    authUrl.searchParams.set('redirect_uri', redirect_uri);
    authUrl.searchParams.set('response_type', 'code');
    authUrl.searchParams.set('state', state);
    authUrl.searchParams.set('scope', providerConfig.scopes.join(' '));

    // Add PKCE parameters
    authUrl.searchParams.set('code_challenge', codeChallenge);
    authUrl.searchParams.set('code_challenge_method', 'S256');

    // Provider-specific parameters
    if (provider === 'google') {
      authUrl.searchParams.set('access_type', 'offline');
      authUrl.searchParams.set('prompt', 'consent');
    }

    if (provider === 'azure') {
      authUrl.searchParams.set('response_mode', 'query');
    }

    if (provider === 'auth0') {
      authUrl.searchParams.set('audience', process.env.AUTH0_AUDIENCE || '');
    }

    // Return authorization URL
    return new Response(
      JSON.stringify({
        success: true,
        auth_url: authUrl.toString()
      }),
      {
        status: 200,
        headers: { ...corsHeaders, 'Content-Type': 'application/json' }
      }
    );

  } catch (error) {
    console.error('OAuth initialization error:', error);

    return new Response(
      JSON.stringify({
        success: false,
        error: 'Internal server error. Please try again later.'
      }),
      {
        status: 500,
        headers: {
          'Access-Control-Allow-Origin': '*',
          'Content-Type': 'application/json'
        }
      }
    );
  }
}

// Vercel Edge Function config
export const config = {
  runtime: 'edge',
};
