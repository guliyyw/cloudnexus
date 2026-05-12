DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'users' AND column_name = 'privacy'
    ) THEN
        ALTER TABLE users ADD COLUMN privacy TEXT DEFAULT '{"allow_search":true,"allow_add_friend":true,"show_online":true}';
    END IF;
END $$;
