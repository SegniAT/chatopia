# Load variables from .env file automatically
set dotenv-load := true

APP_PORT := env_var("APP_PORT")

# Print available commands
default:
    @just --list

# ==================================================================================== #
# INFRASTRUCTURE
# ==================================================================================== #

# Spin up local docker development dependencies
infra-up:
    docker compose up -d redis peerjs-server alloy prometheus loki grafana

# Tear down all local docker containers
infra-down:
    docker compose down

# ==================================================================================== #
# DEVELOPMENT
# ==================================================================================== #

# Run the cmd/web application directly
run-web:
    go run ./cmd/web

# Run templ generation in watch mode
live-templ:
    ~/go/bin/templ generate --watch --proxy="http://localhost:{{APP_PORT}}" -v

# Run air to detect Go file changes and hot-reload
live-server:
    ~/go/bin/air \
        --build.cmd "go build -o tmp/bin/main ./cmd/web/" --build.bin "tmp/bin/main" --build.delay "100" \
        --build.exclude_dir "node_modules" \
        --build.include_ext "go" \
        --build.stop_on_error "false" \
        --misc.clean_on_exit true

# Run tailwindcss watcher
live-tailwind:
    npx tailwindcss -i ./ui/style.css -o ./public/assets/style.css --minify --watch

# Run esbuild watcher
live-esbuild:
    npx esbuild ./ui/index.ts --bundle --outdir=./public/assets/ --watch=forever

# Sync asset updates through templ proxy
live-sync-assets:
    ~/go/bin/air \
        --build.cmd "~/go/bin/templ generate --notify-proxy" \
        --build.bin "true" \
        --build.delay "100" \
        --build.exclude_dir "node_modules" \
        --build.include_dir "public" \
        --build.include_ext "js,css"

# Start all 5 watch processes in parallel
live:
    just --parallel live-templ live-server live-tailwind live-esbuild live-sync-assets

# ==================================================================================== #
# QUALITY CONTROL
# ==================================================================================== #

# Tidy dependencies, format, vet, and run tests
audit: vendor
    @echo 'Formatting code...'
    go fmt ./...
    @echo 'Vetting code...'
    go vet ./...
    staticcheck ./...
    @echo 'Running tests...'
    go test -race -vet=off ./...

# Tidy and vendor dependencies
vendor:
    @echo 'Tidying and verifying module dependencies...'
    go mod tidy
    go mod verify
    @echo 'Vendoring dependencies...'
    go mod vendor

# ==================================================================================== #
# BUILD
# ==================================================================================== #

# Build the cmd/web application for both local and production environments
build-web:
    @echo 'Building cmd/api...'
    go build -ldflags='-s' -o=./bin/web ./cmd/web/
    GOOS=linux GOARCH=amd64 go build -ldflags='-s' -o=./bin/linux_amd64/web ./cmd/web/

# ==================================================================================== #
# PRODUCTION DEPLOYMENT
# ==================================================================================== #

# Build and start the entire stack in production mode on the server
prod-up:
    # -f compose.yaml avoids merging local development 'docker-compose.override.yml' changes.
    # --env-file .env.docker isolates production environment variables from local .env values.
    # 'set -a' automatically exports all variables sourced from the file into the shell. We must override the .env sourced by Justfile.
    # '.' is the POSIX-compliant way to source a file, overriding the local .env values.
    set -a; . ./.env.docker; set +a; docker compose --env-file .env.docker -f compose.yaml up -d --build

# Stop and remove the production stack on the server
prod-down:
    # Explicitly matches the same configuration file and environment context used during spin-up.
    set -a; . ./.env.docker; set +a; docker compose --env-file .env.docker -f compose.yaml down
