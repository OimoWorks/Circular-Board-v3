# 回覧板アプリ（Circular Board）

マルチテナント対応の回覧板アプリです。複数の自治会を1つのシステムで管理できます。

## 技術スタック

| レイヤー | 技術 |
|---|---|
| バックエンド | Go 1.24 / chi v5.0.11 / pgx v5.5.1 / jwt v5.2.0 |
| フロントエンド | Flutter 3.24.0 / Riverpod |
| DB | PostgreSQL 16-alpine |
| プロキシ | nginx 1.25-alpine |
| 開発ツール | air v1.52.3（ホットリロード） |

---

## ディレクトリ構成

```
Circular-Board-v3/
├── backend/
│   ├── cmd/server/          # エントリポイント
│   ├── internal/
│   │   ├── config/          # 環境変数設定
│   │   ├── domain/          # ドメインモデル（User, Association）
│   │   ├── repository/      # DBアクセス層
│   │   ├── service/         # ビジネスロジック
│   │   ├── handler/         # HTTPハンドラ
│   │   ├── middleware/       # JWT認証・CORS
│   │   └── db/              # マイグレーション・シーダー
│   ├── db/
│   │   └── migrations/      # SQLマイグレーションファイル
│   ├── Dockerfile
│   ├── .air.toml
│   └── go.mod
├── frontend/
│   ├── lib/
│   │   ├── core/            # ルーター・APIクライアント・定数
│   │   └── features/auth/   # 認証機能（画面・Provider・Repository）
│   ├── web/                 # Flutter Web設定
│   ├── pubspec.yaml
│   └── Dockerfile
├── nginx/
│   └── nginx.conf
├── docker-compose.yml
├── .env.example
└── README.md
```

---

## 初回起動手順

### 1. 環境変数ファイルを作成

```bash
cp .env.example .env
# 必要に応じて .env の JWT_SECRET を変更してください
```

### 2. Docker で起動

```bash
docker compose up --build
```

初回起動時に自動で以下が実行されます：
- DBマイグレーション（テーブル作成）
- シードデータ投入（管理者・サンプル自治会・ユーザー）

### 3. 動作確認

| サービス | URL |
|---|---|
| バックエンドAPI | http://localhost:8080 |
| フロントエンド（Web） | http://localhost:3000 |
| nginx（統合） | http://localhost |

---

## 動作確認手順

### APIで確認（curl）

#### ログイン（system_admin）

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "association_code": "",
    "email": "admin@system.local",
    "password": "Admin1234!"
  }'
```

#### ログイン（自治会管理者）

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "association_code": "SAMPLE01",
    "email": "admin@sample01.local",
    "password": "Admin1234!"
  }'
```

#### ログイン（一般ユーザー）

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "association_code": "SAMPLE01",
    "email": "user@sample01.local",
    "password": "User1234!"
  }'
```

#### 認証ユーザー情報取得

```bash
# 上記ログインレスポンスの access_token を使用
TOKEN="eyJhbGci..."

curl http://localhost:8080/api/v1/auth/me \
  -H "Authorization: Bearer $TOKEN"
```

#### トークンリフレッシュ

```bash
REFRESH_TOKEN="..."

curl -X POST http://localhost:8080/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d "{\"refresh_token\": \"$REFRESH_TOKEN\"}"
```

#### ログアウト

```bash
curl -X POST http://localhost:8080/api/v1/auth/logout \
  -H "Authorization: Bearer $TOKEN"
```

### Webブラウザで確認

1. http://localhost:3000 にアクセス
2. ログイン画面が表示される
3. 以下のアカウントでログイン：

| 役割 | 自治会コード | メールアドレス | パスワード |
|---|---|---|---|
| システム管理者 | （空欄） | admin@system.local | Admin1234! |
| 自治会管理者 | SAMPLE01 | admin@sample01.local | Admin1234! |
| 一般ユーザー | SAMPLE01 | user@sample01.local | User1234! |

4. ホーム画面でユーザー名・メールアドレス・ロールが表示される
5. 右上のログアウトボタンでログアウト → ログイン画面へ遷移

---

## 初期シードデータ

### 自治会

| コード | 名前 |
|---|---|
| SAMPLE01 | サンプル自治会 |
| TEST001 | テスト自治会 |

### ユーザー

| 名前 | メール | パスワード | ロール | 自治会 |
|---|---|---|---|---|
| システム管理者 | admin@system.local | Admin1234! | system_admin | なし（全横断） |
| 自治会管理者 | admin@sample01.local | Admin1234! | association_admin | SAMPLE01 |
| 一般ユーザー | user@sample01.local | User1234! | user | SAMPLE01 |

---

## API仕様

### 認証API

| メソッド | パス | 認証要否 | 説明 |
|---|---|---|---|
| POST | /api/v1/auth/login | 不要 | ログイン |
| GET | /api/v1/auth/me | 必要 | ログインユーザー情報 |
| POST | /api/v1/auth/logout | 必要 | ログアウト |
| POST | /api/v1/auth/refresh | 不要 | トークン更新 |

### エラーレスポンス形式

```json
{
  "error": {
    "code": "INVALID_CREDENTIALS",
    "message": "自治会コード・メールアドレス・パスワードをご確認ください"
  }
}
```

### 成功レスポンス形式

```json
{
  "data": { ... }
}
```

---

## ロールと権限

| ロール | 説明 |
|---|---|
| `user` | 一般ユーザー。自分の自治会データのみ参照可能 |
| `association_admin` | 自治会管理者。自分の自治会内の管理機能を利用可能 |
| `system_admin` | システム管理者。全自治会横断の操作が可能 |

### テナント境界

- `user` / `association_admin` は JWT の `association_id` に紐づくデータのみアクセス可能
- `system_admin` は `association_id` を持たず、全自治会のデータへのアクセスを許可
- バックエンドのミドルウェアで `association_id` を検証し、テナント越えを防ぐ

---

## 開発コマンド

### バックエンドのみ起動

```bash
docker compose up db backend
```

### DBに直接接続

```bash
docker compose exec db psql -U postgres -d circular_board
```

### テスト用DB

```bash
DATABASE_TEST_URL=postgres://postgres:postgres@localhost:5433/circular_board_test
```

### ログ確認

```bash
docker compose logs -f backend
docker compose logs -f frontend
```

### 停止・クリーンアップ

```bash
# 停止
docker compose down

# ボリュームごと削除（DB初期化）
docker compose down -v
```

---

## 今後の拡張ポイント

### 2. ファイル管理

- `associations` + `users` テーブルはすでに存在する。ファイルテーブルに `association_id` を追加するだけでテナント分離が完成する
- バックエンド: `internal/handler/file_handler.go`、`internal/repository/file_repository.go` を追加
- フロントエンド: `lib/features/files/` を追加

### 3. お知らせ

- `notices` テーブル（`association_id` カラム付き）を追加
- バックエンド: `internal/handler/notice_handler.go` を追加

### 4. アカウント管理

- `middleware.RequireRole(domain.RoleAssociationAdmin, domain.RoleSystemAdmin)` を使ってロール制御
- バックエンド: `internal/handler/user_handler.go` を追加

### 5. 自治会管理

- `middleware.RequireRole(domain.RoleSystemAdmin)` で system_admin のみ許可
- バックエンド: `internal/handler/association_handler.go` を追加

### かんたんモード対応

`flutter_riverpod` の `StateProvider` でUIモードを管理：

```dart
final uiModeProvider = StateProvider<UiMode>((ref) => UiMode.normal);

enum UiMode { normal, easy }
```

テーマにフォントサイズ・ボタン高さのオフセットを持たせることで、
アプリ全体に一貫したかんたんモードを適用できます。
