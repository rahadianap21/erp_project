-- =========================================================================
-- PELACAKAN ERROR (ERROR TRACKING)
-- =========================================================================
--
-- Menyimpan error backend (5xx/panic) dan error frontend yang dilaporkan
-- lewat POST /api/errors. Setiap error dikelompokkan berdasarkan fingerprint
-- (hash dari sumber + pesan yang sudah dinormalisasi + frame stack teratas).
--
--   * t_error_groups : satu baris per jenis error, dipakai untuk mendeteksi
--                      error BARU dan mengirim alert sekali per cooldown;
--   * t_error_events : riwayat kejadian individual untuk investigasi.
--
-- Tidak ada kolom untuk request body, query string, password, email, telepon,
-- atau data pelanggan. Pesan/stack sudah dibersihkan sebelum masuk ke sini.
--
-- Perintah: make migrateup

CREATE TABLE IF NOT EXISTS t_error_groups (
    fingerprint TEXT PRIMARY KEY,
    source VARCHAR(20) NOT NULL
        CHECK (source IN ('backend', 'frontend')),
    message TEXT NOT NULL,
    route TEXT NOT NULL DEFAULT '',
    occurrences BIGINT NOT NULL DEFAULT 1 CHECK (occurrences > 0),
    first_seen TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_seen TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_alerted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_error_groups_last_seen
    ON t_error_groups (last_seen DESC);

CREATE TABLE IF NOT EXISTS t_error_events (
    id BIGSERIAL PRIMARY KEY,
    fingerprint TEXT NOT NULL REFERENCES t_error_groups(fingerprint) ON DELETE CASCADE,
    source VARCHAR(20) NOT NULL
        CHECK (source IN ('backend', 'frontend')),
    level VARCHAR(20) NOT NULL DEFAULT 'error',
    message TEXT NOT NULL,
    stack TEXT NOT NULL DEFAULT '',
    component_stack TEXT NOT NULL DEFAULT '',
    route TEXT NOT NULL DEFAULT '',
    method VARCHAR(10) NOT NULL DEFAULT '',
    status_code INT NOT NULL DEFAULT 0,
    request_id VARCHAR(64) NOT NULL DEFAULT '',
    occurred_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_error_events_fingerprint_created
    ON t_error_events (fingerprint, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_error_events_created
    ON t_error_events (created_at DESC);
