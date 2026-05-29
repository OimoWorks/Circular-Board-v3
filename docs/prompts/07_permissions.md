# 07. 権限管理機能

## 実装対象

- 権限マトリクス取得（自治会別・デフォルト）
- 権限一括更新（自治会別オーバーライド対応）
- ロール一覧取得
- 機能一覧取得
- 緊急任命（ロール変更）
- 操作ログ取得
- DB権限チェックミドルウェア

---

## プロンプト

```
以下の仕様に従って、Go + Chi v5 + pgx v5 で権限管理機能を実装してください。

【技術スタック】
- Go 1.24 / chi v5.0.11 / pgx v5.5.1 / uuid v1.6.0

【DBテーブル】

-- ロールテーブル（マスタ）
CREATE TABLE roles (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name         VARCHAR(50) NOT NULL UNIQUE,
    display_name VARCHAR(100) NOT NULL,
    is_system    BOOLEAN     NOT NULL DEFAULT false,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 機能テーブル（マスタ）
CREATE TABLE features (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name         VARCHAR(50) NOT NULL UNIQUE,
    display_name VARCHAR(100) NOT NULL,
    sort_order   INTEGER     NOT NULL DEFAULT 0
);

-- 権限マトリクステーブル
CREATE TABLE role_permissions (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    role_id        UUID        NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    feature_id     UUID        NOT NULL REFERENCES features(id) ON DELETE CASCADE,
    association_id UUID        NULL REFERENCES associations(id) ON DELETE CASCADE,
    can_view       BOOLEAN     NOT NULL DEFAULT false,
    can_create     BOOLEAN     NOT NULL DEFAULT false,
    can_edit       BOOLEAN     NOT NULL DEFAULT false,
    can_delete     BOOLEAN     NOT NULL DEFAULT false,
    scope          VARCHAR(20) NOT NULL DEFAULT 'own_association'
                   CHECK (scope IN ('all', 'own_association')),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- デフォルト設定（association_id IS NULL）の部分一意インデックス
CREATE UNIQUE INDEX role_permissions_default_unique
  ON role_permissions(role_id, feature_id)
  WHERE association_id IS NULL;

-- 自治会別設定（association_id IS NOT NULL）の部分一意インデックス
CREATE UNIQUE INDEX role_permissions_assoc_unique
  ON role_permissions(role_id, feature_id, association_id)
  WHERE association_id IS NOT NULL;

CREATE INDEX idx_role_permissions_role_id ON role_permissions(role_id);
CREATE INDEX idx_role_permissions_feature_id ON role_permissions(feature_id);
CREATE INDEX idx_role_permissions_association_id ON role_permissions(association_id);

-- 操作ログテーブル
CREATE TABLE operation_logs (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    operator_id    UUID        NOT NULL REFERENCES users(id),
    operation_type VARCHAR(50) NOT NULL,
    target_type    VARCHAR(50) NOT NULL,
    target_id      TEXT,
    before_value   JSONB,
    after_value    JSONB,
    association_id UUID        NULL REFERENCES associations(id) ON DELETE SET NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_operation_logs_operator_id ON operation_logs(operator_id);
CREATE INDEX idx_operation_logs_created_at  ON operation_logs(created_at DESC);
CREATE INDEX idx_operation_logs_association_id ON operation_logs(association_id);

【初期データ（マイグレーションで投入済み）】

ロール:
- system_admin（システム管理者, is_system=true）
- association_admin（自治会長）
- vice_admin（副会長）
- user_admin（ユーザー管理者）
- user（一般ユーザー）

機能:
- notices（お知らせ, sort_order=1）
- files（回覧物, sort_order=2）
- surveys（アンケート, sort_order=3）
- accounts（アカウント管理, sort_order=4）
- associations（自治会管理, sort_order=5）
- permissions（権限管理, sort_order=6）

初期権限マトリクス（association_id IS NULL のデフォルト設定）:
- system_admin: 全機能, view/create/edit/delete=true, scope=all
- association_admin: notices/files/surveys/accounts で全権限, scope=own_association
- vice_admin: notices/files/surveys で全権限, scope=own_association
- user_admin: notices/files/surveys は view のみ, accounts は view/create, scope=own_association
- user: notices/files/surveys は view のみ, scope=own_association

【APIエンドポイント（/api/v1/permissions）】
※すべて JWT 認証必須。system_admin のみアクセス可。

GET /?association_id=<uuid>
  - association_id 省略: デフォルト設定（association_id IS NULL の行）を返す
  - association_id 指定: その自治会の有効な権限（自治会専用→デフォルトフォールバック）
  - is_customized: 自治会専用設定が存在する場合 true
  Response: PermissionMatrixResponse

PUT /
  Request:
  {
    "association_id": "uuid" | null,
    "permissions": [
      {
        "role_id": "uuid",
        "feature_id": "uuid",
        "can_view": bool,
        "can_create": bool,
        "can_edit": bool,
        "can_delete": bool,
        "scope": "all" | "own_association"
      }
    ]
  }
  - association_id が null/省略: デフォルト設定を更新
  - association_id 指定: その自治会専用設定を UPSERT
  - 操作ログを operation_logs に記録
  Response: 200 { "message": "権限を更新しました" }

GET /roles
  Response: { "roles": [RoleModel, ...] }

GET /features
  Response: { "features": [FeatureModel, ...] }

POST /emergency-appointment
  Request: { "user_id": "uuid", "new_role": string }
  - 有効なロール名のみ受け付ける
  - user テーブルの role を更新
  - operation_logs に記録（operation_type="emergency_appointment"）
  Response: 200 { "message": "緊急任命を実行しました" }

GET /logs
  Response: { "logs": [OperationLog, ...] }

【PermissionMatrixResponse 構造】
{
  "roles": [
    { "id", "name", "display_name", "is_system", "created_at", "updated_at" }
  ],
  "features": [
    { "id", "name", "display_name", "sort_order" }
  ],
  "permissions": [
    {
      "id", "role_id", "feature_id", "association_id",
      "can_view", "can_create", "can_edit", "can_delete",
      "scope", "is_customized", "created_at", "updated_at"
    }
  ]
}

【UPSERT のコンフリクト指定】
-- デフォルト設定（association_id IS NULL）:
ON CONFLICT (role_id, feature_id) WHERE association_id IS NULL
DO UPDATE SET can_view=..., ...

-- 自治会専用設定（association_id IS NOT NULL）:
ON CONFLICT (role_id, feature_id, association_id) WHERE association_id IS NOT NULL
DO UPDATE SET can_view=..., ...

【DB権限チェックミドルウェア（middleware/permission.go）】

RequireFeature(featureName, action) の動作:
1. JWTの role が system_admin の場合は無条件で許可
2. JWTの association_id から role_permissions を検索
   - 自治会専用設定があればそれを使用（is_customized=true）
   - なければデフォルト設定（association_id IS NULL）にフォールバック
3. action = "view" → can_view チェック
   action = "create" → can_create チェック
   action = "edit" → can_edit チェック
   action = "delete" → can_delete チェック
4. false の場合 403 FORBIDDEN

フォールバッククエリ:
SELECT rp.* FROM role_permissions rp
JOIN roles ro ON ro.id = rp.role_id
JOIN features f ON f.id = rp.feature_id
WHERE ro.name = $1 AND f.name = $2
  AND (rp.association_id = $3 OR rp.association_id IS NULL)
ORDER BY (rp.association_id IS NULL)  -- false(0)=専用設定が先、true(1)=デフォルト後
LIMIT 1

【フロントエンド（Flutter）】

lib/features/permission/ 配下の構成:
- data/models/permission_model.dart
  - RoleModel, FeatureModel, RolePermission, PermissionMatrix
  - RolePermission: associationId（String?）, isCustomized（bool）
  - PermissionMatrix: permissionFor(roleId, featureId) → RolePermission?
- data/repositories/permission_repository.dart
  - getMatrix({String? associationId}) → PermissionMatrix
  - updatePermissions(List<RolePermission>, {String? associationId})
  - getRoles() / getFeatures()
  - emergencyAppointment({userId, newRole})
  - getLogs() → List<OperationLog>
- providers/permission_provider.dart
  - PermissionNotifier（ChangeNotifier）
  - EmergencyAppointmentNotifier
- screens/permission_screen.dart       : 権限マトリクス編集画面
- screens/emergency_appointment_screen.dart : 緊急任命画面

permission_screen.dart の要件:
- 自治会セレクター（ドロップダウン）
  - 「デフォルト設定（全自治会共通）」= null
  - 各自治会名 = 自治会ID
- 権限マトリクス（ロール × 機能）のテーブル表示
- 各セルに view/create/edit/delete チェックボックス + scope ドロップダウン
- system_admin ロールのセルは非表示（編集不可）
- 自治会専用設定があるセルはオレンジ背景 + tune アイコン
- 保存ボタン（成功/失敗メッセージ表示）
- 緊急任命ボタン（/permissions/emergency-appointment へ遷移）

PermissionNotifier の状態管理:
- selectedAssociationId: String?
- selectedAssociationName: String?（成功メッセージ表示用）
- matrix: PermissionMatrix?
- defaultMatrix: PermissionMatrix?（カスタマイズ判定用）
- editedPermissions: List<RolePermission>（保存前の編集状態）

save() の成功メッセージ:
- selectedAssociationId == null: 「デフォルト権限を更新しました」
- selectedAssociationName != null: 「{名前}の権限を更新しました」
- それ以外: 「権限を更新しました」
```

---

## ファイル配置

### バックエンド

```
backend/internal/permission/
├── handler.go    # GetMatrix/UpdatePermissions/GetRoles/GetFeatures/EmergencyAppointment/GetOperationLogs
├── model.go      # RoleModel/FeatureModel/RolePermission/PermissionMatrix/UpdateInput/OperationLog
├── repository.go # DB操作（DISTINCT ON でフォールバック）
└── service.go    # ビジネスロジック
```

### フロントエンド

```
frontend/lib/features/permission/
├── data/
│   ├── models/permission_model.dart
│   └── repositories/permission_repository.dart
├── providers/permission_provider.dart
└── screens/
    ├── permission_screen.dart
    └── emergency_appointment_screen.dart
```

---

## 実装済みの詳細仕様

### 自治会別フォールバッククエリ（DISTINCT ON）

```sql
SELECT DISTINCT ON (rp.role_id, rp.feature_id)
    rp.*,
    (rp.association_id IS NOT NULL) AS is_customized
FROM role_permissions rp
WHERE rp.association_id = $1 OR rp.association_id IS NULL
ORDER BY rp.role_id, rp.feature_id,
         (rp.association_id IS NULL)  -- false(0)=専用設定優先
```

### is_customized フラグ

- `association_id IS NOT NULL` の行を取得した場合 `is_customized = true`
- `association_id IS NULL`（デフォルト設定）にフォールバックした場合 `is_customized = false`

### 緊急任命の操作ログ

```go
OperationLog{
    OperationType: "emergency_appointment",
    TargetType:    "user",
    TargetID:      userID.String(),
    BeforeValue:   oldRole,
    AfterValue:    newRole,
    AssociationID: nil,
}
```

### 権限更新の操作ログ

```go
OperationLog{
    OperationType: "update_permissions",
    TargetType:    "role_permissions",
    AssociationID: assocID,  // 対象の自治会（デフォルト更新時はnil）
    AfterValue:    JSON形式の権限データ,
}
```
