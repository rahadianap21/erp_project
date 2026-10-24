-- Idempotensi HTTP untuk endpoint mutasi (penjualan, GRN, penutupan terima).
--
-- Klien mengirim header Idempotency-Key. Baris pertama mengklaim key; setelah
-- handler sukses, status/body respons disimpan agar retry tidak membuat
-- dokumen kedua. Key yang sama dengan request hash berbeda ditolak 409.

CREATE TABLE IF NOT EXISTS t_idempotency_keys (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES s_users(id) ON DELETE CASCADE,
    scope VARCHAR(255) NOT NULL,
    idempotency_key VARCHAR(255) NOT NULL,
    request_hash CHAR(64) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'IN_PROGRESS'
        CHECK (status IN ('IN_PROGRESS', 'COMPLETED')),
    response_status INT,
    response_body BYTEA,
    response_content_type VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP + INTERVAL '24 hours',
    CONSTRAINT t_idempotency_keys_unique UNIQUE (user_id, scope, idempotency_key),
    CONSTRAINT t_idempotency_keys_completed_has_status
        CHECK (status = 'IN_PROGRESS' OR response_status IS NOT NULL)
);

CREATE INDEX IF NOT EXISTS idx_t_idempotency_keys_expires
    ON t_idempotency_keys (expires_at);
