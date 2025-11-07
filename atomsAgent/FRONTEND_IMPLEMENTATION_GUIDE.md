# Frontend Implementation Guide - AuthKit JWT for Atoms MCP

## Quick Summary

The backend (atomsAgent) and Atoms MCP server are **already implemented** and ready. Only the frontend needs to be updated to pass the AuthKit JWT token.

## What Needs to Be Done

### 1. Create Auth Helper (`src/lib/mcp/auth.ts`)

```typescript
import { getSession } from '@workos-inc/authkit-nextjs';

/**
 * Get authentication headers for MCP requests
 * 
 * For internal MCPs (atoms, atoms-dev), includes the user's AuthKit JWT
 * For external MCPs, returns empty headers (they handle their own auth)
 */
export async function getMCPAuthHeaders(mcpId: string): Promise<Record<string, string>> {
  // Check if this is an internal MCP
  const isInternal = mcpId === 'atoms' || mcpId === 'atoms-dev';
  
  if (!isInternal) {
    return {}; // External MCPs handle their own auth
  }
  
  // Get user session from AuthKit
  const session = await getSession();
  
  if (!session?.accessToken) {
    console.warn('No AuthKit session found for internal MCP request');
    return {};
  }
  
  // Return Authorization header with AuthKit JWT
  return {
    'Authorization': `Bearer ${session.accessToken}`
  };
}
```

### 2. Create MCP Registry (`src/lib/mcp/registry.ts`)

```typescript
/**
 * Registry of first-party (internal) MCP servers
 * These servers use AuthKit JWT for authentication
 */
export const FIRST_PARTY_MCPS = {
  "atoms": {
    name: "Atoms MCP",
    description: "Core Atoms platform tools",
    url: process.env.NEXT_PUBLIC_ATOMS_MCP_URL || "https://mcp.atoms.tech/api/mcp",
    scope: "system",
    auth: {
      type: "internal",
      requiresUserToken: true
    }
  },
  "atoms-dev": {
    name: "Atoms MCP (Dev)",
    description: "Development instance of Atoms MCP",
    url: process.env.NEXT_PUBLIC_ATOMS_MCP_DEV_URL || "http://localhost:8000/api/mcp",
    scope: "system",
    auth: {
      type: "internal",
      requiresUserToken: true
    }
  }
} as const;

export type FirstPartyMCPId = keyof typeof FIRST_PARTY_MCPS;

/**
 * Check if an MCP ID is a first-party (internal) server
 */
export function isFirstPartyMCP(mcpId: string): mcpId is FirstPartyMCPId {
  return mcpId in FIRST_PARTY_MCPS;
}
```

### 3. Update Chat Route (`src/app/api/chat/route.ts`)

Find the section where you make the request to atomsAgent and update it:

```typescript
import { getMCPAuthHeaders } from '@/lib/mcp/auth';

export async function POST(request: Request) {
  // ... existing code ...
  
  // Get MCP ID from request (adjust based on your implementation)
  const { mcpId, messages, ...otherParams } = await request.json();
  
  // Get auth headers for this MCP
  const authHeaders = await getMCPAuthHeaders(mcpId || 'atoms');
  
  // Make request to atomsAgent
  const response = await fetch(`${atomsAgentUrl}/v1/chat/completions`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...authHeaders,  // ← Add this line
    },
    body: JSON.stringify({
      messages,
      ...otherParams
    })
  });
  
  // ... rest of the code ...
}
```

## Environment Variables

Add to `.env.local` and Vercel:

```bash
# Atoms MCP URLs
NEXT_PUBLIC_ATOMS_MCP_URL=https://mcp.atoms.tech/api/mcp
NEXT_PUBLIC_ATOMS_MCP_DEV_URL=http://localhost:8000/api/mcp
```

## Testing Locally

1. **Start Atoms MCP locally** (optional):
   ```bash
   cd /Users/kooshapari/temp-PRODVERCEL/485/kush/atoms-mcp-prod
   source .venv/bin/activate
   python -m uvicorn server:app --port 8000
   ```

2. **Start atomsAgent**:
   ```bash
   cd /Users/kooshapari/temp-PRODVERCEL/485/kush/agentapi/atomsAgent
   source .venv/bin/activate
   atoms-agent server run --port 3284
   ```

3. **Start frontend**:
   ```bash
   cd atoms.tech
   npm run dev
   ```

4. **Test the flow**:
   - Login via AuthKit
   - Send a chat message
   - Check browser network tab:
     - Request to `/api/chat` should include session cookie
     - Request to atomsAgent should include `Authorization: Bearer <jwt>`
   - Check atomsAgent logs for: "Using user AuthKit JWT for internal MCP"
   - Verify no OAuth prompt appears

## Deployment

1. **Deploy Frontend**:
   ```bash
   cd atoms.tech
   vercel --prod
   ```

2. **Add Environment Variables** in Vercel dashboard:
   - `NEXT_PUBLIC_ATOMS_MCP_URL=https://mcp.atoms.tech/api/mcp`

3. **Verify**:
   - Login to production site
   - Send chat message
   - Check that no OAuth prompt appears
   - Verify in Vercel logs that requests include Authorization header

## Troubleshooting

### No Authorization header in request
- Check that `getMCPAuthHeaders()` is being called
- Verify AuthKit session exists: `const session = await getSession()`
- Check browser console for warnings

### OAuth prompt still appears
- Verify Atoms MCP has correct WorkOS environment variables
- Check Atoms MCP logs for JWT validation errors
- Ensure `FASTMCP_SERVER_AUTH_AUTHKITPROVIDER_AUTHKIT_DOMAIN` is correct

### 401 Unauthorized from Atoms MCP
- Check that AuthKit JWT is valid (not expired)
- Verify WorkOS client ID matches between frontend and Atoms MCP
- Check Atoms MCP logs for specific validation error

## Success Criteria

✅ User logs in via AuthKit
✅ Chat request includes Authorization header
✅ atomsAgent logs show "Using user AuthKit JWT for internal MCP"
✅ Atoms MCP validates JWT successfully
✅ No OAuth prompt appears
✅ User-specific data is returned from Atoms MCP

## Estimated Time

- Implementation: **1-2 hours**
- Testing: **30 minutes**
- Deployment: **15 minutes**
- **Total: ~2-3 hours**

