-- ファイル管理テーブル
CREATE TABLE IF NOT EXISTS files (
    id                UUID        PRIMARY KEY,
    association_id    UUID        NOT NULL REFERENCES associations(id),
    year              SMALLINT    NOT NULL CHECK (year >= 2000 AND year <= 2099),
    month             SMALLINT    NOT NULL CHECK (month >= 1 AND month <= 12),
    filename          VARCHAR(255) NOT NULL,           -- 保存ファイル名（UUID形式）
    original_filename VARCHAR(255) NOT NULL,           -- アップロード元のファイル名
    storage_path      TEXT        NOT NULL,            -- ディスク上のフルパス
    uploaded_by       UUID        NOT NULL REFERENCES users(id),
    file_size         BIGINT      NOT NULL,
    mime_type         VARCHAR(100) NOT NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at        TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS files_association_year_month_idx
    ON files (association_id, year, month)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS files_uploaded_by_idx
    ON files (uploaded_by);
