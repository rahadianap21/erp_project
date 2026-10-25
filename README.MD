# ERP Project

Sistem ERP dengan dua layer:

- **`go/`** — backend HTTP API (Go 1.27, PostgreSQL, sqlc, JWT + RBAC).
- **`next/`** — frontend (Next.js 16 App Router, TypeScript, Tailwind).

---

## Struktur Direktori

```
erp_project/
├── go/                         # Backend
│   ├── cmd/
│   │   ├── api/                # entrypoint API (config → pool → wiring → router)
│   │   └── seeder/             # seeding data awal
│   ├── db/
│   │   ├── init.sql            # snapshot skema untuk sqlc
│   │   ├── query.sql           # input sqlc
│   │   └── migrations/         # migrasi golang-migrate
│   ├── internal/
│   │   ├── config/             # pembacaan env + validasi
│   │   ├── domain/             # entity, port, typed error
│   │   ├── middleware/         # CORS, JWT, RBAC, rate limiter
│   │   ├── router/             # registrasi route
│   │   ├── handler/            # decode/encode HTTP + delegasi usecase
│   │   ├── usecase/            # aturan bisnis
│   │   └── repository/postgres # akses data + db/ (sqlc generated)
│   ├── .env.example
│   ├── .golangci.yml
│   └── Makefile
└── next/                       # Frontend
    ├── app/                    # route App Router + components/ colocated
    ├── components/             # ui/ (shadcn), layout/, shared/
    ├── features/               # auth, session
    ├── hooks/                  # use-api-list, use-mobile
    ├── lib/                    # api-client.ts, config.ts, permissions.ts, utils.ts
    ├── tests/                  # Playwright
    ├── proxy.ts                # gerbang auth + permission per halaman
    ├── playwright.config.ts
    ├── .prettierrc
    └── eslint.config.mjs
```

---

## Prasyarat

- Go 1.27+
- Node.js 20+ dan npm
- PostgreSQL
- `golang-migrate`, `sqlc`, `golangci-lint` (untuk perintah backend)

---

## Backend (`go/`)

```bash
cd go
cp .env.example .env          # isi JWT_SECRET (>=32 karakter) dan DATABASE_URL
make migrateup                # jalankan migrasi
make seed                     # opsional: data awal
make run                      # jalankan API di :8090
```

Variabel environment (lihat `go/.env.example`):

| Variabel       | Wajib | Default                 | Keterangan                                   |
| -------------- | ----- | ----------------------- | -------------------------------------------- |
| `JWT_SECRET`   | ya    | —                       | Minimal 32 karakter, untuk tanda tangan JWT. |
| `DATABASE_URL` | ya    | —                       | DSN PostgreSQL.                              |
| `PORT`         | tidak | `8090`                  | Port API.                                     |
| `CORS_ORIGIN`  | tidak | `http://localhost:3000` | Origin frontend yang diizinkan.               |

Perintah Makefile:

```bash
make lint              # golangci-lint
make fmt               # gofmt + golangci-lint fmt
make build             # go build ./...
make test              # build + vet + unit test
make test-integration  # butuh DATABASE_URL (tag `integration`)
make sqlc              # regenerasi kode sqlc
make migrateup / migratedown / migrateforceup
make seed
make run
```

---

## Frontend (`next/`)

```bash
cd next
npm install
npm run dev            # http://localhost:3000
```

Environment frontend:

- `NEXT_PUBLIC_API_URL` — base URL backend (default `http://localhost:8090`).
  Dipakai oleh rewrite `/api/*` di `next.config.ts` dan `lib/config.ts`.

Script:

```bash
npm run lint           # eslint
npm run lint:fix
npm run format         # prettier --write
npm run format:check
npm run typecheck      # tsc --noEmit
npm test               # unit test (node --test)
npm run build
npm run test:e2e       # Playwright (butuh backend Go di :8090)
npm run verify         # lint + typecheck + test + build
```

Catatan:

- Autentikasi memakai cookie `httpOnly` (`auth_token`); semua panggilan API dari
  client lewat `/api/proxy`. `proxy.ts` menjadi gerbang auth + permission per halaman.
- `next/package-lock.json` belum disinkronkan dengan `package.json`
  (penambahan `prettier`/`shadcn` dilakukan tanpa `npm`). Regenerasi dengan
  `npm install` lalu commit lockfile; setelah itu CI bisa kembali memakai `npm ci`.

---

## Pelacakan Error & Alert

Error backend (respons 5xx dan panic) serta error frontend/Next.js dilacak secara
built-in, tanpa layanan pihak ketiga.

- Backend menyimpan error ke tabel `t_error_groups` dan `t_error_events`
  (migrasi `000022_error_tracking`). Jalankan `make migrateup`.
- Frontend mengirim laporan ke route handler `/api/errors`, yang meneruskannya
  ke backend. Error server Next.js ditangkap lewat `next/instrumentation.ts`.
- Alert dikirim ke webhook bila error **baru** muncul, atau error yang sama
  masih berulang setelah `ERROR_ALERT_COOLDOWN` (default `24h`). Payload webhook
  memuat `text` (Slack) dan `content` (Discord).
- Data yang dikirim hanya pesan, stack, route (tanpa query), metode, dan status.
  Request body, header, cookie, query string, serta identitas pengguna tidak
  pernah dikumpulkan. Email, telepon, password, token, NIK/nomor panjang, dan
  nilai dari kunci sensitif (`password`, `email`, `phone`, `customer`, dst.)
  diredaksi di frontend dan sekali lagi di backend.

Environment backend (lihat `go/.env.example`):

| Variabel | Wajib | Default | Keterangan |
| --- | --- | --- | --- |
| `ERROR_ALERT_WEBHOOK_URL` | tidak | — | Webhook Slack/Discord/generik untuk alert. Kosong = alert mati, error tetap disimpan. |
| `ERROR_ALERT_COOLDOWN` | tidak | `24h` | Jeda minimum sebelum fingerprint yang sama memicu alert lagi. |
| `ERROR_INGEST_KEY` | tidak | — | Kunci bersama untuk `POST /api/errors`. Disarankan di produksi. |

Environment frontend (server-side, jangan pakai prefix `NEXT_PUBLIC_`):

```bash
# next/.env.local — samakan dengan backend
ERROR_INGEST_KEY=nilai-acak-yang-sama
```

Untuk melihat error terbaru:

```sql
SELECT fingerprint, source, message, occurrences, first_seen, last_seen
FROM t_error_groups
ORDER BY last_seen DESC
LIMIT 20;
```

## CI

Workflow ada di `.github/workflows/ci.yml`:

- **go** — `golangci-lint` + `go test ./...`.
- **next** — `lint` + `typecheck` + `test` + `build`.
- **playwright** — e2e (butuh backend + DB).

Catatan: dokumentasi internal (rencana reformasi/perbaikan, panduan pengguna, serta
berkas instruksi agen) sengaja tidak dipublikasikan di repo ini dan hanya disimpan lokal.
