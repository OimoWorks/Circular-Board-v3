# 01. 認証機能

## 実装対象

- ログイン（自治会コード＋メール＋パスワード）
- ログアウト
- トークンリフレッシュ
- 自分の情報取得
- パスワードリセット（SendGrid メール送信）

---

## プロンプト

```
以下の仕様に従って、Go + Chi v5 + pgx v5 でJWT認証機能を実装してください。

【技術スタック】
- Go 1.24
- github.com/go-chi/chi/v5 v5.0.11
- github.com/golang-jwt/jwt/v5 v5.2.0
- github.com/google/uuid v1.6.0
- github.com/jackc/pgx/v5 v5.5.1
- golang.org/x/crypto v0.21.0（bcrypt）
- メール送信: SendGrid v3 API（net/httpで直接呼び出し）

【DBテーブル】

-- ユーザーテーブル
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

-- リフレッシュトークンテーブル
CREATE TABLE refresh_tokens (
    id         UUID        PRIMARY KEY,
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- パスワードリセットトークン
CREATE TABLE password_reset_tokens (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

【JWT Claims 構造】
type Claims struct {
    UserID        string `json:"user_id"`
    AssociationID string `json:"association_id"`  // system_adminは空文字
    Role          string `json:"role"`
    jwt.RegisteredClaims
}
- 署名アルゴリズム: HS256
- アクセストークン有効期限: 設定値（デフォルト15分）
- リフレッシュトークン有効期限: 設定値（デフォルト7日）
- リフレッシュトークンはローテーション（使用時に新規発行・旧トークン削除）

【APIエンドポイント（/api/v1/auth）】

POST /login
  Request:  { "association_code": string, "email": string, "password": string }
  Response: {
    "access_token": string,
    "refresh_token": string,
    "expires_in": int64,
    "token_type": "Bearer",
    "user": UserResponse
  }
  - association_code が空の場合は system_admin のみ検索
  - association_code があれば自治会コードで検索。見つからなければ system_admin にフォールバック
  - アカウントが is_active=false の場合も INVALID_CREDENTIALS を返す（ユーザー存在を隠蔽）

POST /refresh
  Request:  { "refresh_token": string }
  Response: 同上（トークンローテーション実施）

GET /me  [要認証]
  Response: { "data": UserResponse }

POST /logout  [要認証]
  処理: DBの全リフレッシュトークン削除

POST /forgot-password
  Request:  { "email": string }
  Response: { "message": "..." }（ユーザー存在有無に関わらず成功を返す）
  - ゴルーチンで非同期処理
  - トークン有効期限: 1時間
  - SHA-256でトークンをハッシュ化してDB保存
  - SendGridでHTMLメール送信（SENDGRID_API_KEYが空の場合はログのみ）

POST /reset-password
  Request:  { "token": string, "password": string }
  - パスワード最低8文字
  - トークンを SHA-256 でハッシュ化して検証
  - 更新後トークンに used_at を記録

【UserResponse 構造】
{
  "id": string,
  "name": string,
  "email": string,
  "role": string,
  "association_id": string | null,
  "association_name": string | null,
  "is_active": bool,
  "created_at": datetime
}

【エラーコード】
INVALID_REQUEST        400 リクエスト形式不正
INVALID_CREDENTIALS    401 ログイン失敗
INVALID_TOKEN          401 トークン不正・期限切れ
USER_NOT_FOUND         401/404
UNAUTHORIZED           401 認証なし
WEAK_PASSWORD          400 パスワード8文字未満
INTERNAL_ERROR         500

【フロントエンド（Flutter）】

lib/features/auth/ 配下の構成:
- data/auth_repository.dart  : DioでAPIを呼び出す
- domain/user_model.dart     : UserModelクラス（fromJson）
- presentation/auth_provider.dart   : AuthNotifier（ChangeNotifier）
- presentation/login_screen.dart    : ログイン画面
- presentation/home_screen.dart     : TOP画面
- presentation/forgot_password_screen.dart
- presentation/reset_password_screen.dart

AuthNotifier の状態:
- bool initialized / bool isAuthenticated / UserModel? currentUser
- initState() でflutter_secure_storageからトークンを復元して /me を呼ぶ
- トークンはflutter_secure_storageに保存（access_token / refresh_token）
- 401時はリフレッシュを試み、失敗したらログアウト

GoRouterのリダイレクト設定:
- 未認証 → /login
- 認証済みで /login アクセス → /
- publicPaths: ['/login', '/forgot-password', '/reset-password']
```

---

## ファイル配置

### バックエンド

```
backend/internal/
├── handler/auth_handler.go
├── service/auth_service.go
├── repository/user_repository.go
├── repository/refresh_token_repository.go
└── repository/password_reset_repository.go
```

### フロントエンド

```
frontend/lib/features/auth/
├── data/auth_repository.dart
├── domain/user_model.dart
└── presentation/
    ├── auth_provider.dart
    ├── login_screen.dart
    ├── home_screen.dart
    ├── forgot_password_screen.dart
    └── reset_password_screen.dart
```

---

## 実装済みの詳細仕様

### ログイン処理の優先順位

1. `association_code` が空 → `system_admin` のメールで検索
2. `association_code` あり → 自治会コードで検索 → 見つからなければ `system_admin` にフォールバック
3. bcrypt でパスワード検証
4. アクセストークン発行（HS256）
5. リフレッシュトークン生成（32バイトランダム → hex エンコード）、SHA-256 ハッシュでDB保存

### パスワードリセットメール（HTML）

- 件名: 「【回覧板】パスワードリセットのご案内」
- 送信者名: 「回覧板システム」
- リンク有効期限: 1時間
- ユーザーが存在しない場合はゴルーチン内で何もせず終了（ブルートフォース対策）
