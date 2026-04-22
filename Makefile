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

sync-upstream: WB=upstream
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

merge-upstream: ⚙️  ## Auto-merge upstream after sync-upstream
	git fetch origin upstream
	@echo "🛡️ Checking for uncommitted changes before merging..."
	@git diff --quiet || { echo "❌ Uncommitted changes!. Please commit before merging."; exit 1; }
	@echo "💾 Backed up README.md tmp/README.md to preserve changes."
	@# git checkout -f upstream -- README.md .github/PULL_REQUEST_TEMPLATE.md
	@echo "⛓️ Incompatible files are in upstream state."
	@echo "☑️  Ready sync the rest without conflicts."
	git merge upstream -m "Merge upstream into master" --no-ff --no-commit
	@echo "✍️ Renaming incompatible files after merge..."
	mv README.md README.orig.md
	mv .github/PULL_REQUEST_TEMPLATE.md .github/PULL_REQUEST_TEMPLATE.disabled.md
	git checkout --ours README.md .github/PULL_REQUEST_TEMPLATE.md

bla:
	@echo "☑️  Renamed incompatible files to avoid conflicts."
	@# git add .
	@echo "💾 Restored README.md from backup."
	@# git commit -m "Merge upstream changes and resolve conflicts" -q
	@echo "✅ Upstream merged into current branch"
