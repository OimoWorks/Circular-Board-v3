-- 自治会テーブル
CREATE TABLE IF NOT EXISTS associations (
    id         UUID        PRIMARY KEY,
    name       VARCHAR(100) NOT NULL,
    code       VARCHAR(50)  NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ユーザーテーブル
CREATE TABLE IF NOT EXISTS users (
    id            UUID        PRIMARY KEY,
    association_id UUID        REFERENCES associations(id),
    name          VARCHAR(100) NOT NULL,
    email         VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role          VARCHAR(50)  NOT NULL CHECK (role IN ('user', 'association_admin', 'system_admin')),
    is_active     BOOLEAN     NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- (association_id, email) の組合せでユニーク制約
-- system_adminはassociation_idがNULLのためemailのみでユニーク
CREATE UNIQUE INDEX IF NOT EXISTS users_email_assoc_idx
    ON users (COALESCE(association_id::text, ''), email);

-- リフレッシュトークンテーブル
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id         UUID        PRIMARY KEY,
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS refresh_tokens_user_id_idx ON refresh_tokens(user_id);
CREATE INDEX IF NOT EXISTS refresh_tokens_token_hash_idx ON refresh_tokens(token_hash);
