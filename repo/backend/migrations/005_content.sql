BEGIN;

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS categories (
  category_id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id            UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
  parent_category_id   UUID,
  name                 TEXT NOT NULL,
  created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (tenant_id, category_id),
  UNIQUE (tenant_id, parent_category_id, name),
  FOREIGN KEY (tenant_id, parent_category_id) REFERENCES categories(tenant_id, category_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS tags (
  tag_id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id            UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
  name                 TEXT NOT NULL,
  created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (tenant_id, tag_id),
  UNIQUE (tenant_id, name)
);

CREATE TABLE IF NOT EXISTS documents (
  document_id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id            UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
  title                TEXT NOT NULL,
  summary              TEXT,
  difficulty           SMALLINT NOT NULL CHECK (difficulty BETWEEN 1 AND 5),
  duration_minutes     INTEGER NOT NULL CHECK (duration_minutes BETWEEN 5 AND 480),
  current_version_no   INTEGER NOT NULL DEFAULT 0,
  is_archived          BOOLEAN NOT NULL DEFAULT FALSE,
  created_by_user_id   UUID NOT NULL,
  updated_by_user_id   UUID,
  created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  search_vector        TSVECTOR GENERATED ALWAYS AS (
    setweight(to_tsvector('simple', coalesce(title, '')), 'A') ||
    setweight(to_tsvector('simple', coalesce(summary, '')), 'B')
  ) STORED,
  UNIQUE (tenant_id, document_id),
  FOREIGN KEY (tenant_id, created_by_user_id) REFERENCES users(tenant_id, user_id) ON DELETE RESTRICT,
  FOREIGN KEY (tenant_id, updated_by_user_id) REFERENCES users(tenant_id, user_id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_documents_search ON documents USING GIN (search_vector);

CREATE TABLE IF NOT EXISTS document_versions (
  document_version_id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id             UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
  document_id           UUID NOT NULL,
  version_no            INTEGER NOT NULL CHECK (version_no > 0),
  file_name             TEXT NOT NULL,
  storage_path          TEXT NOT NULL,
  mime_type             TEXT NOT NULL,
  file_size_bytes       BIGINT NOT NULL CHECK (file_size_bytes > 0),
  sha256_checksum       TEXT NOT NULL,
  extracted_text        TEXT,
  created_by_user_id    UUID NOT NULL,
  created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  search_vector         TSVECTOR GENERATED ALWAYS AS (
    to_tsvector('simple', coalesce(extracted_text, ''))
  ) STORED,
  UNIQUE (tenant_id, document_version_id),
  UNIQUE (tenant_id, document_id, version_no),
  FOREIGN KEY (tenant_id, document_id) REFERENCES documents(tenant_id, document_id) ON DELETE CASCADE,
  FOREIGN KEY (tenant_id, created_by_user_id) REFERENCES users(tenant_id, user_id) ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS idx_document_versions_search ON document_versions USING GIN (search_vector);
CREATE INDEX IF NOT EXISTS idx_document_versions_checksum ON document_versions (tenant_id, sha256_checksum);

CREATE TABLE IF NOT EXISTS document_categories (
  tenant_id            UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
  document_id          UUID NOT NULL,
  category_id          UUID NOT NULL,
  PRIMARY KEY (tenant_id, document_id, category_id),
  FOREIGN KEY (tenant_id, document_id) REFERENCES documents(tenant_id, document_id) ON DELETE CASCADE,
  FOREIGN KEY (tenant_id, category_id) REFERENCES categories(tenant_id, category_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS document_tags (
  tenant_id            UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
  document_id          UUID NOT NULL,
  tag_id               UUID NOT NULL,
  PRIMARY KEY (tenant_id, document_id, tag_id),
  FOREIGN KEY (tenant_id, document_id) REFERENCES documents(tenant_id, document_id) ON DELETE CASCADE,
  FOREIGN KEY (tenant_id, tag_id) REFERENCES tags(tenant_id, tag_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS upload_sessions (
  upload_id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id            UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
  document_id          UUID,
  file_name            TEXT NOT NULL,
  mime_type            TEXT NOT NULL,
  total_chunks         INTEGER NOT NULL CHECK (total_chunks > 0),
  chunk_size_bytes     INTEGER NOT NULL CHECK (chunk_size_bytes > 0),
  expires_at           TIMESTAMPTZ NOT NULL,
  completed_at         TIMESTAMPTZ,
  created_by_user_id   UUID NOT NULL,
  created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (tenant_id, upload_id),
  FOREIGN KEY (tenant_id, document_id) REFERENCES documents(tenant_id, document_id) ON DELETE SET NULL,
  FOREIGN KEY (tenant_id, created_by_user_id) REFERENCES users(tenant_id, user_id) ON DELETE RESTRICT
);

CREATE TABLE IF NOT EXISTS upload_chunks (
  tenant_id            UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
  upload_id            UUID NOT NULL,
  chunk_index          INTEGER NOT NULL CHECK (chunk_index >= 0),
  storage_path         TEXT NOT NULL,
  chunk_size_bytes     INTEGER NOT NULL CHECK (chunk_size_bytes > 0),
  sha256_checksum      TEXT NOT NULL,
  received_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (tenant_id, upload_id, chunk_index),
  FOREIGN KEY (tenant_id, upload_id) REFERENCES upload_sessions(tenant_id, upload_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS document_share_links (
  share_link_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id            UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
  document_id          UUID NOT NULL,
  document_version_id  UUID,
  token_hash           TEXT NOT NULL,
  expires_at           TIMESTAMPTZ NOT NULL,
  created_by_user_id   UUID NOT NULL,
  created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  used_count           INTEGER NOT NULL DEFAULT 0,
  UNIQUE (token_hash),
  FOREIGN KEY (tenant_id, document_id) REFERENCES documents(tenant_id, document_id) ON DELETE CASCADE,
  FOREIGN KEY (tenant_id, document_version_id) REFERENCES document_versions(tenant_id, document_version_id) ON DELETE SET NULL,
  FOREIGN KEY (tenant_id, created_by_user_id) REFERENCES users(tenant_id, user_id) ON DELETE RESTRICT
);

CREATE TABLE IF NOT EXISTS document_duplicate_flags (
  duplicate_id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id            UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
  left_document_id     UUID NOT NULL,
  right_document_id    UUID NOT NULL,
  checksum             TEXT NOT NULL,
  merge_candidate      BOOLEAN NOT NULL DEFAULT FALSE,
  flagged_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  flagged_by_user_id   UUID,
  UNIQUE (tenant_id, left_document_id, right_document_id, checksum),
  FOREIGN KEY (tenant_id, left_document_id) REFERENCES documents(tenant_id, document_id) ON DELETE CASCADE,
  FOREIGN KEY (tenant_id, right_document_id) REFERENCES documents(tenant_id, document_id) ON DELETE CASCADE,
  FOREIGN KEY (tenant_id, flagged_by_user_id) REFERENCES users(tenant_id, user_id) ON DELETE SET NULL,
  CHECK (left_document_id <> right_document_id)
);

ALTER TABLE categories ENABLE ROW LEVEL SECURITY;
ALTER TABLE tags ENABLE ROW LEVEL SECURITY;
ALTER TABLE documents ENABLE ROW LEVEL SECURITY;
ALTER TABLE document_versions ENABLE ROW LEVEL SECURITY;
ALTER TABLE document_categories ENABLE ROW LEVEL SECURITY;
ALTER TABLE document_tags ENABLE ROW LEVEL SECURITY;
ALTER TABLE upload_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE upload_chunks ENABLE ROW LEVEL SECURITY;
ALTER TABLE document_share_links ENABLE ROW LEVEL SECURITY;
ALTER TABLE document_duplicate_flags ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation_categories ON categories;
CREATE POLICY tenant_isolation_categories ON categories
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

DROP POLICY IF EXISTS tenant_isolation_tags ON tags;
CREATE POLICY tenant_isolation_tags ON tags
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

DROP POLICY IF EXISTS tenant_isolation_documents ON documents;
CREATE POLICY tenant_isolation_documents ON documents
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

DROP POLICY IF EXISTS tenant_isolation_document_versions ON document_versions;
CREATE POLICY tenant_isolation_document_versions ON document_versions
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

DROP POLICY IF EXISTS tenant_isolation_document_categories ON document_categories;
CREATE POLICY tenant_isolation_document_categories ON document_categories
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

DROP POLICY IF EXISTS tenant_isolation_document_tags ON document_tags;
CREATE POLICY tenant_isolation_document_tags ON document_tags
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

DROP POLICY IF EXISTS tenant_isolation_upload_sessions ON upload_sessions;
CREATE POLICY tenant_isolation_upload_sessions ON upload_sessions
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

DROP POLICY IF EXISTS tenant_isolation_upload_chunks ON upload_chunks;
CREATE POLICY tenant_isolation_upload_chunks ON upload_chunks
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

DROP POLICY IF EXISTS tenant_isolation_share_links ON document_share_links;
CREATE POLICY tenant_isolation_share_links ON document_share_links
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

DROP POLICY IF EXISTS tenant_isolation_duplicate_flags ON document_duplicate_flags;
CREATE POLICY tenant_isolation_duplicate_flags ON document_duplicate_flags
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

COMMIT;
