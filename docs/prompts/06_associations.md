# 06. 自治会管理機能

## 実装対象

- 自治会一覧取得
- 自治会作成
- 自治会更新
- 自治会有効化/無効化
- **system_admin 専用機能**

---

## プロンプト

```
以下の仕様に従って、Go + Chi v5 + pgx v5 で自治会管理機能を実装してください。

【技術スタック】
- Go 1.24 / chi v5.0.11 / pgx v5.5.1 / uuid v1.6.0

【DBテーブル】

CREATE TABLE associations (
    id         UUID        PRIMARY KEY,
    name       VARCHAR(100) NOT NULL,
    code       VARCHAR(50)  NOT NULL UNIQUE,
    is_active  BOOLEAN     NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

【APIエンドポイント（/api/v1/associations）】
※すべて JWT 認証必須。system_admin のみアクセス可（RequireRole ミドルウェア）。

GET /       system_admin 専用
  Response: { "associations": [AssociationResponse, ...] }

POST /      system_admin 専用
  Request:  { "name": string, "code": string }
  バリデーション:
  - name は必須
  - code は半角英大文字・数字・アンダースコアのみ（バリデーションはサービス層）
  - code は省略時に name から自動生成
  - code は全体でユニーク
  Response: 201 Created + AssociationResponse

PUT /{id}   system_admin 専用
  Request:  { "name": string, "code": string }
  Response: 200 + AssociationResponse

DELETE /{id}      system_admin 専用
  ※ 論理削除（is_active=false）
  Response: 200 { "message": "無効化しました" }

PUT /{id}/activate    system_admin 専用
  Response: 200 { "message": "有効化しました" }

PUT /{id}/deactivate  system_admin 専用
  Response: 200 { "message": "無効化しました" }

【AssociationResponse 構造】
{
  "id": "uuid",
  "name": string,
  "code": string,
  "is_active": bool,
  "created_at": datetime,
  "updated_at": datetime
}

【エラーコード】
INVALID_REQUEST        400
VALIDATION_ERROR       400 name が空
INVALID_CODE           400 コード形式不正（英大文字・数字・アンダースコア以外）
CODE_CONFLICT          409 コード重複
ASSOCIATION_NOT_FOUND  404
INTERNAL_ERROR         500

【コードバリデーション】
- 正規表現: ^[A-Z0-9_]+$
- 空の場合は name から自動生成（スペースを _ に、小文字を大文字に変換等）

【フロントエンド（Flutter）】

lib/features/association/ 配下の構成:
- data/models/association_model.dart
  - AssociationModel（一覧用シンプル）
  - AssociationDetail（フォーム用、全フィールド）
- data/repositories/association_repository.dart
  - list() → List<AssociationDetail>
  - create({name, code}) → AssociationDetail
  - update(id, {name, code}) → AssociationDetail
  - activate(id) / deactivate(id)
- providers/association_provider.dart : AssociationNotifier
- screens/association_list_screen.dart  : 一覧画面
- screens/association_form_screen.dart  : 作成・編集フォーム

association_list_screen.dart の要件:
- 有効/無効バッジ表示（is_active）
- 自治会コード表示
- 新規作成ボタン
- 編集・有効化/無効化ボタン
- アカウント管理へのリンク（その自治会でフィルタ）

association_form_screen.dart の要件:
- 自治会名入力（必須）
- 自治会コード入力（省略可、自動生成）
- バリデーション: 必須チェック・コード形式

【system_admin の認可】
backend/internal/middleware/auth.go の RequireRole ミドルウェアで制御:
r.Use(authMiddleware.RequireRole(domain.RoleSystemAdmin))
```

---

## ファイル配置

### バックエンド

```
backend/internal/association/
├── handler.go    # List/Create/Update/Delete/Activate/Deactivate
├── model.go      # Association構造体
├── repository.go # DB操作
└── service.go    # ビジネスロジック（コードバリデーション）
```

### フロントエンド

```
frontend/lib/features/association/
├── data/
│   ├── models/association_model.dart
│   └── repositories/association_repository.dart
├── providers/association_provider.dart
└── screens/
    ├── association_list_screen.dart
    └── association_form_screen.dart
```

---

## 実装済みの詳細仕様

### コードバリデーション

```go
// service.go
var validCodePattern = regexp.MustCompile(`^[A-Z0-9_]+$`)

if !validCodePattern.MatchString(code) {
    return ErrInvalidCode
}
```

### 自動コード生成

コードが空の場合、`name` から自動生成:
- スペース → `_`
- 小文字 → 大文字
- 英数字・アンダースコア以外は除去

### 論理削除

`DELETE /{id}` = `is_active = false` にセット。
物理削除は行わない。`is_active = true` で再有効化可能。

### ログイン時の自治会コード

- ログイン画面で「自治会コード」を入力
- `system_admin` は空で入力可能（または任意）
- コードで `associations` テーブルを検索し、`association_id` を特定

### 権限管理画面での自治会セレクター

- 権限管理画面（`/permissions`）で自治会を選択
- `/api/v1/associations` から一覧取得して dropdown 表示
- 自治会選択時は自治会別の権限設定を表示
