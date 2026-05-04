.PHONY: backend frontend clean run tidy test lint

BACKEND_SRC := $(shell find controller/backend -name "*.go")
FRONTEND_SRC := $(shell find controller/frontend -type f ! -path "controller/frontend/dist/*" ! -path "controller/frontend/node_modules/*")
FRONTEND_STAMP := build/frontend/.stamp

all: backend frontend

build/controller: $(BACKEND_SRC)
	@echo "[+] Building backend..."
	mkdir -p build
	cd controller/backend && go build -o ../../build/controller .
	@echo "[✔] Backend build finished"

build/frontend: $(FRONTEND_SRC)
	@echo "[+] Installing frontend deps..."
	cd controller/frontend && yarn install

	@echo "[+] Building frontend..."
	cd controller/frontend && yarn build

	@echo "[✔] Frontend build finished"
	@mkdir -p build/frontend
	@cp -r controller/frontend/dist/. build/frontend/
	@touch $(FRONTEND_STAMP)

backend:
	@if [ -f build/controller ]; then \
		if [ -z "$$(find controller/backend -name '*.go' -newer build/controller)" ]; then \
			echo "[✔] backend is up-to-date, no build needed"; \
			exit 0; \
		fi; \
	fi; \
	$(MAKE) build/controller

frontend:
	@if [ -f $(FRONTEND_STAMP) ]; then \
		if [ -z "$$(find controller/frontend -type f -newer $(FRONTEND_STAMP))" ]; then \
			echo "[✔] frontend is up-to-date, no build needed"; \
			exit 0; \
		fi; \
	fi; \
	$(MAKE) build/frontend

clean:
	rm -rf build

run:
	./build/controller -c config.yaml

tidy:
	cd controller/backend && go mod tidy

test:
	cd controller/backend && go test -v ./...

lint:
	cd controller/backend && golangci-lint run