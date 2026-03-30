-- Phase 2: Package system enhancement - extend groups table and add package_channels table
-- Adds frequency limiting, anti-ban strategy, session isolation, and traffic jitter controls

SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '10min';

BEGIN;

-- 1) Add new columns to groups table
ALTER TABLE groups ADD COLUMN IF NOT EXISTS frequency_period INTEGER NOT NULL DEFAULT 1;
ALTER TABLE groups ADD COLUMN IF NOT EXISTS max_concurrent INTEGER NOT NULL DEFAULT 3;
ALTER TABLE groups ADD COLUMN IF NOT EXISTS enable_anti_ban BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE groups ADD COLUMN IF NOT EXISTS session_isolation BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE groups ADD COLUMN IF NOT EXISTS traffic_jitter BOOLEAN NOT NULL DEFAULT false;

COMMENT ON COLUMN groups.frequency_period IS 'Frequency limit period in hours, e.g. 3 means "every 3 hours" limit';
COMMENT ON COLUMN groups.max_concurrent IS 'Maximum concurrent requests';
COMMENT ON COLUMN groups.enable_anti_ban IS 'Enable anti-ban strategy';
COMMENT ON COLUMN groups.session_isolation IS 'Session isolation switch';
COMMENT ON COLUMN groups.traffic_jitter IS 'Traffic jitter/scrambling switch';

-- 2) Create package_channels table for mapping groups to accounts with weighting
CREATE TABLE IF NOT EXISTS package_channels (
    id          BIGSERIAL PRIMARY KEY,
    group_id    BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    account_id  BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    weight      INTEGER NOT NULL DEFAULT 1,
    max_users   INTEGER NOT NULL DEFAULT 0,
    is_enabled  BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_group_account UNIQUE (group_id, account_id)
);

CREATE INDEX IF NOT EXISTS idx_package_channels_group ON package_channels(group_id);
CREATE INDEX IF NOT EXISTS idx_package_channels_account ON package_channels(account_id);
CREATE INDEX IF NOT EXISTS idx_package_channels_enabled ON package_channels(is_enabled) WHERE is_enabled = true;

COMMENT ON TABLE package_channels IS 'Maps groups to accounts with weights for load balancing';
COMMENT ON COLUMN package_channels.weight IS 'Load balancing weight (higher = more traffic)';
COMMENT ON COLUMN package_channels.max_users IS 'Maximum concurrent users (0 = unlimited)';

COMMIT;