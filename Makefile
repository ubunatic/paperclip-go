.PHONY: ⚙️  # make all non-file targets phony

.DEFAULT_GOAL := all

SHELL   := bash
BINARY  := paperclip-go
CMD     := ./cmd/paperclip-go
OUT     := ./bin/$(BINARY)

include scripts/help.mk

all: ⚙️ build test lint  ## Build, test, and lint the project

build: ⚙️  ## Build the paperclip-go binary
	go build -ldflags "-X main.Version=$$(git describe --tags --always --dirty 2>/dev/null || echo dev)" -o "$(OUT)" "$(CMD)"

run: ⚙️  ## Run the paperclip-go server
	go run "$(CMD)" serve

test: ⚙️  ## Run tests
	go test ./...

lint: ⚙️  ## Run linters
	go vet ./...

clean: ⚙️  ## Clean up build artifacts
	rm -rf bin/

WT := .build/worktree-$(shell date +%s)
wt: ⚙️ clean-wt  ## Create a worktree for the upstream branch
	git worktree add "${WT}"
	@echo "🛠️  Worktree created at ${WT}."

clean-wt: ⚙️  ## Remove the worktree
	git worktree prune -v
	git worktree remove -f "${WT}" 2>/dev/null || true
	@echo "🧹 Worktree ${WT} cleaned up."

sync-upstream: ⚙️  ## Sync upstream repository to upstream branch
	@echo "🔄 Syncing upstream repository to upstream branch..."
	git remote show | grep -q upstream || \
	    git remote add upstream https://github.com/paperclipai/paperclip.git
	git fetch upstream master
	$(MAKE) wt WT=${WT}
	git -C "${WT}" reset --hard upstream/master -q
	git -C "${WT}" push origin upstream --force -q
	$(MAKE) clean-wt WT=${WT}
	@echo "✅ Upstream repository synced to upstream branch."

OURS=README.md cmd internal Makefile go.mod go.sum
merge-upstream: ⚙️  ## Auto-merge upstream changes while preserving Go-specific core files
	@echo "🔄 Merging upstream changes (preserving Go-specific core files)..."
	git fetch origin upstream  # fetch upstream changes from our fork
	git merge upstream --no-commit --no-ff || echo "⚠️ Merge conflicts detected (expected)"
	git checkout upstream -- README.md
	@echo "↪️ Re-apply Go-specific README.md changes..."
	git mv -f README.md README.orig.md
	git rm -f .github/PULL_REQUEST_TEMPLATE.md 2>/dev/null || true
	@echo "⬇️ Preserving Go-specific core files..."
	git checkout HEAD -- ${OURS}
	git commit -m "Sync upstream (preserving Go-specific core)"
	@echo "✅ Upstream changes merged (Go-specific core preserved)."
