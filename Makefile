.PHONY: dev web build test test-go test-web docker up down container-up container-down clean

# Run the Go server locally (UI must be built first, or use `make dev` + vite).
run:
	cd server && go run ./cmd/boop

# Start the Vite dev server (proxies /api to :8080).
dev:
	cd server/web && npm run dev

# Build the web UI into the Go embed directory.
web:
	cd server/web && npm ci && npm run build

# Build a single binary with the UI embedded.
build: web
	cd server && CGO_ENABLED=0 go build -ldflags="-s -w" -o ../bin/boop ./cmd/boop

test: test-go test-web

test-go:
	cd server && go vet ./... && go test ./...

test-web:
	cd server/web && npm run check && npx vitest run

docker:
	docker compose build

up:
	docker compose up -d

down:
	docker compose down

# Apple `container` (macOS 26+) has no compose support, so the single boop
# service is translated by hand. Note: no `restart: unless-stopped` equivalent.
container-up:
	container system start
	mkdir -p data
	container build -t ghcr.io/chrisgreg/boop:latest server
	container run -d --name boop \
		-p 8080:8080 \
		-v "$(PWD)/data:/data" \
		-e BOOP_DATABASE_PATH=/data/boop.db \
		ghcr.io/chrisgreg/boop:latest

container-down:
	-container stop boop
	-container rm boop

clean:
	rm -rf bin server/internal/web/dist/assets server/internal/web/dist/index.html server/web/node_modules
