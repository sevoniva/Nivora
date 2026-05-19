#!/usr/bin/env sh
set -eu

required_files="
VERSION
CHANGELOG.md
.github/release-template.md
"

for file in $required_files; do
  if [ ! -s "$file" ]; then
    echo "missing required alpha release file: $file" >&2
    exit 1
  fi
done

version="$(tr -d '[:space:]' < VERSION)"

if ! grep -q "Version = \"$version\"" internal/version/version.go; then
  echo "internal/version/version.go is not aligned with VERSION" >&2
  exit 1
fi

case "$version" in
  0.1.0-alpha.1)
    required_release_files="
docs/ALPHA_CAPABILITY_MATRIX.md
docs/demo/alpha-demo.md
docs/releases/v0.1.0-alpha.1-checklist.md
"
    for file in $required_release_files; do
      if [ ! -s "$file" ]; then
        echo "missing required alpha release file: $file" >&2
        exit 1
      fi
    done
    for file in README.md CHANGELOG.md docs/ALPHA_CAPABILITY_MATRIX.md docs/demo/alpha-demo.md docs/releases/v0.1.0-alpha.1-checklist.md; do
      if ! grep -qi 'not production-ready\|not production ready\|not a production' "$file"; then
        echo "$file must state the alpha is not production-ready" >&2
        exit 1
      fi
    done
    ;;
  0.9.0-beta-candidate)
    required_release_files="
docs/BETA_CAPABILITY_MATRIX.md
docs/releases/v0.5.0-beta-checklist.md
docs/releases/v0.5.0-beta-release-notes-draft.md
docs/status/CAPABILITY_STATUS.md
docs/status/IMPLEMENTATION_AUDIT.md
docs/status/STORE_PERSISTENCE_MATRIX.md
"
    for file in $required_release_files; do
      if [ ! -s "$file" ]; then
        echo "missing required beta release file: $file" >&2
        exit 1
      fi
    done
    for file in README.md CHANGELOG.md docs/status/CAPABILITY_STATUS.md; do
      if ! grep -qi 'not production-ready\|not production ready\|beta-candidate\|hardened beta-candidate' "$file"; then
        echo "$file must state the beta-candidate is not production-ready" >&2
        exit 1
      fi
    done
    ;;
  1.0.0)
    required_release_files="
docs/releases/v1.0.0-ga-capability-matrix.md
docs/releases/v1.0.0-ga-checklist.md
docs/releases/v1.0.0-release-notes.md
docs/releases/release-playbook.md
docs/security/threat-model.md
docs/security/security-review-checklist.md
"
    for file in $required_release_files; do
      if [ ! -s "$file" ]; then
        echo "missing required GA release file: $file" >&2
        exit 1
      fi
    done
    for file in README.md CHANGELOG.md docs/releases/v1.0.0-ga-capability-matrix.md docs/releases/v1.0.0-ga-checklist.md docs/releases/v1.0.0-release-notes.md; do
      if ! grep -qi 'experimental\|known limitations\|beta\|not production-ready' "$file"; then
        echo "$file must describe future GA limitations or capability labels" >&2
        exit 1
      fi
    done
    ;;
  *)
    echo "unsupported release VERSION for release-doc verification: $version" >&2
    exit 1
    ;;
esac

echo "Release docs are present and version-aligned for $version."
