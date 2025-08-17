# Commands to build images with Earthly and run the playground compose

set shell := ["bash", "-cu"]

# Optional: set EARTHLY_BUILDKIT_HOST in your environment to point to a remote buildkit

@default: list

@list:
    just --list

# Build API Docker image via Earthly target
@build-api:
    cd ../services/api && earthly --config "" +docker
    # Restart API to pick up the new image
    docker compose up -d --no-deps --force-recreate api

# Build Frontend Docker image via Earthly target (allows override of API URL)
@build-frontend VITE_API_URL="http://api:5050":
    cd ../services/frontend && earthly --config "" +docker --VITE_API_URL="{{VITE_API_URL}}"
    # If the prod frontend container is running, restart it to pick up the new image
    if docker compose ps --services --filter status=running | grep -qx frontend; then \
        docker compose up -d --no-deps --force-recreate frontend; \
    else \
        echo "frontend (prod) not running; skipping restart"; \
    fi

# Build all images
@build VITE_API_URL="http://api:5050":
    just build-api
    just build-frontend VITE_API_URL="{{VITE_API_URL}}"

# Start the playground
@up VITE_API_URL="http://api:5050":
    (cd ../services/api && earthly --config "" +docker)
    just build VITE_API_URL="{{VITE_API_URL}}"
    docker compose up -d

# Stop and remove containers
@down:
    docker compose --profile dev down -v --remove-orphans || true
    docker compose down -v --remove-orphans || true

# Tail logs
@logs:
    docker compose logs -f --tail=200

# Start dev profile with live-reload frontend
@up-dev:
    (cd ../services/api && earthly --config "" +docker)
    docker compose --profile dev up -d

# Tail dev logs
@logs-dev:
    docker compose --profile dev logs -f --tail=200

# Generate local TLS certs with mkcert for forge.localhost
@certs:
    mkdir -p .certs
    if command -v mkcert >/dev/null 2>&1; then \
      (cd .certs && mkcert forge.localhost); \
    else \
      echo "mkcert not found. Install it from https://github.com/FiloSottile/mkcert and re-run: just certs"; \
    fi

# Add the local domain to the hosts file
@host:
    sudo hostctl add forge.localhost 127.0.0.1 && sudo hostctl -install

# Open the frontend in the default browser
@open:
    if command -v open >/dev/null 2>&1; then open "https://forge.localhost"; \
    elif command -v xdg-open >/dev/null 2>&1; then xdg-open "https://forge.localhost"; \
    elif command -v powershell.exe >/dev/null 2>&1; then powershell.exe Start-Process "https://forge.localhost"; \
    else echo "Please open https://forge.localhost in your browser"; fi

# Open the dev URL
@open-dev:
    if command -v open >/dev/null 2>&1; then open "https://forge.localhost"; \
    elif command -v xdg-open >/dev/null 2>&1; then xdg-open "https://forge.localhost"; \
    elif command -v powershell.exe >/dev/null 2>&1; then powershell.exe Start-Process "https://forge.localhost"; \
    else echo "Please open https://forge.localhost in your browser"; fi


