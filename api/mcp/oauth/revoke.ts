/**
 * OAuth Token Revocation Endpoint
 *
 * Revokes OAuth tokens and removes them from storage.
 */

import type { VercelRequest, VercelResponse } from '@vercel/node';
import { createClient } from '@supabase/supabase-js';
import { revokeOAuthTokens } from './helpers';

const SUPABASE_URL = process.env.SUPABASE_URL!;
const SUPABASE_ANON_KEY = process.env.SUPABASE_ANON_KEY!;

interface RevokeRequest {
  mcp_name: string;
  provider: string;
}

export default async function handler(
  req: VercelRequest,
  res: VercelResponse
) {
  // Only accept POST or DELETE requests
  if (req.method !== 'POST' && req.method !== 'DELETE') {
    return res.status(405).json({
      error: 'Method not allowed',
      message: 'This endpoint only accepts POST or DELETE requests',
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
    const { mcp_name, provider }: RevokeRequest = req.body;

    // Validate required fields
    if (!mcp_name || !provider) {
      return res.status(400).json({
        error: 'Invalid request',
        message: 'Missing required fields: mcp_name and provider',
      });
    }

    // Revoke the OAuth tokens
    await revokeOAuthTokens(user.id, mcp_name, provider);

    // Log audit event
    const { error: auditError } = await supabase.from('audit_logs').insert({
      user_id: user.id,
      action: 'oauth_token_revoked',
      resource_type: 'oauth_token',
      details: {
        provider,
        mcp_name,
      },
      ip_address: req.headers['x-forwarded-for'] as string,
      user_agent: req.headers['user-agent'],
    });

    if (auditError) {
      console.error('Failed to log audit event:', auditError);
    }

    // Return success
    return res.status(200).json({
      success: true,
      message: 'OAuth tokens revoked successfully',
    });
  } catch (err) {
    console.error('Error revoking OAuth tokens:', err);

    return res.status(500).json({
      error: 'Token revocation failed',
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
