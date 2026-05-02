-- お知らせテーブル
CREATE TABLE IF NOT EXISTS notices (
    id                UUID         PRIMARY KEY,
    association_id    UUID         NOT NULL REFERENCES associations(id),
    title             VARCHAR(255) NOT NULL,
    body              TEXT         NOT NULL,
    is_pinned         BOOLEAN      NOT NULL DEFAULT false,
    created_by        UUID         NOT NULL REFERENCES users(id),
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at        TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS notices_association_idx
    ON notices (association_id, is_pinned DESC, created_at DESC)
    WHERE deleted_at IS NULL;

-- 既読管理テーブル
CREATE TABLE IF NOT EXISTS notice_reads (
    id         UUID        PRIMARY KEY,
    notice_id  UUID        NOT NULL REFERENCES notices(id),
    user_id    UUID        NOT NULL REFERENCES users(id),
    read_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (notice_id, user_id)
);

CREATE INDEX IF NOT EXISTS notice_reads_user_idx
    ON notice_reads (user_id);
