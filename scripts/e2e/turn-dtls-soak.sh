#!/usr/bin/env bash
# Hour-scale DTLS soak for coturn.
#
# Why this exists: turn-capacity.mjs covers UDP and TLS soaks with full metrics,
# but Node has no DTLS stack, so the DTLS leg was only ever a short smoke
# (turn-dtls.sh, ~20 messages). A short smoke cannot see the two things that
# only show up over time: whether the allocation survives past its 600s
# lifetime, and whether anything degrades after hundreds of refreshes.
#
# This drives coturn's own turnutils_uclient over DTLS for a configurable
# duration, asserts every message was relayed, and counts the DTLS allocation
# refreshes coturn logged during the run as evidence that the refresh path -
# not just the initial allocate - was exercised.
#
# Prerequisites: coturn installed and running with `dtls` enabled (coturn 4.18
# needs that option explicitly), and QD_TURN_SECRET matching its
# static-auth-secret.
#
# Usage:
#   QD_TURN_SECRET=qd-turn-secret-dev scripts/e2e/turn-dtls-soak.sh
#
# Env:
#   QD_TURN_HOST         default 127.0.0.1
#   QD_TURN_TLS_PORT     default 5349   (TLS and DTLS share this port)
#   QD_TURN_PEER_PORT    default 3480   (turnutils_peer listens here)
#   QD_TURN_SOAK_SECONDS default 3600
#   QD_TURN_INTERVAL_MS  default 1000   (per-message interval)
#   QD_TURN_MSG_LEN      default 120
#   QD_TURN_LOG          default /tmp/qd-turn.log (coturn log to count refreshes in)
#   TURNUTILS_DIR        default /opt/homebrew/opt/coturn/bin
#
# Exit codes: 0 = every message relayed and the run lasted the requested time,
# 1 = a leg failed or the run finished early.

set -uo pipefail

HOST="${QD_TURN_HOST:-127.0.0.1}"
PORT="${QD_TURN_TLS_PORT:-5349}"
PEER_PORT="${QD_TURN_PEER_PORT:-3480}"
SOAK_SECONDS="${QD_TURN_SOAK_SECONDS:-3600}"
INTERVAL_MS="${QD_TURN_INTERVAL_MS:-1000}"
MSG_LEN="${QD_TURN_MSG_LEN:-120}"
TURN_LOG="${QD_TURN_LOG:-/tmp/qd-turn.log}"
TURNUTILS_DIR="${TURNUTILS_DIR:-/opt/homebrew/opt/coturn/bin}"
SECRET="${QD_TURN_SECRET:-qd-turn-secret-dev}"

UCLIENT="$TURNUTILS_DIR/turnutils_uclient"
PEER="$TURNUTILS_DIR/turnutils_peer"

if [[ ! -x "$UCLIENT" ]]; then
  echo "turnutils_uclient not found at $UCLIENT (set TURNUTILS_DIR)" >&2
  exit 1
fi

# uclient counts a fixed number of messages; the interval picks how long that
# takes, so the soak duration is expressed as (messages x interval).
MESSAGES=$(( SOAK_SECONDS * 1000 / INTERVAL_MS ))
if (( MESSAGES < 1 )); then
  echo "QD_TURN_SOAK_SECONDS ($SOAK_SECONDS) x 1000 / QD_TURN_INTERVAL_MS ($INTERVAL_MS) rounds to zero messages" >&2
  exit 1
fi

peer_pid=""
cleanup() {
  if [[ -n "$peer_pid" ]] && kill -0 "$peer_pid" 2>/dev/null; then
    kill "$peer_pid" 2>/dev/null
  fi
}
trap cleanup EXIT

# The echo peer answers the relayed packets; without it every send counts as
# lost and the run is meaningless (this is exactly how the first attempt at
# this script looked before the peer was started).
if [[ -x "$PEER" ]]; then
  "$PEER" -p "$PEER_PORT" >/dev/null 2>&1 &
  peer_pid=$!
  sleep 1
else
  echo "turnutils_peer not found at $PEER; relayed packets have no echo peer" >&2
  exit 1
fi

# Remember how far into the log we are, so the refresh count covers this run
# only rather than every previous test on the machine.
log_start=0
if [[ -f "$TURN_LOG" ]]; then
  log_start="$(wc -c <"$TURN_LOG" | tr -d ' ')"
fi

echo "DTLS soak: ${SOAK_SECONDS}s (${MESSAGES} messages @ ${INTERVAL_MS}ms, ${MSG_LEN}B) to ${HOST}:${PORT}"
started="$(date +%s)"

out="$("$UCLIENT" -S -W "$SECRET" -p "$PORT" \
      -n "$MESSAGES" -z "$INTERVAL_MS" -l "$MSG_LEN" \
      -e "$HOST" -r "$PEER_PORT" "$HOST" 2>&1)"
rc=$?
elapsed=$(( $(date +%s) - started ))

sent="$(grep -oE 'tot_send_msgs=[0-9]+' <<<"$out" | tail -1 | cut -d= -f2)"
recv="$(grep -oE 'tot_recv_msgs=[0-9]+' <<<"$out" | tail -1 | cut -d= -f2)"
lost="$(grep -oiE 'total lost packets [0-9]+' <<<"$out" | tail -1 | awk '{print $4}')"

# Count the DTLS allocation refreshes coturn logged during this run. The
# cipher/method fields are what make this evidence about the DTLS leg rather
# than about any other transport that happened to be busy.
refreshes=0
if [[ -f "$TURN_LOG" ]]; then
  refreshes="$(tail -c +$(( log_start + 1 )) "$TURN_LOG" \
    | grep -c 'allocation refreshed.*method=DTLSv1' || true)"
fi

echo "DTLS soak result: rc=$rc elapsed=${elapsed}s sent=${sent:-0} recv=${recv:-0} lost=${lost:-?} dtls_refreshes=${refreshes}"

status=0
if [[ "$rc" -ne 0 ]]; then
  echo "FAIL: turnutils_uclient exited $rc" >&2
  status=1
fi
if [[ -z "${sent:-}" || "${sent:-0}" -eq 0 ]]; then
  echo "FAIL: no messages were sent" >&2
  status=1
fi
if [[ "${sent:-0}" -ne "${recv:-0}" ]]; then
  echo "FAIL: sent $sent but received $recv (100% loss means the peer or the relay leg is wrong, not a flaky run)" >&2
  status=1
fi
if [[ "${lost:-1}" -ne 0 ]]; then
  echo "FAIL: coturn reported ${lost} lost packets" >&2
  status=1
fi
# A silent early exit is the failure mode a duration assertion catches: the
# counters can look perfect on a run that stopped after ten seconds.
if (( elapsed < SOAK_SECONDS )); then
  echo "FAIL: run lasted ${elapsed}s but ${SOAK_SECONDS}s was requested" >&2
  status=1
fi
# A soak longer than the 600s allocation lifetime that never refreshed would
# have died, so it either refreshed or the counters would already have failed;
# require the evidence explicitly anyway.
if (( SOAK_SECONDS > 600 )) && (( refreshes == 0 )); then
  echo "FAIL: no DTLS allocation refresh was logged during a ${SOAK_SECONDS}s run" >&2
  status=1
fi

if [[ "$status" -eq 0 ]]; then
  echo "PASS: DTLS relayed every packet for ${elapsed}s across ${refreshes} allocation refreshes"
else
  echo "FAILED" >&2
  echo "$out" | tail -5 >&2
fi
exit "$status"
