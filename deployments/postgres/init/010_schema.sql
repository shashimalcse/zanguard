CREATE TABLE tenants (
    id               VARCHAR(128) PRIMARY KEY,
    display_name     VARCHAR(256) NOT NULL,
    parent_tenant_id VARCHAR(128) REFERENCES tenants(id),
    status           VARCHAR(32) NOT NULL DEFAULT 'active',
    schema_mode      VARCHAR(32) NOT NULL DEFAULT 'own',
    shared_schema_ref VARCHAR(128),
    config           JSONB NOT NULL DEFAULT '{}',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at       TIMESTAMPTZ
);

CREATE INDEX idx_tenants_parent ON tenants(parent_tenant_id) WHERE parent_tenant_id IS NOT NULL;
CREATE INDEX idx_tenants_status ON tenants(status);

CREATE TABLE cross_tenant_grants (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    from_tenant_id        VARCHAR(128) NOT NULL REFERENCES tenants(id),
    to_tenant_id          VARCHAR(128) NOT NULL REFERENCES tenants(id),
    allowed_subject_types TEXT[] NOT NULL,
    allowed_relations     TEXT[] NOT NULL,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (from_tenant_id, to_tenant_id)
);

CREATE TABLE relation_tuples (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         VARCHAR(128) NOT NULL REFERENCES tenants(id),
    object_type       VARCHAR(128) NOT NULL,
    object_id         VARCHAR(256) NOT NULL,
    relation          VARCHAR(128) NOT NULL,
    subject_type      VARCHAR(128) NOT NULL,
    subject_id        VARCHAR(256) NOT NULL,
    subject_relation  VARCHAR(128),
    subject_tenant_id VARCHAR(128),
    source_system     VARCHAR(64),
    external_id       VARCHAR(512),
    attributes        JSONB NOT NULL DEFAULT '{}',
    expires_at        TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at        TIMESTAMPTZ
);

CREATE INDEX idx_tuples_lookup ON relation_tuples(tenant_id, object_type, object_id, relation)
    WHERE deleted_at IS NULL;
CREATE INDEX idx_tuples_subject ON relation_tuples(tenant_id, subject_type, subject_id, subject_relation)
    WHERE deleted_at IS NULL;
CREATE INDEX idx_tuples_source ON relation_tuples(tenant_id, source_system, external_id)
    WHERE source_system IS NOT NULL;
CREATE INDEX idx_tuples_expiry_active ON relation_tuples(tenant_id, expires_at)
    WHERE deleted_at IS NULL AND expires_at IS NOT NULL;
CREATE UNIQUE INDEX uq_tuples_identity_active ON relation_tuples
    (tenant_id, object_type, object_id, relation, subject_type, subject_id, COALESCE(subject_relation, ''))
    WHERE deleted_at IS NULL;

CREATE TABLE object_attributes (
    tenant_id   VARCHAR(128) NOT NULL REFERENCES tenants(id),
    object_type VARCHAR(128) NOT NULL,
    object_id   VARCHAR(256) NOT NULL,
    attributes  JSONB NOT NULL DEFAULT '{}',
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, object_type, object_id)
);

CREATE TABLE subject_attributes (
    tenant_id    VARCHAR(128) NOT NULL REFERENCES tenants(id),
    subject_type VARCHAR(128) NOT NULL,
    subject_id   VARCHAR(256) NOT NULL,
    attributes   JSONB NOT NULL DEFAULT '{}',
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, subject_type, subject_id)
);

CREATE TABLE changelog (
    sequence         BIGSERIAL PRIMARY KEY,
    tenant_id        VARCHAR(128) NOT NULL REFERENCES tenants(id),
    operation        VARCHAR(8) NOT NULL,
    object_type      VARCHAR(128) NOT NULL,
    object_id        VARCHAR(256) NOT NULL,
    relation         VARCHAR(128) NOT NULL,
    subject_type     VARCHAR(128) NOT NULL,
    subject_id       VARCHAR(256) NOT NULL,
    subject_relation VARCHAR(128),
    actor            VARCHAR(256),
    source           VARCHAR(32) NOT NULL,
    metadata         JSONB,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_changelog_tenant_seq ON changelog(tenant_id, sequence);
CREATE INDEX idx_changelog_tenant_time ON changelog(tenant_id, created_at);

CREATE TABLE schema_versions (
    id           BIGSERIAL PRIMARY KEY,
    tenant_id    VARCHAR(128),
    version      VARCHAR(64) NOT NULL,
    schema_yaml  TEXT NOT NULL,
    schema_hash  VARCHAR(64) NOT NULL,
    compiled_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_active    BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE UNIQUE INDEX uq_schema_versions_scope_version ON schema_versions
    (COALESCE(tenant_id, '__shared__'), version);
CREATE INDEX idx_schema_active ON schema_versions(tenant_id) WHERE is_active = TRUE;
