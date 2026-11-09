.PHONY: test go-test go-lint web-test web-build compose-up compose-down integration

go-test:
	go vet ./...
	go test ./... -race -count=1

go-lint:
	golangci-lint run ./...

web-test:
	cd web && npm run lint && npm run typecheck && npm run test && npm run build

compose-up:
	docker compose --profile dev up -d --build

compose-down:
	docker compose down -v

integration:
	./scripts/ci-integration.sh

test: go-test web-test
