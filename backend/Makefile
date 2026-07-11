-include ../.env

PORT ?= 4005

run: kill-stale
	go run ./cmd

test:
	ENV_MODE=development go test -count=1 ./...

lint:
	golangci-lint fmt ./cmd/... ./internal/... && golangci-lint run ./cmd/... ./internal/...

# A stale `make run` keeps holding $(PORT) and answers with the previously built
# code; a second run that fails to bind only logs the error and serves nothing,
# so requests silently hit the old binary. Free the port first — killing the
# listener also makes its parent `go run` exit. Resolve the listener through /proc
# since a minimal environment may have no lsof/fuser.
kill-stale:
	@port_hex=$$(printf ':%04X' $(PORT)); \
	inode=$$(awk -v p="$$port_hex" '$$4 == "0A" && $$2 ~ p"$$" {print $$10}' /proc/net/tcp /proc/net/tcp6 | head -1); \
	[ -n "$$inode" ] || exit 0; \
	for pid in $$(ls /proc | grep -E '^[0-9]+$$'); do \
		if ls -l /proc/$$pid/fd 2>/dev/null | grep -q "socket:\[$$inode\]"; then \
			echo "freeing port $(PORT): killing stale backend pid $$pid"; \
			kill -9 $$pid 2>/dev/null || true; \
		fi; \
	done
