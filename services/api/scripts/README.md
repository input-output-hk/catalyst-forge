# Local TLS Proxy (Caddy)

This stack includes an optional Caddy reverse proxy to terminate TLS for `api.localhost` and forward to the API container.

## Quick start

1) Generate locally-trusted certificates (requires `mkcert`):

```bash
mkdir -p services/api/.certs
cd services/api/.certs
mkcert api.localhost
```

This creates `api.localhost.pem` and `api.localhost-key.pem` in `.certs/`.

2) Start the stack with the proxy profile:

```bash
docker compose -f services/api/docker-compose.yml --profile proxy up -d
```

3) Configure frontend to call the proxy base URL:

- Set `VITE_API_URL=https://api.localhost`

The API compose already sets:

- `PUBLIC_BASE_URL=http://localhost:5050` (internal)
- `AUTH_ALLOWED_WEB_ORIGINS=http://localhost:5173,https://localhost:5173`

When using the proxy, calls from `https://localhost:5173` to `https://api.localhost` will succeed without extra CORS tweaks.

## Notes

- Ensure `/etc/hosts` has `127.0.0.1 api.localhost`.
- If you prefer, add `https://127.0.0.1:5173` to `AUTH_ALLOWED_WEB_ORIGINS`.
- For clean shutdown: `docker compose --profile proxy down`.