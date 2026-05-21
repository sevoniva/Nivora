#!/bin/sh
set -e
# Helm chart passes: server --config /etc/nivora/server.yaml
# The binary expects: nivora-server <config-path> (no --config flag)
CMD="$1"
shift
# Strip --config flag if present, use the next arg as config path.
CONFIG_PATH=""
while [ $# -gt 0 ]; do
  case "$1" in
    --config) CONFIG_PATH="$2"; shift 2 ;;
    *) break ;;
  esac
done

case "$CMD" in
  server) exec /usr/local/bin/nivora-server ${CONFIG_PATH:+"$CONFIG_PATH"} ;;
  worker) exec /usr/local/bin/nivora-worker ${CONFIG_PATH:+"$CONFIG_PATH"} ;;
  runner) exec /usr/local/bin/nivora-runner ${CONFIG_PATH:+"$CONFIG_PATH"} ;;
  *) exec "$CMD" "$@" ;;
esac
