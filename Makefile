.PHONY: backend frontend clean run tidy test lint

BACKEND_SRC := $(shell find backend -name "*.go")
FRONTEND_SRC := $(shell find frontend -type f ! -path "frontend/dist/*" ! -path "frontend/node_modules/*")
FRONTEND_STAMP := build/frontend/.stamp

all: backend frontend

build/system: $(BACKEND_SRC)
	@echo "[+] Building backend..."
	mkdir -p build
	cd backend && go build -o ../build/system .
	@echo "[✔] Backend build finished"

build/frontend: $(FRONTEND_SRC)
	@echo "[+] Installing frontend deps..."
	cd frontend && yarn install

	@echo "[+] Building frontend..."
	cd frontend && yarn build

	@echo "[✔] Frontend build finished"
	@mkdir -p build/frontend
	@cp -r frontend/dist/. build/frontend/
	@touch $(FRONTEND_STAMP)

backend:
	@if [ -f build/system ]; then \
		if [ -z "$$(find backend -name '*.go' -newer build/system)" ]; then \
			echo "[✔] backend is up-to-date, no build needed"; \
			exit 0; \
		fi; \
	fi; \
	$(MAKE) build/system

frontend:
	@if [ -f $(FRONTEND_STAMP) ]; then \
		if [ -z "$$(find frontend -type f -newer $(FRONTEND_STAMP))" ]; then \
			echo "[✔] frontend is up-to-date, no build needed"; \
			exit 0; \
		fi; \
	fi; \
	$(MAKE) build/frontend

clean:
	rm -rf build
	rm -rf build

run:
	./build/system -c config.yaml

tidy:
	cd backend && go mod tidy

test:
	cd backend && go test -v ./...

lint:
	cd backend && golangci-lint run