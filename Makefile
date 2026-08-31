.PHONY: registry-check registry-sync vet build

registry-check:
	@diff -u schema/asset-registry.json client/src/assets/registry.json && echo "registry sync OK" || (echo "registry mismatch: schema/asset-registry.json vs client/src/assets/registry.json" && exit 1)

registry-sync:
	cp schema/asset-registry.json client/src/assets/registry.json
	@echo "synced schema/asset-registry.json -> client/src/assets/registry.json"

vet:
	go vet ./...
	cd client && npx tsc --noEmit

build:
	go build -o /tmp/glam-server ./server
	cd client && npm run build
