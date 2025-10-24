/**
 * OAuth Initiation Endpoint
 *
 * Initiates OAuth flow by generating state, PKCE parameters,
 * and redirecting to the provider's authorization endpoint.
 */

import type { VercelRequest, VercelResponse } from '@vercel/node';
import { createClient } from '@supabase/supabase-js';
import {
  generatePKCE,
  generateState,
  storeOAuthState,
  buildAuthorizationUrl,
} from './helpers';

const SUPABASE_URL = process.env.SUPABASE_URL!;
const SUPABASE_ANON_KEY = process.env.SUPABASE_ANON_KEY!;
const API_BASE_URL = process.env.VERCEL_URL
  ? `https://${process.env.VERCEL_URL}`
  : 'http://localhost:3000';

interface InitiateRequest {
  provider: string;
  mcp_name: string;
  scopes?: string[];
}

export default async function handler(
  req: VercelRequest,
  res: VercelResponse
) {
  // Only accept POST requests
  if (req.method !== 'POST') {
    return res.status(405).json({
      error: 'Method not allowed',
      message: 'This endpoint only accepts POST requests',
    });
  }

  try {
    // Get user from Supabase auth
    const authHeader = req.headers.authorization;
    if (!authHeader || !authHeader.startsWith('Bearer ')) {
      return res.status(401).json({
        error: 'Unauthorized',
        message: 'Missing or invalid authorization header',
      });
    }

    const token = authHeader.substring(7);
    const supabase = createClient(SUPABASE_URL, SUPABASE_ANON_KEY);

    const {
      data: { user },
      error: authError,
    } = await supabase.auth.getUser(token);

    if (authError || !user) {
      return res.status(401).json({
        error: 'Unauthorized',
        message: 'Invalid authentication token',
      });
    }

    // Parse request body
    const { provider, mcp_name, scopes }: InitiateRequest = req.body;

    // Validate required fields
    if (!provider || !mcp_name) {
      return res.status(400).json({
        error: 'Invalid request',
        message: 'Missing required fields: provider and mcp_name',
      });
    }

    // Validate provider
    const validProviders = ['github', 'google', 'azure', 'auth0'];
    if (!validProviders.includes(provider)) {
      return res.status(400).json({
        error: 'Invalid provider',
        message: `Provider must be one of: ${validProviders.join(', ')}`,
      });
    }

    // Generate PKCE parameters
    const { codeVerifier, codeChallenge } = generatePKCE();

    // Generate state for CSRF protection
    const state = generateState();

    // Build redirect URI
    const redirectUri = `${API_BASE_URL}/api/mcp/oauth/callback`;

    // Store state and PKCE parameters in database
    await storeOAuthState(
      user.id,
      provider,
      mcp_name,
      state,
      codeVerifier,
      redirectUri
    );

    // Build authorization URL
    const authUrl = buildAuthorizationUrl(
      provider,
      state,
      codeChallenge,
      redirectUri,
      scopes
    );

    // Log audit event
    const { error: auditError } = await supabase.from('audit_logs').insert({
      user_id: user.id,
      action: 'oauth_initiated',
      resource_type: 'oauth_token',
      details: {
        provider,
        mcp_name,
        scopes: scopes || [],
      },
      ip_address: req.headers['x-forwarded-for'] as string,
      user_agent: req.headers['user-agent'],
    });

    if (auditError) {
      console.error('Failed to log audit event:', auditError);
    }

    // Return authorization URL
    return res.status(200).json({
      success: true,
      authorization_url: authUrl,
      state,
      provider,
      mcp_name,
    });
  } catch (err) {
    console.error('Error initiating OAuth flow:', err);

    return res.status(500).json({
      error: 'Internal server error',
      message: err instanceof Error ? err.message : 'Unknown error',
    });
  }
}

// Export handler configuration
export const config = {
  api: {
    bodyParser: true,
  },
};
