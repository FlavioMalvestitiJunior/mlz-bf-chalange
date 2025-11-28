-- Add is_blacklisted column to users table
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_blacklisted BOOLEAN DEFAULT false;

-- Create index for blacklisted users
CREATE INDEX IF NOT EXISTS idx_users_blacklisted ON users(is_blacklisted);
