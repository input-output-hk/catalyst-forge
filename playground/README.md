# Playground

A docker-compose based sandbox to run the API and the Frontend together for integration testing.

## Prerequisites

- Docker and Docker Compose
- Earthly installed and available as `earthly`

## Usage

- Build images and start the stack:

```bash
cd playground
just up
```

- Override API URL embedded in the frontend at build time (e.g., using the local proxy `https://api.localhost`):

```bash
cd playground
just up VITE_API_URL="https://api.localhost"
```

- View logs:

```bash
cd playground
just logs
```

- Stop and remove:

```bash
cd playground
just down
```

## Services

- Edge proxy: https://forge.localhost → routes `/api` to API, all else to Frontend
- API: http://localhost:5050 (direct), or via edge at `https://forge.localhost/api`
- Postgres: localhost:5432
- pgAdmin: http://localhost:5051

## Local HTTPS (mkcert)

1) Install mkcert and trust the local CA (see mkcert docs)
2) Generate certs:

```bash
mkdir -p .certs
cd .certs
mkcert forge.localhost
```

3) Add to `/etc/hosts`:

```
127.0.0.1 forge.localhost forge
```

4) Start the stack:

```bash
just up
```

Open `https://forge.localhost` in your browser.
