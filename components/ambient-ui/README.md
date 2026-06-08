# Ambient UI

Operations console for the Ambient Code Platform. Next.js BFF with OIDC authentication (Keycloak), shadcn/ui components, and Red Hat design system.

## Local Development

Prerequisites: Kind cluster running (`make kind-up`).

```bash
make dev COMPONENT=ambient-ui
```

This single command:
1. Sets kubectl context to the correct Kind cluster
2. Port-forwards the API server and Keycloak
3. Patches Keycloak's `KC_HOSTNAME` so OIDC redirects work locally
4. Generates `.env.local` with correct SSO config
5. Starts the Next.js dev server on `http://localhost:3001`

Login with the Keycloak admin credentials (`admin` / `admin`) or any user configured in the `ambient-code` realm.

### Other useful commands

```bash
make dev-env-ambient-ui     # Regenerate .env.local without starting the dev server
make kind-reload-ambient-ui # Rebuild image and redeploy to Kind
make kind-status            # Show cluster ports (including Keycloak)
```

## Testing

```bash
npm test              # Unit tests (vitest, ~3s)
npm run test:watch    # Watch mode
npm run test:coverage # With coverage

# E2E (requires: npm install && npx playwright install chromium)
npm run test:e2e
```

## Architecture

- **BFF pattern**: Next.js server handles OIDC auth, proxies API calls with JWT injection. Browser never sees raw tokens.
- **Port/Adapter**: Domain types in `src/domain/`, ports in `src/ports/`, SDK adapters in `src/adapters/`. Components consume ports, never SDK types.
- **iron-session**: Per-user encrypted cookie sessions for connection context and auth state.
