-- Companion to: 414_diplomas_certificates.sql

DROP TABLE IF EXISTS credentials.diploma_batch_items;
DROP TABLE IF EXISTS credentials.diploma_batches;
DROP TABLE IF EXISTS credentials.diplomas;
DROP TABLE IF EXISTS credentials.diploma_templates;

ALTER TABLE settings.platform_app_settings
    DROP COLUMN IF EXISTS ff_diplomas;
