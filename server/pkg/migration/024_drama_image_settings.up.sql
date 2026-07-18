ALTER TABLE drama_settings
    ADD COLUMN IF NOT EXISTS image_settings JSONB NOT NULL DEFAULT '{}'::jsonb;
