# Frontend — CLAUDE.md

## Running the Frontend

### Prerequisites

- Node.js 22+
- API server running on `localhost:8000` (for dual UI API Server mode)

### Install Dependencies

```bash
cd components/frontend
npm install
```

### Development Server

```bash
# Default — connects to K8s backend at localhost:8080
npm run dev

# With API server integration (dual UI toggle)
NEXT_PUBLIC_AMBIENT_API_URL=http://localhost:8000/api/ambient-api-server/v1 npm run dev
```

Frontend starts at `http://localhost:3000`.

### Production Build

```bash
npm run build   # Must pass with 0 errors, 0 warnings
npm run start   # Serve the production build
```

### Lint

```bash
npm run lint
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `NEXT_PUBLIC_AMBIENT_API_URL` | `http://localhost:8000/api/ambient-api-server/v1` | API server base URL for SDK client |
| `BACKEND_URL` | `http://localhost:8080/api` | K8s backend URL (BFF proxy) |
| `OC_TOKEN` | — | OpenShift token for server-side auth |
| `NEXT_PUBLIC_E2E_TOKEN` | — | Token override for E2E testing |
| `FEEDBACK_URL` | — | Optional feedback link in navbar |
