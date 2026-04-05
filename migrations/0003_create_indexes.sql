CREATE INDEX IF NOT EXISTS idx_document_chunks_document_id ON document_chunks (document_id);
CREATE INDEX IF NOT EXISTS idx_document_chunks_user_id ON document_chunks (user_id);
CREATE INDEX IF NOT EXISTS idx_document_chunks_source_name ON document_chunks (source_name);
CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions (user_id);
CREATE INDEX IF NOT EXISTS idx_messages_session_id ON messages (session_id);
CREATE INDEX IF NOT EXISTS idx_knowledge_entries_session_id ON knowledge_entries (session_id);
CREATE INDEX IF NOT EXISTS idx_knowledge_entries_document_id ON knowledge_entries (document_id);
CREATE INDEX IF NOT EXISTS idx_document_chunks_content_trgm ON document_chunks USING gin (content gin_trgm_ops);
ALTER TABLE document_chunks
    ALTER COLUMN embedding TYPE VECTOR(64)
    USING embedding::vector(64);
CREATE INDEX IF NOT EXISTS idx_document_chunks_embedding ON document_chunks USING hnsw (embedding vector_cosine_ops);
