#!/usr/bin/env sh
set -eu

required_files="
VERSION
CHANGELOG.md
.github/release-template.md
docs/ALPHA_CAPABILITY_MATRIX.md
docs/demo/alpha-demo.md
docs/releases/v0.1.0-alpha.1-checklist.md
"

for file in $required_files; do
  if [ ! -s "$file" ]; then
    echo "missing required alpha release file: $file" >&2
    exit 1
  fi
done

version="$(tr -d '[:space:]' < VERSION)"
if [ "$version" != "0.1.0-alpha.1" ]; then
  echo "VERSION must be 0.1.0-alpha.1, got $version" >&2
  exit 1
fi

if ! grep -q 'Version = "0.1.0-alpha.1"' internal/version/version.go; then
  echo "internal/version/version.go is not aligned with VERSION" >&2
  exit 1
fi

for file in README.md CHANGELOG.md docs/ALPHA_CAPABILITY_MATRIX.md docs/demo/alpha-demo.md docs/releases/v0.1.0-alpha.1-checklist.md; do
  if ! grep -qi 'not production-ready\|not production ready\|not a production' "$file"; then
    echo "$file must state the alpha is not production-ready" >&2
    exit 1
  fi
done

echo "Alpha release docs are present and version-aligned."
