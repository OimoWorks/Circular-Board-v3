# 04. アンケート機能

## 実装対象

- アンケート作成（質問・選択肢・画像添付）
- アンケート一覧
- アンケート詳細取得
- 回答送信（単一選択・複数選択）
- 集計結果表示
- アンケート削除
- 未回答件数取得
- アンケート画像アップロード/削除/取得

---

## プロンプト

```
以下の仕様に従って、Go + Chi v5 + pgx v5 でアンケート機能を実装してください。

【技術スタック】
- Go 1.24 / chi v5.0.11 / pgx v5.5.1 / uuid v1.6.0

【DBテーブル】

-- アンケート本体
CREATE TABLE surveys (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    association_id UUID NOT NULL REFERENCES associations(id),
    title          TEXT NOT NULL,
    description    TEXT,
    expires_at     TIMESTAMPTZ NOT NULL,
    created_by     UUID NOT NULL REFERENCES users(id),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at     TIMESTAMPTZ
);

CREATE INDEX idx_surveys_association_id ON surveys(association_id);
CREATE INDEX idx_surveys_expires_at ON surveys(expires_at);
CREATE INDEX idx_surveys_deleted_at ON surveys(deleted_at) WHERE deleted_at IS NULL;

-- 質問
CREATE TABLE survey_questions (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    survey_id     UUID NOT NULL REFERENCES surveys(id) ON DELETE CASCADE,
    question_text TEXT NOT NULL,
    question_type VARCHAR(20) NOT NULL CHECK (question_type IN ('single', 'multiple')),
    sort_order    INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX idx_survey_questions_survey_id ON survey_questions(survey_id);

-- 選択肢
CREATE TABLE survey_choices (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    question_id UUID NOT NULL REFERENCES survey_questions(id) ON DELETE CASCADE,
    choice_text TEXT NOT NULL,
    sort_order  INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX idx_survey_choices_question_id ON survey_choices(question_id);

-- 回答
CREATE TABLE survey_answers (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    survey_id   UUID NOT NULL REFERENCES surveys(id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    question_id UUID NOT NULL REFERENCES survey_questions(id) ON DELETE CASCADE,
    choice_id   UUID NOT NULL REFERENCES survey_choices(id) ON DELETE CASCADE,
    answered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, question_id, choice_id)
);

CREATE INDEX idx_survey_answers_survey_user ON survey_answers(survey_id, user_id);
CREATE INDEX idx_survey_answers_question ON survey_answers(question_id);

-- 添付画像
CREATE TABLE survey_images (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    survey_id      UUID        NOT NULL REFERENCES surveys(id) ON DELETE CASCADE,
    association_id UUID        NOT NULL REFERENCES associations(id),
    filename       TEXT        NOT NULL,
    storage_path   TEXT        NOT NULL,
    sort_order     INT         NOT NULL DEFAULT 0,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_survey_images_survey_id ON survey_images(survey_id);

【APIエンドポイント（/api/v1/surveys）】
※すべて JWT 認証必須。DB権限チェック。
※ system_admin は ?association_id=<uuid> クエリ必須

GET /unanswered-count   [feature=surveys, action=view]
  ※ /{id} より前にルート登録すること（静的ルート優先）
  Response: { "unanswered_count": int }

GET /                   [feature=surveys, action=view]
  Response: { "surveys": [SurveyListResponse, ...] }

GET /{id}               [feature=surveys, action=view]
  Response: SurveyResponse（質問・選択肢・画像を含む）

POST /{id}/answer       [feature=surveys, action=view]
  Request:
  {
    "answers": [
      {
        "question_id": "uuid",
        "choice_ids": ["uuid", ...]
      }
    ]
  }
  - 期限切れ（expires_at < NOW()）の場合 SURVEY_EXPIRED を返す
  - 選択肢のバリデーション（survey に属する question_id / choice_id か確認）
  Response: 200 { "message": "回答しました" }

POST /                  [feature=surveys, action=create]
  Request:
  {
    "title": string,
    "description": string,
    "expires_at": "2024-12-31T23:59:00Z",  // RFC3339
    "questions": [
      {
        "question_text": string,
        "question_type": "single"|"multiple",
        "sort_order": int,
        "choices": [
          { "choice_text": string, "sort_order": int }
        ]
      }
    ]
  }
  バリデーション（サービス層で実施）:
  - title は必須
  - questions は1件以上
  - expires_at は未来の日時
  - 各 question の question_text は必須
  - 各 question の type は "single" または "multiple"
  - 各 question の choices は2件以上
  Response: 201 Created + SurveyResponse

DELETE /{id}            [feature=surveys, action=delete]
  - system_admin: 全自治会削除可
  - 一般: JWT の association_id に属するもののみ
  - 論理削除（deleted_at セット）
  Response: 200 { "message": "削除しました" }

GET /{id}/results       [feature=surveys, action=view]
  Response: SurveyResultResponse

POST /{id}/images       [feature=surveys, action=create]
  - multipart/form-data: image（必須）, sort_order
  - MIME: image/jpeg, image/png, image/gif
  Response: 200 + ImageResponse

DELETE /{id}/images/{image_id}  [feature=surveys, action=delete]
  Response: 200 { "message": "削除しました" }

GET /{id}/images/{image_id}    [feature=surveys, action=view]
  Response: 画像バイナリ（Content-Type: 画像のMIMEタイプ）

【SurveyListResponse 構造】
{
  "id", "association_id", "title", "description",
  "expires_at", "is_answered", "is_expired",
  "created_by", "created_at"
}

【SurveyResponse 構造（SurveyListResponse + 以下）】
{
  ...,
  "questions": [
    {
      "id", "question_text", "question_type", "sort_order",
      "choices": [{ "id", "choice_text", "sort_order" }]
    }
  ],
  "images": [
    { "id", "survey_id", "association_id", "filename", "sort_order", "created_at" }
  ]
}

【SurveyResultResponse 構造】
{
  "survey_id": "uuid",
  "title": "string",
  "total_answered": int,
  "questions": [
    {
      "question_id", "question_text", "question_type",
      "total_answers": int,
      "choices": [
        { "choice_id", "choice_text", "count": int, "percentage": float64 }
      ]
    }
  ]
}

【エラーコード】
SURVEY_NOT_FOUND  404
SURVEY_EXPIRED    400 期限切れ
INVALID_REQUEST   400
INVALID_PARAM     400
VALIDATION_ERROR  400 バリデーション失敗
INTERNAL_ERROR    500

【フロントエンド（Flutter）】

lib/features/survey/ 配下の構成:
- data/models/survey_model.dart
  - SurveyModel, SurveyQuestion, SurveyChoice, SurveyImage,
    SurveyResult, QuestionResult, ChoiceResult
  - statusLabel ゲッター: "期限切れ" / "回答済み" / "未回答"
- data/repositories/survey_repository.dart
  - list / get / create / delete / answer / getResults
  - uploadImage(surveyId, bytes, filename, mimeType, sortOrder) → SurveyImage
  - getImageBytes(surveyId, imageId) → Uint8List
- providers/survey_provider.dart : SurveyNotifier
- screens/survey_list_screen.dart    : 一覧・ステータスバッジ
- screens/survey_create_screen.dart  : フォーム（質問追加・選択肢追加・画像添付）
- screens/survey_answer_screen.dart  : 回答画面
- screens/survey_result_screen.dart  : 集計結果グラフ

survey_create_screen.dart のバリデーション:
- タイトル空: 「タイトルを入力してください」
- 回答期限未設定: 「回答期限を設定してください」
- 質問0件: 「質問を1つ以上追加してください」
- 質問テキスト空: 「質問内容を入力してください」
- 選択肢1件以下: 「選択肢を2つ以上追加してください」

APIエラーのマッピング（survey_repository.dart の mapDioError）:
- API レスポンスの error.message がある場合はそれを優先して表示
- error.message がない場合のフォールバック:
  - 400 → 「入力内容を確認してください」
  - 401 → 「再度ログインしてください」
  - 403 → 「この操作の権限がありません」
  - 500 → 「サーバーエラーが発生しました。しばらく待ってから再試行してください」
```

---

## ファイル配置

### バックエンド

```
backend/internal/survey/
├── handler.go        # Create/Delete/List/Get/Answer/Results/UnansweredCount
├── image_handler.go  # UploadImage/DeleteImage/GetImage
├── model.go          # Survey/Question/Choice/Answer/SurveyResult 構造体
├── repository.go     # DB操作
└── service.go        # ビジネスロジック（バリデーション含む）
```

### フロントエンド

```
frontend/lib/features/survey/
├── data/
│   ├── models/survey_model.dart
│   └── repositories/survey_repository.dart
├── providers/survey_provider.dart
└── screens/
    ├── survey_list_screen.dart
    ├── survey_create_screen.dart
    ├── survey_answer_screen.dart
    └── survey_result_screen.dart
```

---

## 実装済みの詳細仕様

### サービス層バリデーション

```go
// service.go の Create メソッド
if input.Title == "" { return nil, fmt.Errorf("title is required") }
if len(input.Questions) == 0 { return nil, fmt.Errorf("at least one question is required") }
if !input.ExpiresAt.After(time.Now()) { return nil, fmt.Errorf("expires_at must be a future datetime") }
for i, q := range input.Questions {
    if q.QuestionText == "" { return nil, fmt.Errorf("question[%d]: text is required", i) }
    if q.QuestionType != "single" && q.QuestionType != "multiple" { ... }
    if len(q.Choices) < 2 { return nil, fmt.Errorf("question[%d]: at least 2 choices are required", i) }
}
```

エラーは HTTP 400 VALIDATION_ERROR として返す。

### 画像保存パス

```
{UPLOAD_DIR}/{association_id}/surveys/{survey_id}/{uuid}{ext}
```

### 未回答件数クエリ

```sql
SELECT COUNT(*)
FROM surveys s
WHERE s.association_id = $1
  AND s.deleted_at IS NULL
  AND s.expires_at > NOW()
  AND NOT EXISTS (
    SELECT 1 FROM survey_answers a
    WHERE a.survey_id = s.id AND a.user_id = $2
  )
```

### is_answered フラグ

一覧取得時、`survey_answers` を LEFT JOIN して回答済みかを判定。

### DB権限ミドルウェア

```
permMW.RequireFeature("surveys", "view")   → can_view チェック
permMW.RequireFeature("surveys", "create") → can_create チェック
permMW.RequireFeature("surveys", "delete") → can_delete チェック
```

### ホーム画面への統合

- `unansweredSurveyCountProvider` で未回答件数取得（バッジ表示）
- `recentSurveysProvider` で未回答かつ期限内を期限昇順で最大3件取得
- system_admin はアンケートセクション非表示
