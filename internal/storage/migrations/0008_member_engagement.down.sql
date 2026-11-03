UPDATE lux_shop_items
SET name = 'Dagas de Maze'
WHERE item_key = 'maze-daggers';

UPDATE lux_shop_items
SET name = 'Rosa de Maze'
WHERE item_key = 'maze-rose';

UPDATE lux_inventory
SET item_name = 'Dagas de Maze'
WHERE item_key = 'maze-daggers';

UPDATE lux_inventory
SET item_name = 'Rosa de Maze'
WHERE item_key = 'maze-rose';

DROP INDEX IF EXISTS idx_celestial_rewards_active;
DROP INDEX IF EXISTS idx_celestial_completed_reward_once;
DROP INDEX IF EXISTS idx_member_activity_words_top;
DROP INDEX IF EXISTS idx_member_voice_daily_leaderboard;
DROP INDEX IF EXISTS idx_member_activity_hourly_leaderboard;
DROP TABLE IF EXISTS celestial_redemptions;
DROP TABLE IF EXISTS celestial_rewards;
DROP TABLE IF EXISTS celestial_mission_claims;
DROP TABLE IF EXISTS celestial_wallets;
DROP TABLE IF EXISTS member_voice_daily;
DROP TABLE IF EXISTS member_voice_sessions;
DROP TABLE IF EXISTS member_activity_words;
DROP TABLE IF EXISTS member_activity_channels;
DROP TABLE IF EXISTS member_activity_hourly;
DROP TABLE IF EXISTS member_activity_preferences;
