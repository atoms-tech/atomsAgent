-- Platform admin management tables
-- This migration creates tables for managing platform-wide administrators

-- Table for platform-wide admins
CREATE TABLE platform_admins (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workos_user_id TEXT NOT NULL UNIQUE,
    email TEXT NOT NULL UNIQUE,
    name TEXT,
    added_at TIMESTAMPTZ DEFAULT NOW(),
    added_by UUID REFERENCES platform_admins(id),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Table for audit logging of admin actions
CREATE TABLE admin_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    admin_id UUID NOT NULL REFERENCES platform_admins(id),
    action TEXT NOT NULL, -- 'added_admin', 'removed_admin', 'accessed_stats', etc.
    target_org_id TEXT,
    target_user_id TEXT,
    details JSONB,
    ip_address TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create indexes for better performance
CREATE INDEX idx_platform_admins_workos_user_id ON platform_admins(workos_user_id);
CREATE INDEX idx_platform_admins_email ON platform_admins(email);
CREATE INDEX idx_platform_admins_is_active ON platform_admins(is_active);
CREATE INDEX idx_admin_audit_log_admin_id ON admin_audit_log(admin_id);
CREATE INDEX idx_admin_audit_log_action ON admin_audit_log(action);
CREATE INDEX idx_admin_audit_log_created_at ON admin_audit_log(created_at);

-- Add RLS policies for security (if using Supabase/PostgreSQL with RLS)
-- Note: These policies assume you're using Supabase auth.uid() function
-- Adjust based on your authentication system

-- Enable RLS
ALTER TABLE platform_admins ENABLE ROW LEVEL SECURITY;
ALTER TABLE admin_audit_log ENABLE ROW LEVEL SECURITY;

-- Platform admins can read all platform admin records
CREATE POLICY platform_admins_select ON platform_admins
    FOR SELECT
    USING (auth.uid()::text IN (SELECT workos_user_id FROM platform_admins WHERE is_active = true));

-- Platform admins can insert new platform admin records
CREATE POLICY platform_admins_insert ON platform_admins
    FOR INSERT
    WITH CHECK (auth.uid()::text IN (SELECT workos_user_id FROM platform_admins WHERE is_active = true));

-- Platform admins can update platform admin records
CREATE POLICY platform_admins_update ON platform_admins
    FOR UPDATE
    USING (auth.uid()::text IN (SELECT workos_user_id FROM platform_admins WHERE is_active = true));

-- Platform admins can read all audit log records
CREATE POLICY admin_audit_log_select ON admin_audit_log
    FOR SELECT
    USING (auth.uid()::text IN (SELECT workos_user_id FROM platform_admins WHERE is_active = true));

-- Platform admins can insert audit log records
CREATE POLICY admin_audit_log_insert ON admin_audit_log
    FOR INSERT
    WITH CHECK (auth.uid()::text IN (SELECT workos_user_id FROM platform_admins WHERE is_active = true));

-- Insert a default platform admin (you'll need to replace with actual values)
-- This is optional - you can add the first admin manually via API
-- INSERT INTO platform_admins (workos_user_id, email, name, added_by) 
-- VALUES ('your-workos-user-id', 'admin@yourcompany.com', 'Platform Admin', NULL);