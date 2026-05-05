-- アンケート添付画像テーブル
CREATE TABLE IF NOT EXISTS survey_images (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    survey_id      UUID        NOT NULL REFERENCES surveys(id) ON DELETE CASCADE,
    association_id UUID        NOT NULL REFERENCES associations(id),
    filename       TEXT        NOT NULL,
    storage_path   TEXT        NOT NULL,
    sort_order     INT         NOT NULL DEFAULT 0,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_survey_images_survey_id ON survey_images(survey_id);
