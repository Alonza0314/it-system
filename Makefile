.PHONY: controller backend frontend runner clean run-controller tidy-controller test-controller lint-controller run-runner tidy-runner test-runner lint-runner

BACKEND_SRC := $(shell find controller/backend -name "*.go")
FRONTEND_SRC := $(shell find controller/frontend -type f ! -path "controller/frontend/dist/*" ! -path "controller/frontend/node_modules/*")
FRONTEND_STAMP := build/frontend/.stamp
RUNNER_SRC := $(shell find runner -name "*.go")

all: controller runner

controller: backend frontend

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

build/runner: $(RUNNER_SRC)
	@echo "[+] Building runner..."
	mkdir -p build
	cd runner && go build -o ../build/runner .
	@echo "[✔] Runner build finished"

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

runner:
	@if [ -f build/runner ]; then \
		if [ -z "$$(find runner -name '*.go' -newer build/runner)" ]; then \
			echo "[✔] runner is up-to-date, no build needed"; \
			exit 0; \
		fi; \
	fi; \
	$(MAKE) build/runner

clean:
	rm -rf build

run-controller::
	./build/controller -c config-controller.yaml

tidy-controller:
	cd controller/backend && go mod tidy

test-controller:
	cd controller/backend && go test -v ./...

lint-controller:
	cd controller/backend && golangci-lint run

run-runner::
	./build/runner -c config-runner.yaml

tidy-runner:
	cd runner && go mod tidy

test-runner:
	cd runner && go test -v ./...

lint-runner:
	cd runner && golangci-lint run