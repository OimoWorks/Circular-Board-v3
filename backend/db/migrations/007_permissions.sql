-- ═══════════════════════════════════════════════════════════
-- 権限管理 + 新ロール追加
-- ═══════════════════════════════════════════════════════════

-- ─── users テーブルのロール制約を拡張 ─────────────────────────
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check
    CHECK (role IN ('user', 'user_admin', 'vice_admin', 'association_admin', 'system_admin'));

-- ─── ロールテーブル ────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS roles (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name         VARCHAR(50) NOT NULL UNIQUE,
    display_name VARCHAR(100) NOT NULL,
    is_system    BOOLEAN     NOT NULL DEFAULT false,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ─── 機能テーブル ──────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS features (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name         VARCHAR(50) NOT NULL UNIQUE,
    display_name VARCHAR(100) NOT NULL,
    sort_order   INTEGER     NOT NULL DEFAULT 0
);

-- ─── ロール権限テーブル ────────────────────────────────────────
CREATE TABLE IF NOT EXISTS role_permissions (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    role_id    UUID        NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    feature_id UUID        NOT NULL REFERENCES features(id) ON DELETE CASCADE,
    can_view   BOOLEAN     NOT NULL DEFAULT false,
    can_create BOOLEAN     NOT NULL DEFAULT false,
    can_edit   BOOLEAN     NOT NULL DEFAULT false,
    can_delete BOOLEAN     NOT NULL DEFAULT false,
    scope      VARCHAR(20) NOT NULL DEFAULT 'own_association'
               CHECK (scope IN ('all', 'own_association')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (role_id, feature_id)
);

CREATE INDEX IF NOT EXISTS idx_role_permissions_role_id ON role_permissions(role_id);
CREATE INDEX IF NOT EXISTS idx_role_permissions_feature_id ON role_permissions(feature_id);

-- ─── 操作ログテーブル ──────────────────────────────────────────
CREATE TABLE IF NOT EXISTS operation_logs (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    operator_id    UUID        NOT NULL REFERENCES users(id),
    operation_type VARCHAR(50) NOT NULL,
    target_type    VARCHAR(50) NOT NULL,
    target_id      TEXT,
    before_value   JSONB,
    after_value    JSONB,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_operation_logs_operator_id ON operation_logs(operator_id);
CREATE INDEX IF NOT EXISTS idx_operation_logs_created_at  ON operation_logs(created_at DESC);

-- ─── 初期データ投入 ────────────────────────────────────────────

INSERT INTO roles (name, display_name, is_system) VALUES
    ('system_admin',      'システム管理者',   true),
    ('association_admin', '自治会長',         false),
    ('vice_admin',        '副会長',           false),
    ('user_admin',        'ユーザー管理者',   false),
    ('user',              '一般ユーザー',     false)
ON CONFLICT (name) DO NOTHING;

INSERT INTO features (name, display_name, sort_order) VALUES
    ('notices',      'お知らせ',     1),
    ('files',        '回覧物',       2),
    ('surveys',      'アンケート',   3),
    ('accounts',     'アカウント管理', 4),
    ('associations', '自治会管理',   5),
    ('permissions',  '権限管理',     6)
ON CONFLICT (name) DO NOTHING;

-- ─── 初期権限設定 ─────────────────────────────────────────────
-- CTEでロールID・機能IDを解決してから一括INSERT

WITH
  r AS (SELECT id, name FROM roles),
  f AS (SELECT id, name FROM features)
INSERT INTO role_permissions (role_id, feature_id, can_view, can_create, can_edit, can_delete, scope)
SELECT r.id, f.id, p.can_view, p.can_create, p.can_edit, p.can_delete, p.scope
FROM (VALUES
    -- system_admin: 全機能・全権限・スコープ=all
    ('system_admin', 'notices',      true,  true,  true,  true,  'all'),
    ('system_admin', 'files',        true,  true,  true,  true,  'all'),
    ('system_admin', 'surveys',      true,  true,  true,  true,  'all'),
    ('system_admin', 'accounts',     true,  true,  true,  true,  'all'),
    ('system_admin', 'associations', true,  true,  true,  true,  'all'),
    ('system_admin', 'permissions',  true,  true,  true,  true,  'all'),
    -- association_admin: 自治体内全権限（自治会管理・権限管理は不可）
    ('association_admin', 'notices',      true,  true,  true,  true,  'own_association'),
    ('association_admin', 'files',        true,  true,  true,  true,  'own_association'),
    ('association_admin', 'surveys',      true,  true,  true,  true,  'own_association'),
    ('association_admin', 'accounts',     true,  true,  true,  true,  'own_association'),
    ('association_admin', 'associations', false, false, false, false, 'own_association'),
    ('association_admin', 'permissions',  false, false, false, false, 'own_association'),
    -- vice_admin: お知らせ・回覧物・アンケートの編集権限
    ('vice_admin', 'notices',      true,  true,  true,  true,  'own_association'),
    ('vice_admin', 'files',        true,  true,  true,  true,  'own_association'),
    ('vice_admin', 'surveys',      true,  true,  true,  true,  'own_association'),
    ('vice_admin', 'accounts',     false, false, false, false, 'own_association'),
    ('vice_admin', 'associations', false, false, false, false, 'own_association'),
    ('vice_admin', 'permissions',  false, false, false, false, 'own_association'),
    -- user_admin: 一般ユーザーの追加のみ・閲覧
    ('user_admin', 'notices',      true,  false, false, false, 'own_association'),
    ('user_admin', 'files',        true,  false, false, false, 'own_association'),
    ('user_admin', 'surveys',      true,  false, false, false, 'own_association'),
    ('user_admin', 'accounts',     true,  true,  false, false, 'own_association'),
    ('user_admin', 'associations', false, false, false, false, 'own_association'),
    ('user_admin', 'permissions',  false, false, false, false, 'own_association'),
    -- user: 閲覧・回答のみ
    ('user', 'notices',      true,  false, false, false, 'own_association'),
    ('user', 'files',        true,  false, false, false, 'own_association'),
    ('user', 'surveys',      true,  false, false, false, 'own_association'),
    ('user', 'accounts',     false, false, false, false, 'own_association'),
    ('user', 'associations', false, false, false, false, 'own_association'),
    ('user', 'permissions',  false, false, false, false, 'own_association')
) AS p(role_name, feature_name, can_view, can_create, can_edit, can_delete, scope)
JOIN r ON r.name = p.role_name
JOIN f ON f.name = p.feature_name
ON CONFLICT (role_id, feature_id) DO NOTHING;
