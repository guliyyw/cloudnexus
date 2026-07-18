DO $$
BEGIN
    BEGIN
        ALTER TABLE oauth_bindings
            ADD COLUMN token_encryption_version SMALLINT NOT NULL DEFAULT 0;
    EXCEPTION WHEN duplicate_column THEN NULL;
    END;
END $$;
