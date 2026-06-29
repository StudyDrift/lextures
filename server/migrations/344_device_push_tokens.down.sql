-- Rollback for 344_device_push_tokens.sql (tested in CI integration test)
DROP TABLE IF EXISTS settings.device_push_tokens;
