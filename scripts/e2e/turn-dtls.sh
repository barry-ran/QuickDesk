#!/usr/bin/env bash
# DTLS / TLS smoke check for coturn, using coturn's own client.
#
# Why not turn-capacity.mjs? Node has no DTLS stack (`node:tls` is TLS-over-TCP
# only, and `node:dgram` is plain UDP), so TURN over DTLS cannot be exercised
# from the JS harness. turnutils_uclient speaks both: `-S` means "TLS over TCP,
# DTLS over UDP", and the transport is picked by whether `-t` is present.
#
# Prerequisites:
#   - coturn installed (turnserver, turnutils_uclient, turnutils_peer)
#   - a coturn running in `use-auth-secret` mode with TLS/DTLS enabled; note
#     that coturn 4.18 needs the explicit `dtls` option, otherwise it logs
#     "DTLS listeners are not started; use --dtls to start them" and the secure
#     port stays TCP-only
#
# Usage:
#   QD_TURN_SECRET=qd-turn-secret-dev scripts/e2e/turn-dtls.sh
#
# Env:
#   QD_TURN_HOST      default 127.0.0.1
#   QD_TURN_TLS_PORT  default 5349        (TLS and DTLS share this port)
#   QD_TURN_PEER_PORT default 3480        (turnutils_peer listens here)
#   QD_TURN_MESSAGES  default 20
#   QD_TURN_SKIP_TLS  set to skip the TLS-over-TCP leg
#   TURNUTILS_DIR     default /opt/homebrew/opt/coturn/bin
#
# Exit codes: 0 = all legs relayed every packet, 1 = a leg failed.

set -uo pipefail

HOST="${QD_TURN_HOST:-127.0.0.1}"
PORT="${QD_TURN_TLS_PORT:-5349}"
PEER_PORT="${QD_TURN_PEER_PORT:-3480}"
MESSAGES="${QD_TURN_MESSAGES:-20}"
TURNUTILS_DIR="${TURNUTILS_DIR:-/opt/homebrew/opt/coturn/bin}"
SECRET="${QD_TURN_SECRET:-qd-turn-secret-dev}"

UCLIENT="$TURNUTILS_DIR/turnutils_uclient"
PEER="$TURNUTILS_DIR/turnutils_peer"

if [[ ! -x "$UCLIENT" ]]; then
  echo "turnutils_uclient not found at $UCLIENT (set TURNUTILS_DIR)" >&2
  exit 1
fi

peer_pid=""
cleanup() {
  if [[ -n "$peer_pid" ]] && kill -0 "$peer_pid" 2>/dev/null; then
    kill "$peer_pid" 2>/dev/null
  fi
}
trap cleanup EXIT

# The echo peer only needs to answer relayed packets; it is started once and
# shared by both legs.
if [[ -x "$PEER" ]]; then
  "$PEER" -p "$PEER_PORT" >/dev/null 2>&1 &
  peer_pid=$!
  sleep 1
else
  echo "warning: $PEER not found; relayed packets have no echo peer" >&2
fi

run_leg() {
  local label="$1"; shift
  local out rc sent recv lost
  out="$("$UCLIENT" "$@" -W "$SECRET" -p "$PORT" -n "$MESSAGES" -l 120 \
        -e "$HOST" -r "$PEER_PORT" "$HOST" 2>&1)"
  rc=$?
  sent="$(grep -oE 'tot_send_msgs=[0-9]+' <<<"$out" | tail -1 | cut -d= -f2)"
  recv="$(grep -oE 'tot_recv_msgs=[0-9]+' <<<"$out" | tail -1 | cut -d= -f2)"
  lost="$(grep -oiE 'total lost packets [0-9]+' <<<"$out" | tail -1 | awk '{print $4}')"
  echo "$label: rc=$rc sent=${sent:-0} recv=${recv:-0} lost=${lost:-?}"
  if [[ "$rc" -ne 0 || -z "${sent:-}" || "${sent:-0}" -eq 0 || "${sent:-0}" -ne "${recv:-0}" || "${lost:-0}" -ne 0 ]]; then
    echo "FAIL: $label did not relay every packet" >&2
    echo "$out" | tail -5 >&2
    return 1
  fi
  return 0
}

status=0
# No -t, so -S negotiates DTLS over UDP.
run_leg "DTLS (secure UDP)" -S || status=1
if [[ -z "${QD_TURN_SKIP_TLS:-}" ]]; then
  # -t + -S negotiates TLS over TCP on the same port.
  run_leg "TLS  (secure TCP)" -t -S || status=1
fi

if [[ "$status" -eq 0 ]]; then
  echo "PASS: coturn relayed every packet over DTLS (and TLS)"
else
  echo "FAILED" >&2
fi
exit "$status"
