-- Candidate applications table
CREATE TABLE IF NOT EXISTS candidate_applications (
    id VARCHAR(36) PRIMARY KEY,
    first_name VARCHAR(255) NOT NULL,
    last_name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    phone VARCHAR(20),
    position VARCHAR(255),
    source VARCHAR(50) NOT NULL,
    source_ref_id VARCHAR(255) NOT NULL UNIQUE,
    raw_payload TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    processed_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_source ON candidate_applications (source);
CREATE INDEX IF NOT EXISTS idx_created_at ON candidate_applications (created_at);

-- Transactional outbox
CREATE TABLE IF NOT EXISTS outbox_events (
    id VARCHAR(36) PRIMARY KEY,
    application_id VARCHAR(36) NOT NULL REFERENCES candidate_applications(id) ON DELETE CASCADE,
    event_type VARCHAR(100) NOT NULL,
    payload TEXT NOT NULL,
    published BOOLEAN DEFAULT false,
    published_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_published ON outbox_events (published);
CREATE INDEX IF NOT EXISTS idx_created_at_outbox ON outbox_events (created_at);

-- Idempotency keys
CREATE TABLE IF NOT EXISTS idempotency_keys (
    id SERIAL PRIMARY KEY,
    request_id VARCHAR(255) UNIQUE NOT NULL,
    application_id VARCHAR(36) NOT NULL REFERENCES candidate_applications(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_request_id ON idempotency_keys (request_id);
CREATE INDEX IF NOT EXISTS idx_application_id ON idempotency_keys (application_id);
