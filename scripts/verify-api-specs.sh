#!/usr/bin/env sh
set -eu

for file in api/openapi/openapi.yaml api/asyncapi/asyncapi.yaml; do
  if [ ! -s "$file" ]; then
    echo "missing API spec: $file" >&2
    exit 1
  fi
done

if command -v ruby >/dev/null 2>&1; then
  ruby -e 'require "yaml"; ARGV.each { |f| YAML.load_file(f) || abort("empty YAML: #{f}") }' \
    api/openapi/openapi.yaml \
    api/asyncapi/asyncapi.yaml
  echo "API specs parsed as YAML."
else
  echo "ruby not found; skipping YAML parse for API specs"
fi
