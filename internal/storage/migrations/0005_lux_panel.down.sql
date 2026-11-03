DROP INDEX IF EXISTS idx_lux_group_search;
DROP TABLE IF EXISTS lux_inventory;
DROP TABLE IF EXISTS lux_member_preferences;
ALTER TABLE economy_accounts DROP COLUMN IF EXISTS experience;
