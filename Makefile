# gOpenID — Makefile
# Merkezi kimlik ve yetkilendirme sunucusu (Go backend + React frontend)

# ----------------------------------------------------------------------------
# Configuration
# ----------------------------------------------------------------------------
FRONTEND_DIR := frontend
DIST_DIR     := internal/web/dist
BINARY       := bin/gopenid
PKG          := ./cmd/server
PM           := npm

# Use a single shell per recipe so `cd` persists across lines.
.ONESHELL:
.DEFAULT_GOAL := help

# ----------------------------------------------------------------------------
# Help
# ----------------------------------------------------------------------------
.PHONY: help
help: ## Bu yardım metnini göster
	@echo "gOpenID — kullanılabilir komutlar:"
	@echo ""
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'
	@echo ""

# ----------------------------------------------------------------------------
# Setup
# ----------------------------------------------------------------------------
.PHONY: install
install: install-frontend install-backend ## Tüm bağımlılıkları kur (frontend + backend)

.PHONY: install-frontend
install-frontend: ## Frontend npm bağımlılıklarını kur
	cd $(FRONTEND_DIR) && $(PM) install

.PHONY: install-backend
install-backend: ## Go modüllerini indir
	go mod download

.PHONY: tidy
tidy: ## go.mod / go.sum düzenle
	go mod tidy

# ----------------------------------------------------------------------------
# Development
# ----------------------------------------------------------------------------
.PHONY: dev
dev: ## Backend + frontend dev sunucularını birlikte çalıştır
	@echo "Backend :8080 ve frontend :5173 başlatılıyor (Ctrl-C ile durdurun)..."
	@trap 'kill 0' INT TERM EXIT; \
	go run $(PKG) & \
	cd $(FRONTEND_DIR) && $(PM) run dev & \
	wait

.PHONY: dev-backend
dev-backend: ## Sadece Go backend'i çalıştır (:8080)
	go run $(PKG)

.PHONY: dev-frontend
dev-frontend: ## Sadece Vite frontend dev sunucusunu çalıştır (:5173)
	cd $(FRONTEND_DIR) && $(PM) run dev

# ----------------------------------------------------------------------------
# Build
# ----------------------------------------------------------------------------
.PHONY: build
build: build-frontend build-backend ## Frontend + backend üretim derlemesi

.PHONY: build-frontend
build-frontend: ## Frontend'i derle ($(DIST_DIR) içine)
	cd $(FRONTEND_DIR) && $(PM) run build

.PHONY: build-backend
build-backend: build-frontend ## Go binary'sini derle (frontend gömülü)
	mkdir -p bin
	go build -o $(BINARY) $(PKG)
	@echo "Derlendi: $(BINARY)"

.PHONY: run
run: build-frontend ## Üretim modunda çalıştır (frontend derlenip Go ile sunulur)
	go run $(PKG)

# ----------------------------------------------------------------------------
# Quality
# ----------------------------------------------------------------------------
.PHONY: test
test: test-backend ## Tüm testleri çalıştır

.PHONY: test-backend
test-backend: ## Go testlerini çalıştır
	go test ./...

.PHONY: test-verbose
test-verbose: ## Go testlerini ayrıntılı çalıştır
	go test ./... -v

.PHONY: lint
lint: lint-backend lint-frontend ## Tüm linterları çalıştır

.PHONY: lint-backend
lint-backend: ## go vet çalıştır
	go vet ./...

.PHONY: lint-frontend
lint-frontend: ## ESLint + TypeScript tip kontrolü
	cd $(FRONTEND_DIR) && $(PM) run lint && npx tsc -b

.PHONY: fmt
fmt: ## Go kaynaklarını biçimlendir
	gofmt -w internal cmd

# ----------------------------------------------------------------------------
# Database
# ----------------------------------------------------------------------------
.PHONY: db-reset
db-reset: ## Şemayı sıfırla ve yeniden tohumla (DİKKAT: tüm veriyi siler)
	GOPENID_DB_RESET=true GOPENID_DEV_SEED=true go run $(PKG)

# ----------------------------------------------------------------------------
# Cleanup
# ----------------------------------------------------------------------------
.PHONY: clean
clean: ## Derleme çıktılarını temizle
	rm -rf bin $(DIST_DIR)
	@echo "Temizlendi: bin/, $(DIST_DIR)"
