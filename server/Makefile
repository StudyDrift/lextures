# Common tasks for the Go API (`server/`). Run with DATABASE_URL for full tests (see CI).
.PHONY: test testshort lint build docker-build stubs intro-course-validate marketplace-courses-validate check-openrouter-coupling route-inventory route-inventory-update
stubs:
	python3 scripts/gen_stubs.py
test:
	go test ./internal/... -count=1 -timeout=5m -coverprofile=coverage.out -covermode=atomic
	go tool cover -func=coverage.out | tail -1

testshort:
	go test ./... -count=1 -short -timeout=1m

# TD.1 — route inventory harness (no DB).
route-inventory:
	go test ./internal/httpserver/ -run 'TestRouteInventoryPrint$$' -count=1 -v

route-inventory-update:
	UPDATE_GOLDEN=1 go test ./internal/httpserver/ -run 'TestRouteInventory$$' -count=1 -v

check-openrouter-coupling:
	bash ../scripts/check-openrouter-coupling.sh

lint:
	golangci-lint run ./...

intro-course-validate:
	go run ./cmd/intro-course-validate

marketplace-courses-validate:
	go run ./cmd/marketplace-courses-validate

build:
	go build -o bin/server ./cmd/server

docker-build:
	docker build -t lextures-server-go:local -f Dockerfile .
