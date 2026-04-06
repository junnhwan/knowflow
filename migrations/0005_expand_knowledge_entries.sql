ALTER TABLE knowledge_entries
    ADD COLUMN IF NOT EXISTS title TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS summary TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS keywords JSONB NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS review_status TEXT NOT NULL DEFAULT 'draft',
    ADD COLUMN IF NOT EXISTS quality_score DOUBLE PRECISION NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS dedupe_hash TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS merged_into_id TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS disabled_at TIMESTAMPTZ NULL;

CREATE INDEX IF NOT EXISTS idx_knowledge_entries_user_review_status
    ON knowledge_entries (user_id, review_status);

CREATE INDEX IF NOT EXISTS idx_knowledge_entries_user_dedupe_hash
    ON knowledge_entries (user_id, dedupe_hash);
