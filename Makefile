.PHONY: registry-check registry-sync fmt-check vet build

registry-check:
	@diff -u schema/asset-registry.json client/src/assets/registry.json && echo "registry sync OK" || (echo "registry mismatch: schema/asset-registry.json vs client/src/assets/registry.json" && exit 1)

registry-sync:
	cp schema/asset-registry.json client/src/assets/registry.json
	@echo "synced schema/asset-registry.json -> client/src/assets/registry.json"

fmt-check:
	@test -z "$$(cd server && gofmt -l .)" || (echo "gofmt needed:" && cd server && gofmt -l . && exit 1)

vet: fmt-check
	cd server && go vet ./...
	cd client && npx tsc --noEmit
	cd teacher-interface && npm run lint

build:
	cd server && go build -o /tmp/glam-server .
	cd client && npm run build
	cd teacher-interface && npm run build
