#!/usr/bin/env bash
# Fails when a committed client no longer matches what the spec generates.
#
# Usage: check-generated-sdk.sh [language]
#   language defaults to typescript-fetch; `go` checks sdk/go, `python`
#   sdk/python, `java` sdk/java.
#
# This is the gate that keeps `SignalingServer/api/openapi.yaml` and `sdk/ts`
# from drifting: a spec edit without a regenerated client fails here, and so does
# a hand-edit of the generated tree. CI runs it; so can anyone locally.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
lang="${1:-typescript-fetch}"

case "$lang" in
  typescript-fetch)
    committed="$root/sdk/ts"
    ;;
  go)
    committed="$root/sdk/go"
    ;;
  python)
    committed="$root/sdk/python"
    ;;
  java)
    committed="$root/sdk/java"
    ;;
  *)
    echo "unknown language '$lang': expected typescript-fetch, go, python or java" >&2
    exit 2
    ;;
esac

fresh="$(mktemp -d)"
trap 'rm -rf "$fresh"' EXIT

echo "generating a fresh $lang client to compare against ${committed#"$root"/}..."
"$root/scripts/sdk/generate-client.sh" "$fresh/out" "$lang" >/dev/null

# Excluded from the comparison, and why:
#   FILES             the generator's own manifest of what *that run* wrote. It
#                     lists `.openapi-generator-ignore` only when the file did
#                     not already exist, so regenerating in place differs from a
#                     fresh generation for reasons that say nothing about the
#                     client.
#   node_modules      local npm state; the committed client carries no deps.
#   dist              local build output.
#   package-lock.json produced by `npm install` here, not by the generator.
#   __pycache__ / *.egg-info / build / .venv
#                     local Python state, for the same reason as node_modules.
#                     (`build` and the egg-info tree are what `pip install .`
#                     leaves behind in the source directory.)
#   target            what `mvn package` leaves behind in the Java client.
#   examples          the Go client's hand-written usage example, which the CI
#                     smoke job runs against a live server. It is the only file
#                     under sdk/ that is not generated, and excluding it here is
#                     what keeps that distinction honest.
if ! diff -r -x FILES -x node_modules -x dist -x package-lock.json \
     -x __pycache__ -x '*.egg-info' -x build -x .venv -x target -x examples \
     "$committed" "$fresh/out"; then
  echo
  echo "${committed#"$root"/} is stale: run scripts/sdk/generate-client.sh and commit the result." >&2
  exit 1
fi

echo "${committed#"$root"/} matches a fresh $lang generation from SignalingServer/api/openapi.yaml"
