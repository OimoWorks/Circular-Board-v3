# 02. 回覧物管理機能

## 実装対象

- ファイルアップロード（PDF / JPG / PNG、最大10MB）
- ファイル一覧（年月フィルタ）
- ファイルダウンロード
- 年一覧取得
- ファイル削除（論理削除）

---

## プロンプト

```
以下の仕様に従って、Go + Chi v5 + pgx v5 で回覧物管理機能を実装してください。

【技術スタック】
- Go 1.24 / chi v5.0.11 / pgx v5.5.1 / uuid v1.6.0

【DBテーブル】

CREATE TABLE files (
    id                UUID        PRIMARY KEY,
    association_id    UUID        NOT NULL REFERENCES associations(id),
    year              SMALLINT    NOT NULL CHECK (year >= 2000 AND year <= 2099),
    month             SMALLINT    NOT NULL CHECK (month >= 1 AND month <= 12),
    filename          VARCHAR(255) NOT NULL,           -- 保存ファイル名（UUID形式）
    original_filename VARCHAR(255) NOT NULL,           -- アップロード元のファイル名
    storage_path      TEXT        NOT NULL,            -- ディスク上のフルパス
    uploaded_by       UUID        NOT NULL REFERENCES users(id),
    file_size         BIGINT      NOT NULL,
    mime_type         VARCHAR(100) NOT NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at        TIMESTAMPTZ
);

CREATE INDEX files_association_year_month_idx
    ON files (association_id, year, month)
    WHERE deleted_at IS NULL;

CREATE INDEX files_uploaded_by_idx ON files (uploaded_by);

【APIエンドポイント（/api/v1/files）】
※すべて JWT 認証必須。DB権限チェック（role_permissions テーブル参照）。

POST /          [feature=files, action=create]
  - multipart/form-data
  - フィールド: file（必須）, year（省略時=現在年）, month（省略時=現在月）,
                association_id（system_admin のみ必須）
  - 対応MIME: application/pdf, image/jpeg, image/png のみ
  - 最大サイズ: 10MB（11MBでボディ制限）
  - 保存ファイル名: UUID + 拡張子（元のファイル名は original_filename に保存）
  - MIME判定: Content-Type ヘッダー。octet-stream の場合は先頭512バイトでSniff
  - Response: 201 Created + FileResponse

DELETE /{id}    [feature=files, action=delete]
  - system_admin: 自治会を問わず削除可（AdminDelete）
  - その他: JWT の association_id に属するファイルのみ削除
  - Response: 200 { "message": "削除しました" }

GET /           [feature=files, action=view]
  - クエリ: year（省略=全年）, month（省略=全月）,
            association_id（system_admin のみ）
  - system_admin は association_id 必須
  - Response: { "files": [...], "year": int, "month": int }

GET /{id}/download  [feature=files, action=view]
  - system_admin: 自治会不問
  - その他: JWT の association_id に属するファイルのみ
  - Response: ファイルバイナリ
    - Content-Type: ファイルのmime_type
    - Content-Disposition: attachment; filename*=UTF-8''<URLエンコードした元ファイル名>
    - Content-Length: ファイルサイズ
    - http.ServeFile で配信

GET /years      [feature=files, action=view]
  - クエリ: association_id（system_admin のみ）
  - Response: { "years": [2024, 2023, ...] }

【FileResponse 構造】
{
  "id": "uuid",
  "association_id": "uuid",
  "year": 2024,
  "month": 5,
  "original_filename": "回覧板5月号.pdf",
  "file_size": 102400,
  "mime_type": "application/pdf",
  "uploaded_by": "uuid",
  "created_at": "2024-05-01T00:00:00Z"
}

【エラーコード】
FILE_TOO_LARGE       413 ファイルサイズが10MB超
UNSUPPORTED_FILE_TYPE 400 対応外MIME
FILE_NOT_FOUND       404
MISSING_PARAM        400 association_id が必要なのに未指定
INVALID_ID           400 UUID形式不正
UNAUTHORIZED         401
FORBIDDEN            403 自治会不一致
INTERNAL_ERROR       500

【association_id 解決ルール】
- system_admin: フォーム/クエリの association_id パラメータから取得（必須）
- 一般ユーザー: JWT の association_id を使用

【許可MIME タイプ】
var AllowedMIMETypes = map[string]string{
    "application/pdf": ".pdf",
    "image/jpeg":      ".jpg",
    "image/png":       ".png",
}

【フロントエンド（Flutter）】

lib/features/files/ 配下の構成:
- data/models/file_model.dart       : FileModel クラス（fromJson, fileSizeLabel ゲッター）
- data/repositories/file_repository.dart : 一覧取得・アップロード・削除・ダウンロード
- providers/file_provider.dart      : FileNotifier（ChangeNotifier）
- screens/file_list_screen.dart     : 年月フィルタ付き一覧画面
- screens/file_preview_screen.dart  : PDF/画像プレビュー（GoRouterで FileModel を extra 渡し）
- widgets/pdf_viewer.dart           : プラットフォーム別PDF表示

file_repository.dart のメソッド:
- list({int? year, int? month, String? associationId}) → List<FileModel>
- upload(Uint8List bytes, String filename, String mimeType, int year, int month, {String? associationId}) → FileModel
- delete(String id) → void
- download(String id) → Uint8List（ブラウザはURL起動、モバイルはpathに保存）

FileListScreen の要件:
- 年セレクター + 月セレクター（ドロップダウン）
- ファイルカード表示（ファイル名・サイズ・アップロード日時）
- PDF はプレビューアイコン → FilePreviewScreen に遷移
- 画像はダウンロードボタン
- アップロードボタン（権限ありの場合のみ表示）
- 削除ボタン（権限ありの場合のみ表示）
- system_admin は自治会セレクター表示
```

---

## ファイル配置

### バックエンド

```
backend/internal/files/
├── handler.go    # HTTPハンドラ（Upload/Delete/List/Download/AvailableYears）
├── model.go      # File構造体、AllowedMIMETypes
├── repository.go # DB操作
└── service.go    # ビジネスロジック（ファイル保存パス生成等）
```

### フロントエンド

```
frontend/lib/features/files/
├── data/
│   ├── models/file_model.dart
│   └── repositories/file_repository.dart
├── providers/file_provider.dart
├── screens/
│   ├── file_list_screen.dart
│   └── file_preview_screen.dart
└── widgets/
    ├── pdf_viewer.dart
    ├── pdf_viewer_io.dart
    ├── pdf_viewer_web.dart
    └── pdf_viewer_stub.dart
```

---

## 実装済みの詳細仕様

### ファイル保存パス

```
{UPLOAD_DIR}/{association_id}/{year}/{month}/{uuid}{ext}
```

例: `/app/uploads/xxx-xxx/2024/5/yyy-yyy.pdf`

### MIMEタイプ判定

1. `Content-Type` ヘッダーが `application/pdf` / `image/jpeg` / `image/png` → そのまま使用
2. `application/octet-stream` または空の場合 → 先頭512バイトで `http.DetectContentType()` を実行
3. それでも対応外 → `UNSUPPORTED_FILE_TYPE` エラー

### 論理削除

- `DELETE /{id}` で `deleted_at = NOW()` にセット
- 一覧・ダウンロードは `WHERE deleted_at IS NULL` でフィルタ

### ダウンロードレスポンスヘッダー

```
Content-Disposition: attachment; filename*=UTF-8''<URL-encoded-filename>
Content-Type: <mime_type>
Content-Length: <file_size>
```

### フロントエンドのダウンロード処理

- Web: `url_launcher` でダウンロードURLを開く
- モバイル/デスクトップ: `path_provider` で一時ディレクトリに保存

### DB権限ミドルウェア

```
permMW.RequireFeature("files", "view")   → can_view チェック
permMW.RequireFeature("files", "create") → can_create チェック
permMW.RequireFeature("files", "delete") → can_delete チェック
```
