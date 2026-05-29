# Circular Board プロンプト集

マルチテナント対応 回覧板アプリ（Circular Board）の
開発プロンプト集です。実際に動作しているコードから
リバースエンジニアリングして作成しました。

---

## ファイル一覧と説明

| ファイル | 内容 |
|---|---|
| [00_common_settings.md](./00_common_settings.md) | バージョン・ディレクトリ構造・共通前提 |
| [01_auth.md](./01_auth.md) | 認証機能（ログイン・JWT・リフレッシュ） |
| [02_files.md](./02_files.md) | 回覧物管理（PDF/画像アップロード・ダウンロード） |
| [03_notices.md](./03_notices.md) | お知らせ機能（既読管理・ピン留め） |
| [04_surveys.md](./04_surveys.md) | アンケート機能（質問・選択肢・集計） |
| [05_accounts.md](./05_accounts.md) | アカウント管理（CRUD・ロール制御） |
| [06_associations.md](./06_associations.md) | 自治会管理（system_admin専用） |
| [07_permissions.md](./07_permissions.md) | 権限管理（DBドリブンRBAC・自治会別オーバーライド） |
| [08_password_reset.md](./08_password_reset.md) | パスワードリセット（SendGridメール） |
| [09_fixes.md](./09_fixes.md) | UI修正・バグ修正・スマホ対応のプロンプト集 |
| [10_tests.md](./10_tests.md) | 各機能のテストコードプロンプト |

---

## 推奨する実装順序

新規でゼロから構築する場合は以下の順序で実装します。

```
Step 1: 00_common_settings.md
  └── 技術スタック確認・DBセットアップ・プロジェクト初期化

Step 2: 01_auth.md
  └── 認証基盤（JWT・ミドルウェア）を先に構築

Step 3: 06_associations.md
  └── マルチテナントの基盤となる自治会テーブル

Step 4: 05_accounts.md
  └── ユーザー管理（各機能のテストデータ作成に必要）

Step 5: 07_permissions.md
  └── DB権限チェックミドルウェア（他機能のAPIに必要）

Step 6: 03_notices.md
  └── 基本的な CRUD（比較的シンプル）

Step 7: 02_files.md
  └── ファイルアップロード・ダウンロード

Step 8: 04_surveys.md
  └── 複雑なリレーション（質問・選択肢・回答）

Step 9: 08_password_reset.md
  └── パスワードリセット（メール送信含む）

Step 10: 09_fixes.md / 10_tests.md
  └── 修正・テスト追加
```

---

## 各プロンプトの使い方

### 基本的な使い方

```
1. 00_common_settings.md を最初に読む（前提条件の確認）
2. 実装したい機能のファイルを開く
3. 「プロンプト」セクションのコードブロック内をそのままAIに貼り付ける
4. 「実装済みの詳細仕様」セクションで補足情報を確認する
```

### プロンプトのカスタマイズ

各プロンプトは実際の実装から抽出しているため、
以下の点を変更して再利用できます：

- **テーブル名・カラム名**: 別プロジェクトに合わせて変更
- **エラーコード・エラーメッセージ**: 日本語 → 英語 等
- **ロール名**: 自治会管理の文脈に合わせた名称を変更
- **APIパス**: `/api/v1/` のプレフィックスを変更

---

## 技術スタック（動作確認済み）

```
【バックエンド】
Go:          1.24
Chi:         v5.0.11
pgx:         v5.5.1
JWT:         golang-jwt/jwt v5.2.0
UUID:        google/uuid v1.6.0
bcrypt:      golang.org/x/crypto v0.21.0
Test:        stretchr/testify v1.9.0
DB:          PostgreSQL（バージョン不問）
メール:      SendGrid v3 API

【フロントエンド】
Flutter:     >=3.24.0
Dart SDK:    >=3.3.0 <4.0.0
Riverpod:    ^2.5.1
GoRouter:    ^13.2.0
Dio:         ^5.4.3
SecureStore: flutter_secure_storage ^9.2.2
FilePicker:  ^8.0.0
PDF:         pdfx ^2.5.0 / flutter_pdfview ^1.4.4
intl:        ^0.19.0
```

---

## 設計上の重要な考え方

### マルチテナント分離

- 各テーブルに `association_id` を持たせ、APIレイヤーで自動フィルタ
- `system_admin` は `association_id` を持たず、全自治会にアクセス可
- `system_admin` は API リクエストに `?association_id=xxx` を付与して操作

### DB権限チェック（RBAC）

- `role_permissions` テーブルで各ロールの機能別権限を管理
- ハードコードした if 分岐ではなく、DBの設定を参照
- 自治会ごとにデフォルトをオーバーライドした設定が可能
- ミドルウェアで自動チェックするため各ハンドラーは権限を意識不要

### JWT設計

- `user_id`, `association_id`, `role` をペイロードに含める
- ハンドラーはコンテキストから Claims を取り出して使用
- リフレッシュトークンはローテーション（毎回新発行・旧削除）

### 論理削除

- `files`, `notices`, `surveys`, `users`, `associations` は物理削除しない
- `deleted_at` または `is_active` で管理
- 一覧取得時は `WHERE deleted_at IS NULL` または `WHERE is_active = true` でフィルタ

---

## よくある質問

**Q: system_admin はどのAPIにもアクセスできますか？**

A: `/api/v1/associations` と `/api/v1/permissions` は `system_admin` 専用。
その他は DB権限チェックを経由しますが、system_admin はチェックをスキップします。

**Q: 自治会コードはなんのために使いますか？**

A: ログイン画面で「自治会コード」を入力することで、
同じメールアドレスが別の自治会に登録されていても区別できます。

**Q: トークンの有効期限が切れたらどうなりますか？**

A: Flutter アプリが 401 レスポンスを受け取った際に
自動的にリフレッシュトークンで再取得を試みます。
リフレッシュも失敗した場合はログイン画面にリダイレクトします。

**Q: テスト用DBは本番DBと別にする必要がありますか？**

A: 別DBを推奨します。環境変数 `DATABASE_URL` でテスト用DBを指定してください。
デフォルト: `postgres://postgres:postgres@localhost:5432/circular_board_test`
