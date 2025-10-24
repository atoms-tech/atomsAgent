# OAuth Init Endpoint - Quick Start Guide

Get the OAuth initialization endpoint up and running in 5 minutes.

## Prerequisites

- Node.js 18+
- Supabase account
- OAuth provider app created (GitHub, Google, Azure, or Auth0)

## 1. Install Dependencies

```bash
cd /Users/kooshapari/temp-PRODVERCEL/485/kush/agentapi
npm install
```

## 2. Set Up Database

```bash
# Apply the schema to Supabase
psql $SUPABASE_DB_URL < database/schema.sql
```

Or use Supabase dashboard:
1. Go to SQL Editor
2. Copy contents of `database/schema.sql`
3. Run the migration

## 3. Configure Environment

```bash
cp .env.example .env
```

Edit `.env` with your credentials:

```env
# Required
SUPABASE_URL=https://xxxxx.supabase.co
SUPABASE_SERVICE_ROLE_KEY=eyJhbGc...

# At least one provider required
GITHUB_CLIENT_ID=Iv1.xxxxx
GITHUB_CLIENT_SECRET=xxxxx
```

## 4. Test Locally

```bash
# Start local development server
vercel dev
```

## 5. Make a Test Request

```bash
# Replace with your actual JWT token
export USER_TOKEN="eyJhbGc..."

curl -X POST http://localhost:3000/api/mcp/oauth/init \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $USER_TOKEN" \
  -d '{
    "mcp_name": "test-mcp",
    "provider": "github",
    "redirect_uri": "http://localhost:3000/callback"
  }'
```

Expected response:
```json
{
  "success": true,
  "auth_url": "https://github.com/login/oauth/authorize?..."
}
```

## 6. Deploy to Vercel

```bash
# First time
vercel

# Production
vercel --prod
```

## Common Issues

### "Missing Supabase configuration"
- Check `SUPABASE_URL` and `SUPABASE_SERVICE_ROLE_KEY` are set
- Ensure no trailing slashes in URL

### "Provider is not properly configured"
- Verify `GITHUB_CLIENT_ID` and `GITHUB_CLIENT_SECRET` are set
- Check for typos in environment variable names

### "Invalid or expired token"
- Get a fresh JWT token from Supabase auth
- Ensure token hasn't expired

### "Failed to store OAuth state"
- Verify database migration ran successfully
- Check `oauth_state` table exists
- Ensure RLS policies are enabled

## Next Steps

1. **Implement Callback**: Create `/api/mcp/oauth/callback.ts`
2. **Frontend Integration**: Build OAuth popup flow
3. **Token Management**: Implement refresh logic
4. **Monitoring**: Add logging and metrics

## Files You Created

```
api/mcp/oauth/
├── init.ts              # Main endpoint
├── utils.ts             # Utilities
├── types.ts             # TypeScript types
├── init.test.ts         # Tests
├── README.md            # Full documentation
├── IMPLEMENTATION.md    # Implementation details
└── QUICKSTART.md        # This file
```

## Resources

- [Full Documentation](./README.md)
- [Setup Guide](../../../OAUTH_SETUP.md)
- [Implementation Details](./IMPLEMENTATION.md)
- [Vercel Edge Functions](https://vercel.com/docs/functions/edge-functions)
- [Supabase Auth](https://supabase.com/docs/guides/auth)

## Support

Need help? Check:
1. Test suite: `npx vitest`
2. Vercel logs: `vercel logs`
3. Database logs in Supabase dashboard
4. Environment variables: `vercel env ls`
