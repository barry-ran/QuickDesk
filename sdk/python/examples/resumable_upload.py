#!/usr/bin/env python3
"""Drive the chunked recording upload through the committed Python client.

The Go client has `sdk/go/examples/resumableupload`; this is the same contract
through the generated Python client: stage two chunks with PATCH, probe the
offset with `Upload-Probe: 1` without staging anything, resync after a 409 that
carries the authoritative offset in the response header, let the declared total
finish the upload, read the bytes back through a playback ticket, and delete the
row.

Usage:

    QD_IDENTIFIER=user QD_PASSWORD=secret QD_SMOKE_DEVICE=123456789 \
        python examples/resumable_upload.py

No online host is required: a recording row is an index entry, not a session.
"""

import os
import sys
import urllib.parse

import quickdesk_signaling
from quickdesk_signaling.models.auth_session_request import AuthSessionRequest
from quickdesk_signaling.models.session_recording_input import SessionRecordingInput

TOTAL = 1 << 20  # 1 MiB


def env_or(key: str, fallback: str) -> str:
    return os.environ.get(key) or fallback


def header(headers, name: str):
    """Read a response header without depending on its case."""
    if headers is None:
        return None
    for key in (name, name.lower(), name.upper()):
        value = headers.get(key)
        if value is not None:
            return value
    return None


def parse_offset(headers) -> int:
    raw = header(headers, "Upload-Offset")
    if raw is None:
        raise RuntimeError("response carried no Upload-Offset header")
    return int(raw)


def append(recordings, recording_id: int, offset: int, data, probe: bool = False) -> int:
    """PATCH one chunk (or probe) and return the offset the server reports."""
    response = recordings.append_recording_artifact_with_http_info(
        ext=".mp4",
        upload_offset=offset,
        recording_id=recording_id,
        upload_length=str(TOTAL),
        upload_probe="1" if probe else None,
        body=data,
    )
    if response.status_code != 202:
        raise RuntimeError(f"append at offset {offset} answered {response.status_code}, want 202")
    return parse_offset(response.headers)


def main() -> int:
    server = env_or("QD_SERVER", "http://127.0.0.1:8000")
    api_key = env_or("QD_API_KEY", "qd-local-dev-key")
    identifier = os.environ.get("QD_IDENTIFIER")
    password = os.environ.get("QD_PASSWORD")
    device_id = os.environ.get("QD_SMOKE_DEVICE")
    if not identifier or not password or not device_id:
        print("QD_IDENTIFIER, QD_PASSWORD and QD_SMOKE_DEVICE are required", file=sys.stderr)
        return 1

    config = quickdesk_signaling.Configuration(
        host=server,
        api_key={"ApiKeyAuth": api_key},
    )

    with quickdesk_signaling.ApiClient(config) as client:
        session = quickdesk_signaling.AuthApi(client).create_session(
            AuthSessionRequest(identifier=identifier, password=password)
        )
        config.access_token = session.access_token
        print("login ok")

        recordings = quickdesk_signaling.RecordingsApi(client)
        created = recordings.create_session_recording(
            SessionRecordingInput(
                device_id=device_id,
                device_name="sdk-resumable-python",
                session_id=f"sdk-resumable-py-{os.getpid()}",
                status="recording",
            )
        )
        recording_id = created.id
        print(f"recording {recording_id} created for a {TOTAL}-byte upload")

        try:
            media = bytes((i * 31 + 7) & 0xFF for i in range(TOTAL))

            offset = append(recordings, recording_id, 0, media[:400000])
            if offset != 400000:
                print(f"first chunk left offset={offset}, want 400000", file=sys.stderr)
                return 1
            print(f"chunk 1 staged, server reports offset {offset}")

            # The probe tells the server to ignore the body and stage nothing.
            offset = append(recordings, recording_id, offset, b"\x00", probe=True)
            if offset != 400000:
                print(f"probe moved the offset to {offset}, want 400000", file=sys.stderr)
                return 1
            print(f"probe (Upload-Probe: 1) answered offset {offset} without staging")

            try:
                recordings.append_recording_artifact(
                    ext=".mp4",
                    upload_offset=12345,
                    recording_id=recording_id,
                    upload_length=str(TOTAL),
                    body=media[400000:800000],
                )
                print("a chunk with a wrong offset was accepted", file=sys.stderr)
                return 1
            except quickdesk_signaling.ApiException as exc:
                if exc.status != 409:
                    print(f"wrong offset answered {exc.status}, want 409", file=sys.stderr)
                    return 1
                conflict_offset = parse_offset(exc.headers)
                if conflict_offset != 400000:
                    print(f"409 reported offset={conflict_offset}, want 400000", file=sys.stderr)
                    return 1
                print(
                    f"wrong offset refused with 409 and the authoritative offset {conflict_offset}"
                )

            offset = append(recordings, recording_id, offset, media[400000:800000])
            if offset != 800000:
                print(f"second chunk left offset={offset}, want 800000", file=sys.stderr)
                return 1
            print(f"chunk 2 staged, server reports offset {offset}")

            # The last byte lands at the declared total, so the server finalizes
            # by itself; the 200 answer is the completed row.
            row = recordings.append_recording_artifact(
                ext=".mp4",
                upload_offset=offset,
                recording_id=recording_id,
                upload_length=str(TOTAL),
                body=media[800000:],
            )
            if row.size_bytes != TOTAL:
                print(f"finalized size_bytes={row.size_bytes}, want {TOTAL}", file=sys.stderr)
                return 1
            print(f"final chunk finalized the upload (status={row.status}, size={row.size_bytes})")

            ticket = recordings.create_recording_artifact_ticket(recording_id)
            token = urllib.parse.parse_qs(urllib.parse.urlparse(ticket.url).query).get("t", [""])[0]
            downloaded = recordings.download_recording_artifact_by_ticket(
                t=token, recording_id=recording_id
            )
            if bytes(downloaded) != media:
                print(
                    f"downloaded {len(downloaded)} bytes, want {len(media)} with identical content",
                    file=sys.stderr,
                )
                return 1
            print(f"artifact read back through the ticket ({len(downloaded)} bytes, identical)")
            print("PASS")
        finally:
            recordings.delete_session_recording(recording_id)
            print(f"recording {recording_id} deleted")
    return 0


if __name__ == "__main__":
    sys.exit(main())
