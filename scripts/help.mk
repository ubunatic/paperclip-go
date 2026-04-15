.PHONY: ⚙️  # make all non-file targets phony

# format:      cyan   $trg  rst     $desc
_target_fmt = "   \033[36m%-10s\033[0m  %s\n"

help: ⚙️  ## Show this help message
	@printf "Usage: \033[36mmake [target]\033[0m \033[33m(default: $(.DEFAULT_GOAL))\033[0m\n"
	@printf "Targets:\n"
	@awk -v FS=':|#+' -v fmt=$(_target_fmt) '/[a-z0-9_-]+:.*##/ { printf fmt, $$1, $$3 }' $(MAKEFILE_LIST)