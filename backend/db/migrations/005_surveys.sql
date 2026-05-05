-- ═══════════════════════════════════════════════════════════
-- アンケート機能 + パスワードリセット
-- ═══════════════════════════════════════════════════════════

-- ─── アンケートテーブル ────────────────────────────────────────
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

-- ─── 質問テーブル ─────────────────────────────────────────────
CREATE TABLE survey_questions (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    survey_id     UUID NOT NULL REFERENCES surveys(id) ON DELETE CASCADE,
    question_text TEXT NOT NULL,
    question_type VARCHAR(20) NOT NULL CHECK (question_type IN ('single', 'multiple')),
    sort_order    INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX idx_survey_questions_survey_id ON survey_questions(survey_id);

-- ─── 選択肢テーブル ───────────────────────────────────────────
CREATE TABLE survey_choices (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    question_id UUID NOT NULL REFERENCES survey_questions(id) ON DELETE CASCADE,
    choice_text TEXT NOT NULL,
    sort_order  INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX idx_survey_choices_question_id ON survey_choices(question_id);

-- ─── 回答テーブル ─────────────────────────────────────────────
-- user_id + question_id + choice_id で一意（同じ選択肢を二重登録しない）
-- 複数選択の場合は同じ question_id で複数行
-- 単一選択の場合はサービス層で1件のみに制限する
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

-- ─── パスワードリセットトークンテーブル ──────────────────────
CREATE TABLE password_reset_tokens (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_password_reset_tokens_user_id ON password_reset_tokens(user_id);
CREATE INDEX idx_password_reset_tokens_token_hash ON password_reset_tokens(token_hash);
