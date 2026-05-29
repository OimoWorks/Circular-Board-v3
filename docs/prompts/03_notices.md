# 03. お知らせ機能

## 実装対象

- お知らせ作成
- お知らせ一覧（ピン留め優先ソート）
- お知らせ詳細取得
- 既読管理（閲覧時に自動既読）
- お知らせ削除（論理削除）
- 未読件数取得

---

## プロンプト

```
以下の仕様に従って、Go + Chi v5 + pgx v5 でお知らせ機能を実装してください。

【技術スタック】
- Go 1.24 / chi v5.0.11 / pgx v5.5.1 / uuid v1.6.0

【DBテーブル】

CREATE TABLE notices (
    id                UUID         PRIMARY KEY,
    association_id    UUID         NOT NULL REFERENCES associations(id),
    title             VARCHAR(255) NOT NULL,
    body              TEXT         NOT NULL,
    is_pinned         BOOLEAN      NOT NULL DEFAULT false,
    created_by        UUID         NOT NULL REFERENCES users(id),
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at        TIMESTAMPTZ
);

CREATE INDEX notices_association_idx
    ON notices (association_id, is_pinned DESC, created_at DESC)
    WHERE deleted_at IS NULL;

-- 既読管理テーブル
CREATE TABLE notice_reads (
    id         UUID        PRIMARY KEY,
    notice_id  UUID        NOT NULL REFERENCES notices(id),
    user_id    UUID        NOT NULL REFERENCES users(id),
    read_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (notice_id, user_id)
);

CREATE INDEX notice_reads_user_idx ON notice_reads (user_id);

【APIエンドポイント（/api/v1/notices）】
※すべて JWT 認証必須。DB権限チェック（role_permissions テーブル参照）。
※お知らせは自治会に紐づく。system_admin はこのエンドポイントにアクセス不可
  （JWT の association_id が空の場合 FORBIDDEN を返す）。

POST /          [feature=notices, action=create]
  Request:  { "title": string, "body": string, "is_pinned": bool }
  - title, body は必須（空文字不可）
  - association_id, created_by は JWT から取得
  Response: 201 Created + NoticeResponse

DELETE /{id}    [feature=notices, action=delete]
  - JWT の association_id に属するお知らせのみ削除
  - deleted_at をセット（論理削除）
  Response: 200 { "message": "削除しました" }

GET /           [feature=notices, action=view]
  - JWT の association_id の一覧
  - ソート: is_pinned DESC, created_at DESC
  - deleted_at IS NULL のみ
  - is_read フラグを各アイテムに付与（notice_reads テーブル参照）
  Response: { "notices": [NoticeResponse, ...] }

GET /unread-count  [feature=notices, action=view]
  ※ /{id} より先にルート登録すること（静的ルート優先）
  Response: { "unread_count": int }

GET /{id}       [feature=notices, action=view]
  - JWT の association_id のお知らせのみ取得
  - 取得時に自動で既読処理（エラーは無視）
  Response: 200 + NoticeResponse

POST /{id}/read [feature=notices, action=view]
  - 手動既読エンドポイント（GETと同じ自動既読があるが明示的にも呼べる）
  Response: 200 { "message": "既読にしました" }

【NoticeResponse 構造】
{
  "id": "uuid",
  "association_id": "uuid",
  "title": "string",
  "body": "string",
  "is_pinned": bool,
  "is_read": bool,
  "created_by": "uuid",
  "created_at": "datetime",
  "updated_at": "datetime"
}

【エラーコード】
INVALID_REQUEST   400 リクエスト形式不正
VALIDATION_ERROR  400 タイトル/本文が空
NOTICE_NOT_FOUND  404
FORBIDDEN         403 自治会不一致またはassociation_idなし
INTERNAL_ERROR    500

【JWT からの情報取得】
- association_id: JWTのclaims.AssociationID（空の場合はFORBIDDEN）
- user_id: JWTのclaims.UserID

【フロントエンド（Flutter）】

lib/features/notice/ 配下の構成:
- data/models/notice_model.dart     : NoticeModel（fromJson, copyWith(isRead)）
- data/repositories/notice_repository.dart : CRUD操作
- providers/notice_provider.dart    : NoticeNotifier（ChangeNotifier）
- screens/notice_list_screen.dart   : 一覧画面
- screens/notice_create_screen.dart : 作成画面
- screens/notice_detail_screen.dart : 詳細画面

NoticeNotifier の状態:
- List<NoticeModel> notices / bool isLoading / bool isSubmitting
- String? errorMessage / int unreadCount

メソッド:
- load() : 一覧取得 + 未読件数取得
- create(title, body, isPinned) → bool
- delete(id) → bool
- markAsRead(id)

notice_list_screen.dart の要件:
- ピン留めアイコン表示
- 未読バッジ表示
- 作成ボタン（権限ありの場合のみ表示、role判定）
- 削除ボタン（権限ありの場合のみ）
- タップで notice_detail_screen へ遷移

notice_create_screen.dart の要件:
- タイトル入力（必須）
- 本文入力（必須、multiline）
- ピン留めスイッチ
- 送信ボタン（ローディング状態対応）

notice_detail_screen.dart の要件:
- 本文のフルテキスト表示
- 詳細画面を開いた時点で自動既読
- 削除ボタン（権限ありの場合のみ）
```

---

## ファイル配置

### バックエンド

```
backend/internal/notice/
├── handler.go    # Create/Delete/List/UnreadCount/Get/MarkAsRead
├── model.go      # Notice構造体
├── repository.go # DB操作（ソート・is_read JOIN）
└── service.go    # ビジネスロジック
```

### フロントエンド

```
frontend/lib/features/notice/
├── data/
│   ├── models/notice_model.dart
│   └── repositories/notice_repository.dart
├── providers/notice_provider.dart
└── screens/
    ├── notice_list_screen.dart
    ├── notice_create_screen.dart
    └── notice_detail_screen.dart
```

---

## 実装済みの詳細仕様

### 一覧ソート

```sql
SELECT n.*, 
    (nr.id IS NOT NULL) AS is_read
FROM notices n
LEFT JOIN notice_reads nr ON nr.notice_id = n.id AND nr.user_id = $2
WHERE n.association_id = $1 AND n.deleted_at IS NULL
ORDER BY n.is_pinned DESC, n.created_at DESC
```

### 既読処理

- `GET /{id}` で詳細取得時、`MarkAsRead` を自動呼び出し（エラーは無視して続行）
- `notice_reads` テーブルに `(notice_id, user_id)` で UPSERT（ON CONFLICT DO NOTHING）

### 未読件数

```sql
SELECT COUNT(*)
FROM notices n
WHERE n.association_id = $1
  AND n.deleted_at IS NULL
  AND NOT EXISTS (
    SELECT 1 FROM notice_reads nr
    WHERE nr.notice_id = n.id AND nr.user_id = $2
  )
```

### ホーム画面への統合

- `recentNoticesProvider` で最新3件取得（ホーム画面表示用）
- system_admin のホーム画面にはお知らせセクション非表示

### DB権限ミドルウェア

```
permMW.RequireFeature("notices", "view")   → can_view チェック
permMW.RequireFeature("notices", "create") → can_create チェック
permMW.RequireFeature("notices", "delete") → can_delete チェック
```
