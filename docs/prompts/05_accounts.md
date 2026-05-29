# 05. アカウント管理機能

## 実装対象

- アカウント一覧取得
- アカウント作成
- アカウント更新
- アカウント無効化（論理削除）
- アカウント有効化/無効化切替

---

## プロンプト

```
以下の仕様に従って、Go + Chi v5 + pgx v5 でアカウント管理機能を実装してください。

【技術スタック】
- Go 1.24 / chi v5.0.11 / pgx v5.5.1 / uuid v1.6.0
- パスワードハッシュ: golang.org/x/crypto/bcrypt

【DBテーブル】

CREATE TABLE users (
    id            UUID        PRIMARY KEY,
    association_id UUID        REFERENCES associations(id),
    name          VARCHAR(100) NOT NULL,
    email         VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role          VARCHAR(50)  NOT NULL
        CHECK (role IN ('user', 'user_admin', 'vice_admin', 'association_admin', 'system_admin')),
    is_active     BOOLEAN     NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX users_email_assoc_idx
    ON users (COALESCE(association_id::text, ''), email);

【APIエンドポイント（/api/v1/accounts）】
※すべて JWT 認証必須。DB権限チェック。

GET /           [feature=accounts, action=view]
  クエリ: association_id（system_admin のみ、省略時は全件）
  - system_admin: association_id フィルタ可（省略=全件）
  - 一般: JWT の association_id に属するユーザーのみ
  Response: { "accounts": [AccountResponse, ...] }

POST /          [feature=accounts, action=create]
  Request:
  {
    "name": string,
    "email": string,
    "password": string,
    "role": string,
    "association_id": string | null  // system_admin 専用
  }
  バリデーション:
  - name, email, password は必須
  - password は8文字以上
  - role は有効なロール名のみ
  - メールアドレスの重複チェック（同一自治会内）
  - 呼び出し元が作成できるロールの制限:
    - user_admin: "user" のみ作成可
    - association_admin: "user", "user_admin", "vice_admin" 作成可
    - system_admin: すべてのロール作成可。association_id 指定必須
  Response: 201 Created + AccountResponse

PUT /{id}       [feature=accounts, action=edit]
  Request: { "name": string, "email": string, "password": string, "role": string }
  - password が空文字の場合は変更しない
  - ロール変更の権限は create と同様のルール
  Response: 200 + AccountResponse

DELETE /{id}    [feature=accounts, action=delete]
  ※ 論理削除（is_active=false）
  - 自分自身を削除しようとした場合はエラー
  Response: 200 { "message": "無効化しました" }

PUT /{id}/activate    [feature=accounts, action=edit]
  Response: 200 { "message": "有効化しました" }

PUT /{id}/deactivate  [feature=accounts, action=edit]
  - 自分自身の無効化は不可
  Response: 200 { "message": "無効化しました" }

【AccountResponse 構造】
{
  "id": "uuid",
  "association_id": "uuid"|"",
  "association_name": string|null,
  "name": string,
  "email": string,
  "role": "user"|"user_admin"|"vice_admin"|"association_admin"|"system_admin",
  "is_active": bool,
  "created_at": datetime,
  "updated_at": datetime
}

【エラーコード】
INVALID_REQUEST    400
INVALID_PARAM      400 association_id 形式不正
VALIDATION_ERROR   400 バリデーション失敗
ACCOUNT_NOT_FOUND  404
EMAIL_CONFLICT     409 メール重複
WEAK_PASSWORD      400 パスワード8文字未満
INVALID_ROLE       400 ロール名不正
FORBIDDEN          403 権限不足・自治会外操作
CANNOT_DELETE_SELF 400 自分自身の削除・無効化
INTERNAL_ERROR     500

【ロール作成権限マトリクス】
caller role       → 作成可能なロール
user_admin        → user のみ
association_admin → user, user_admin, vice_admin
system_admin      → user, user_admin, vice_admin, association_admin, system_admin

【フロントエンド（Flutter）】

lib/features/account/ 配下の構成:
- data/models/account_model.dart
  - AccountModel（fromJson, copyWith）
  - roleLabel ゲッター（日本語変換）
- data/repositories/account_repository.dart
  - list({String? associationId}) → List<AccountModel>
  - create({name, email, password, role, associationId}) → AccountModel
  - update(id, {name, email, password, role}) → AccountModel
  - activate(id) / deactivate(id)
- providers/account_provider.dart : AccountNotifier（ChangeNotifier）
- screens/account_list_screen.dart  : 一覧画面
- screens/account_form_screen.dart  : 作成・編集フォーム

account_list_screen.dart の要件:
- ロールバッジ（色分け）表示
- 有効/無効ステータス表示
- 作成ボタン（権限ありの場合のみ）
- 編集・有効化/無効化ボタン（権限ありの場合のみ）
- system_admin は自治会セレクター表示

account_form_screen.dart の要件:
- 名前・メール・パスワード（編集時は空で変更なし）入力
- ロールセレクター（呼び出し元の権限に応じて選択肢を制限）
- system_admin は自治会セレクター表示
- バリデーション: 必須チェック・パスワード8文字以上
- preselectedAssociationId パラメータ対応（自治会管理画面からの遷移用）
```

---

## ファイル配置

### バックエンド

```
backend/internal/account/
├── handler.go    # List/Create/Update/Delete/Activate/Deactivate
├── model.go      # Account構造体、CreateInput/UpdateInput
├── repository.go # DB操作
└── service.go    # ビジネスロジック（権限チェック・バリデーション）
```

### フロントエンド

```
frontend/lib/features/account/
├── data/
│   ├── models/account_model.dart
│   └── repositories/account_repository.dart
├── providers/account_provider.dart
└── screens/
    ├── account_list_screen.dart
    └── account_form_screen.dart
```

---

## 実装済みの詳細仕様

### メールアドレスのユニーク制約

```sql
-- 同一自治会内でのメール重複は不可
-- system_admin（association_id=NULL）はシステム全体でユニーク
CREATE UNIQUE INDEX users_email_assoc_idx
    ON users (COALESCE(association_id::text, ''), email);
```

### 自分自身の削除防止

```go
// handler.go の Delete
callerUserID, _ := uuid.Parse(claims.UserID)
if callerUserID == targetUserID {
    return ErrSelfDeactivation
}
```

### association_id の解決

```
- system_admin: リクエストボディの association_id を使用（必須）
- 一般ユーザー: JWT の association_id を使用
```
