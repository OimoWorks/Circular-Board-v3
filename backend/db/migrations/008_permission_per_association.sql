-- ═══════════════════════════════════════════════════════════
-- 権限管理：自治会別設定対応
-- role_permissions に association_id を追加し、自治会ごとに
-- 独立した権限設定ができるようにする。
-- association_id が NULL の行はデフォルト設定として全自治会に適用される。
-- association_id が指定されている行はその自治会専用の設定となり、
-- 自治会専用設定が存在しない場合はデフォルト設定にフォールバックする。
-- ═══════════════════════════════════════════════════════════

-- ─── role_permissions に association_id カラムを追加 ────────
ALTER TABLE role_permissions
  ADD COLUMN association_id UUID NULL REFERENCES associations(id) ON DELETE CASCADE;

-- 既存の UNIQUE 制約（role_id, feature_id）を削除する
-- PostgreSQL が自動生成する制約名は "role_permissions_role_id_feature_id_key"
ALTER TABLE role_permissions
  DROP CONSTRAINT IF EXISTS role_permissions_role_id_feature_id_key;

-- デフォルト設定（association_id IS NULL）の部分一意インデックス
-- 同一ロール×機能のデフォルト設定は1件のみ許可
CREATE UNIQUE INDEX role_permissions_default_unique
  ON role_permissions(role_id, feature_id)
  WHERE association_id IS NULL;

-- 自治会別設定（association_id IS NOT NULL）の部分一意インデックス
-- 同一ロール×機能×自治会の設定は1件のみ許可
CREATE UNIQUE INDEX role_permissions_assoc_unique
  ON role_permissions(role_id, feature_id, association_id)
  WHERE association_id IS NOT NULL;

-- association_id による検索用インデックス
CREATE INDEX idx_role_permissions_association_id
  ON role_permissions(association_id);

-- ─── operation_logs に association_id カラムを追加 ──────────
-- 権限変更操作がどの自治会に対する設定変更かを記録する
ALTER TABLE operation_logs
  ADD COLUMN association_id UUID NULL REFERENCES associations(id) ON DELETE SET NULL;

CREATE INDEX idx_operation_logs_association_id
  ON operation_logs(association_id);
