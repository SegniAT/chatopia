# Include variables from the .envrc file
include .envrc

# ==================================================================================== #
# HELPERS
# ==================================================================================== #

## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

.PHONY: confirm
confirm:
	@echo -n 'Are you sure? [y/N] ' && read ans && [ $${ans: -N} = y ]



# ==================================================================================== #
# DEVELOPMENT
# ==================================================================================== #

## run/web: run the cmd/web application
.PHONY: run/web
run/web:
	go run ./cmd/web -port=${PORT}

## live/templ: run templ generation in watch mode to detect all .templ files and re-create _templ.txt files on change, then send reload event to browser
.PHONY: live/templ
live/templ:
	~/go/bin/templ generate --watch --proxy="http://localhost:${PORT}" -v

## live/server: run air to detect any go file changes to re-build and re-run the server.
.PHONY: live/server
live/server:
	~/go/bin/air \
	--build.cmd "go build -o tmp/bin/main ./cmd/web/" --build.bin "tmp/bin/main" --build.delay "100" \
	--build.args_bin "-port ${PORT} -secret ${SECRET} -env ${ENV}" \
	--build.exclude_dir "node_modules" \
	--build.include_ext "go" \
	--build.stop_on_error "false" \
	--misc.clean_on_exit true 

## live/tailwind: run tailwindcss to generate the styles.css bundle in watch mode.
.PHONY: live/tailwind
live/tailwind:
	npx tailwindcss -i ./ui/style.css -o ./public/assets/style.css --minify --watch

## live/esbuild: run esbuild to generate the index.js bundle in watch mode.
.PHONY: live/esbuild
live/esbuild:
	npx esbuild ./ui/index.ts --bundle --outdir=./public/assets/ --watch=forever

## live/sync_assets: watch for any js or css change in the assets/ folder, then reload the browser via templ proxy.
.PHONY: live/sync_assets
live/sync_assets:
	~/go/bin/air \
	--build.cmd "~/go/bin/templ generate --notify-proxy" \
	--build.bin "true" \
	--build.delay "100" \
	--build.exclude_dir "node_modules" \
	--build.include_dir "public" \
	--build.include_ext "js,css"

## live: start all 5 watch processes in parallel.
.PHONY: live
live: 
	make -j5 live/templ live/server live/tailwind live/esbuild live/sync_assets

# ==================================================================================== #
# QUALITY CONTROL 
# ==================================================================================== #

## audit: tidy and vendor dependencies and format, vet and test all code
.PHONY: audit
audit: vendor
	@echo 'Formatting code...'
	go fmt ./...
	@echo 'Vetting code...'
	go vet ./...
	staticcheck ./...
	@echo 'Running tests...'
	go test -race -vet=off ./...

## vendor: tidy and vendor dependencies
.PHONY: vendor
vendor:
	@echo 'Tidying and verifying module dependendencies...'
	go mod tidy
	go mod verify
	@echo 'Vendoring dependencies...'
	go mod vendor


# ==================================================================================== #
# BUILD
# ==================================================================================== #

## build/web: build the cmd/web application
.PHONY: build/web
build/web:
	@echo 'Building cmd/api...'
	go build -ldflags='-s' -o=./bin/web ./cmd/web/
	GOOS=linux GOARCH=amd64 go build -ldflags='-s' -o=./bin/linux_amd64/web ./cmd/web/
