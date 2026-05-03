-- associations テーブルに is_active を追加
ALTER TABLE associations ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT true;

-- users.is_active は 001_init.sql 済み。念のため IF NOT EXISTS で保護
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT true;
