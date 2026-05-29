# 00. 共通設定・前提条件

このプロジェクトで使用するすべての技術バージョンと、
全プロンプト共通の前提条件をまとめています。

---

## プロジェクト概要

**マルチテナント対応 回覧板アプリ（Circular Board）**

- 自治会ごとにテナントが分離されたWebアプリ
- バックエンド：Go REST API
- フロントエンド：Flutter Web
- DB：PostgreSQL

---

## バージョン一覧

### バックエンド（Go）

```
module: circular-board
Go:     1.24

直接依存:
  github.com/go-chi/chi/v5          v5.0.11   # HTTPルーター
  github.com/golang-jwt/jwt/v5      v5.2.0    # JWT認証
  github.com/google/uuid            v1.6.0    # UUID生成
  github.com/jackc/pgx/v5           v5.5.1    # PostgreSQLドライバ
  github.com/stretchr/testify       v1.9.0    # テスト
  golang.org/x/crypto               v0.21.0   # bcrypt
```

### フロントエンド（Flutter）

```
name:    circular_board
version: 1.0.0+1
SDK:     Dart >=3.3.0 <4.0.0 / Flutter >=3.24.0

依存パッケージ:
  flutter_riverpod:      ^2.5.1   # 状態管理
  go_router:             ^13.2.0  # ルーティング
  dio:                   ^5.4.3   # HTTPクライアント
  flutter_secure_storage: ^9.2.2  # セキュアトークン保存
  file_picker:           ^8.0.0   # ファイル選択
  url_launcher:          ^6.2.0
  path_provider:         ^2.1.0
  pdfx:                  ^2.5.0   # PDFビューア
  flutter_pdfview:       ^1.4.4
  intl:                  ^0.19.0  # 日付フォーマット
  google_fonts:          ^6.1.0
```

---

## ディレクトリ構造

### バックエンド

```
backend/
├── cmd/server/main.go                  # エントリポイント・ルーティング定義
├── db/migrations/                      # SQLマイグレーション（001〜008）
│   ├── 001_init.sql                    # associations / users / refresh_tokens
│   ├── 002_files.sql                   # files
│   ├── 003_notices.sql                 # notices / notice_reads
│   ├── 004_master.sql                  # is_active カラム追加
│   ├── 005_surveys.sql                 # surveys / questions / choices / answers / password_reset_tokens
│   ├── 006_survey_images.sql           # survey_images
│   ├── 007_permissions.sql             # roles / features / role_permissions / operation_logs
│   └── 008_permission_per_association.sql # role_permissions に association_id 追加
├── internal/
│   ├── config/config.go                # 環境変数設定
│   ├── domain/user.go                  # ロール定義・User構造体
│   ├── domain/association.go
│   ├── db/migrator.go                  # マイグレーション実行
│   ├── db/seeder.go                    # 初期データ投入
│   ├── handler/auth_handler.go         # 認証API
│   ├── handler/response.go             # 共通レスポンス
│   ├── middleware/auth.go              # JWT認証ミドルウェア
│   ├── middleware/cors.go
│   ├── middleware/permission.go        # DB権限チェックミドルウェア
│   ├── repository/user_repository.go
│   ├── repository/refresh_token_repository.go
│   ├── repository/password_reset_repository.go
│   ├── service/auth_service.go
│   ├── files/{handler,model,repository,service}.go
│   ├── notice/{handler,model,repository,service}.go
│   ├── account/{handler,model,repository,service}.go
│   ├── association/{handler,model,repository,service}.go
│   ├── survey/{handler,image_handler,model,repository,service}.go
│   ├── permission/{handler,model,repository,service}.go
│   ├── home/{handler,model,repository,service}.go
│   └── operation_log/{model,repository}.go
```

### フロントエンド

```
frontend/lib/
├── main.dart                           # アプリエントリポイント
├── core/
│   ├── api_client.dart                 # Dio HTTPクライアント
│   ├── constants.dart                  # 定数（AppColors等）
│   ├── router.dart                     # GoRouterルーティング定義
│   └── download_helper*.dart           # ファイルダウンロード（Web/IO切替）
└── features/
    ├── auth/
    │   ├── data/auth_repository.dart
    │   ├── domain/user_model.dart
    │   └── presentation/{auth_provider,home_screen,login_screen,
    │       forgot_password_screen,reset_password_screen}.dart
    ├── files/
    │   ├── data/models/{file_model,association_model}.dart
    │   ├── data/repositories/{file_repository,association_repository}.dart
    │   ├── providers/{file_provider,association_provider}.dart
    │   ├── screens/{file_list_screen,file_preview_screen}.dart
    │   └── widgets/pdf_viewer*.dart
    ├── notice/
    │   ├── data/models/notice_model.dart
    │   ├── data/repositories/notice_repository.dart
    │   ├── providers/notice_provider.dart
    │   └── screens/{notice_list_screen,notice_create_screen,notice_detail_screen}.dart
    ├── account/
    │   ├── data/models/account_model.dart
    │   ├── data/repositories/account_repository.dart
    │   ├── providers/account_provider.dart
    │   └── screens/{account_list_screen,account_form_screen}.dart
    ├── association/
    │   ├── data/models/association_model.dart
    │   ├── data/repositories/association_repository.dart
    │   ├── providers/association_provider.dart
    │   └── screens/{association_list_screen,association_form_screen}.dart
    ├── survey/
    │   ├── data/models/survey_model.dart
    │   ├── data/repositories/survey_repository.dart
    │   ├── providers/survey_provider.dart
    │   └── screens/{survey_list_screen,survey_create_screen,
    │       survey_answer_screen,survey_result_screen}.dart
    └── permission/
        ├── data/models/permission_model.dart
        ├── data/repositories/permission_repository.dart
        ├── providers/permission_provider.dart
        └── screens/{permission_screen,emergency_appointment_screen}.dart
```

---

## APIベースパス

```
/api/v1/
```

---

## 共通レスポンス形式

### 成功

```json
{
  "data": { ... }
}
```

### エラー

```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "エラーメッセージ"
  }
}
```

---

## ロール定義

| ロール名 | 日本語名 | 説明 |
|---|---|---|
| `system_admin` | システム管理者 | 全テナント管理、自治会管理、権限管理 |
| `association_admin` | 自治会長 | 自治会内全権限（管理系除く） |
| `vice_admin` | 副会長 | お知らせ・回覧物・アンケートの作成・削除 |
| `user_admin` | ユーザー管理者 | アカウント追加・閲覧のみ |
| `user` | 一般ユーザー | 閲覧・回答のみ |

---

## JWT Claims 構造

```go
type Claims struct {
    UserID        string `json:"user_id"`
    AssociationID string `json:"association_id"`  // system_adminは空文字
    Role          string `json:"role"`
    jwt.RegisteredClaims
}
```

- 署名アルゴリズム：HS256
- アクセストークン有効期限：15分（`ACCESS_TOKEN_EXPIRY_MIN`）
- リフレッシュトークン有効期限：7日（`REFRESH_TOKEN_EXPIRY_DAYS`）

---

## 環境変数

| 変数名 | デフォルト値 | 説明 |
|---|---|---|
| `PORT` | `8080` | サーバーポート |
| `DATABASE_URL` | `postgres://postgres:postgres@localhost:5432/circular_board?sslmode=disable` | DB接続文字列 |
| `JWT_SECRET` | `change-me-in-production` | JWT署名秘密鍵 |
| `ACCESS_TOKEN_EXPIRY_MIN` | `15` | アクセストークン有効分数 |
| `REFRESH_TOKEN_EXPIRY_DAYS` | `7` | リフレッシュトークン有効日数 |
| `APP_ENV` | `development` | 環境名 |
| `UPLOAD_DIR` | `/app/uploads` | ファイルアップロードディレクトリ |
| `MAX_UPLOAD_BYTES` | `10485760`（10MB） | 最大アップロードサイズ |
| `SENDGRID_API_KEY` | 空 | SendGrid APIキー |
| `SENDGRID_FROM_EMAIL` | `noreply@example.com` | 送信元メールアドレス |
| `APP_BASE_URL` | `http://localhost` | アプリベースURL（リセットリンク生成用） |

---

## 認証方式

- `Authorization: Bearer <access_token>` ヘッダー
- リフレッシュトークンはレスポンスボディで返却（`refresh_token`フィールド）
- トークンはDBの `refresh_tokens` テーブルでhash管理（SHA-256）
- パスワードハッシュ：bcrypt DefaultCost

---

## フロントエンド共通

- Flutter Web ビルドターゲット
- 状態管理：Riverpod（ChangeNotifier + Provider）
- ルーター：GoRouter（認証状態をrefreshListenableで監視）
- HTTPクライアント：Dio（`/api/v1` をベースURLとして設定）
- トークン保存：flutter_secure_storage

---

## マルチテナント設計

- `association_id` が各テーブルのテナント識別子
- `system_admin` は全自治会にアクセス可能（JWTの `association_id` が空）
- 一般ユーザーはJWTの `association_id` で自動フィルタリング
- DB権限チェック：`role_permissions` テーブルで動的に制御
  - デフォルト設定（`association_id IS NULL`）
  - 自治会別オーバーライド（`association_id IS NOT NULL`）
