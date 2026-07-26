CREATE TABLE IF NOT EXISTS music_likes (
    user_id    BIGINT      NOT NULL,
    track_id   BIGINT      NOT NULL,
    source     VARCHAR(10) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, track_id, source)
);
CREATE INDEX IF NOT EXISTS idx_music_likes_user_created ON music_likes(user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS music_recent_plays (
    user_id   BIGINT      NOT NULL,
    track_id  BIGINT      NOT NULL,
    source    VARCHAR(10) NOT NULL,
    played_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, track_id, source)
);
CREATE INDEX IF NOT EXISTS idx_music_recent_user_played ON music_recent_plays(user_id, played_at DESC);
