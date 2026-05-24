#!/bin/bash
set -e

BOLD='\033[1m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

HOOKS_DIR="$(git rev-parse --git-dir)/hooks"
SCRIPTS_DIR="$(dirname "$0")"

HOOKS=("pre-commit")

echo -e "${BOLD}Installing git hooks...${NC}"

for hook in "${HOOKS[@]}"; do
    SRC="$SCRIPTS_DIR/$hook"
    DST="$HOOKS_DIR/$hook"

    if [ ! -f "$SRC" ]; then
        echo -e "${YELLOW}⚠  Skipping $hook — source not found: $SRC${NC}"
        continue
    fi

    # Backup existing hook if it's not already our symlink
    if [ -f "$DST" ] && [ ! -L "$DST" ]; then
        mv "$DST" "${DST}.bak"
        echo "  Backed up existing $hook → ${hook}.bak"
    fi

    ln -sf "../../scripts/$hook" "$DST"
    chmod +x "$SRC"
    echo -e "  ${GREEN}✓${NC} $hook → .git/hooks/$hook"
done

echo -e "${GREEN}${BOLD}Done.${NC} Hooks installed."
echo ""
echo "Tip: Make sure staticcheck is installed:"
echo "  go install honnef.co/go/tools/cmd/staticcheck@latest"
