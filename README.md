# check-in-ccf

Mobile-first children check-in app for church events.

## Architecture

```
check-in-ccf/
├── backend/          Go API server (Chi router)
│   ├── cmd/server/   main entry point
│   └── internal/
│       ├── auth/     JWT creation & middleware
│       ├── ct/       ChurchTools API client
│       └── handlers/ HTTP handlers
└── frontend/         Vue 3 + Vite + Tailwind CSS
    └── src/
        ├── api/      Typed fetch wrappers
        ├── router/   Vue Router (history mode)
        ├── stores/   Pinia stores
        └── views/    All page components
```

## Flow

1. **Admin** visits `/admin`, logs in with the `ADMIN_PASSWORD`.
2. Selects a child → sees parent contact details and linked children.
3. Confirms details → backend generates a signed JWT and returns a QR code PNG.  
   The QR encodes `FRONTEND_BASE_URL/checkin/<jwt>`.
4. **Parent** scans QR → opens `/checkin/<token>` — no login required.
5. Parent sees all their children with live check-in status and can toggle each.

## Backend setup

```bash
cd backend
cp .env.example .env          # fill in your values
go run ./cmd/server
```

Required env vars:

| Variable           | Description                                           |
|--------------------|-------------------------------------------------------|
| `CT_BASE_URL`      | e.g. `https://yourchurch.church.tools/api`            |
| `CT_API_TOKEN`     | ChurchTools login token (Settings → My Account)       |
| `JWT_SECRET`       | Long random string — keep secret                      |
| `ADMIN_PASSWORD`   | Password for the admin login screen                   |
| `FRONTEND_BASE_URL`| e.g. `https://checkin.yourchurch.org`                 |
| `PORT`             | Default `8080`                                        |

### ChurchTools configuration

- **Kids group type ID** — default `3`. Change `groupTypeId != 3` in `ct/client.go` if different.
- **Child relationship type ID** — default `1`. Change `rel.RelationshipTypeID == 1` in `ct/client.go`.
- **Check-in endpoints** — update `/checkin/checkin` and `/checkin/checkout` paths to match your CT version.

## Frontend setup

```bash
cd frontend
npm install
cp .env.example .env
npm run dev        # proxies /api → http://localhost:8080
```

## Production build

```bash
# Backend
go build -o server ./cmd/server

# Frontendv
npm run build      # outputs to frontend/dist/
```

Serve `frontend/dist/` as static files behind Nginx/Caddy. Point `/api` proxy to the Go server.
