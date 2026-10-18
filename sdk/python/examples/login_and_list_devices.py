#!/usr/bin/env python3
"""Drive the committed Python client against a running signaling server.

This is the one hand-written file in this module - the freshness check excludes
this directory for exactly that reason - and it earns its place by being run in
the CI smoke job against a real server, where it signs in with the seeded
account and reads back the device list. A client that imports but cannot talk
to the server would otherwise pass every gate.

Usage:

    QD_IDENTIFIER=user QD_PASSWORD=secret python examples/login_and_list_devices.py
"""

import os
import sys

import quickdesk_signaling
from quickdesk_signaling.models.auth_session_request import AuthSessionRequest


def env_or(key: str, fallback: str) -> str:
    return os.environ.get(key) or fallback


def main() -> int:
    server = env_or("QD_SERVER", "http://127.0.0.1:8000")
    api_key = env_or("QD_API_KEY", "qd-local-dev-key")
    identifier = os.environ.get("QD_IDENTIFIER")
    password = os.environ.get("QD_PASSWORD")
    if not identifier or not password:
        print("QD_IDENTIFIER and QD_PASSWORD are required", file=sys.stderr)
        return 1

    # A self-hosted deployment that sets ALLOWED_ORIGINS requires the API key on
    # every request; sending it unconditionally is harmless on one that does not.
    config = quickdesk_signaling.Configuration(
        host=server,
        api_key={"ApiKeyAuth": api_key},
    )

    with quickdesk_signaling.ApiClient(config) as client:
        session = quickdesk_signaling.AuthApi(client).create_session(
            AuthSessionRequest(identifier=identifier, password=password)
        )
        token = session.access_token
        if not token:
            print("login returned no access token", file=sys.stderr)
            return 1
        print(f"login ok (token length {len(token)})")

        # The access token is applied by the client on every later call, the
        # same way the Go example stores it in the context.
        config.access_token = token
        devices = quickdesk_signaling.DevicesApi(client).list_my_devices()
        items = devices.items
        print(f"devices: {len(items)}")
        for device in items:
            print(
                f"  {device.device_id} {device.device_name!r} "
                f"online={device.online}"
            )
    return 0


if __name__ == "__main__":
    sys.exit(main())
