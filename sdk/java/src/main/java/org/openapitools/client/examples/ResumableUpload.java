package org.openapitools.client.examples;

// Drive the chunked recording upload through the committed Java client.
// The Go client has sdk/go/examples/resumableupload; this is the same contract
// through the generated Java client: two PATCH chunks with a declared total, an
// Upload-Probe offset probe, a 409 resync from the response header, server-side
// finalisation, a byte-identical ticket read back, and a delete.
//
// Usage (from sdk/java, after `mvn -q -DskipTests compile`):
//
//   QD_IDENTIFIER=user QD_PASSWORD=secret QD_SMOKE_DEVICE=123456789 \
//     java -cp "target/classes:$(cat /tmp/qd-java-cp.txt)" \
//     org.openapitools.client.examples.ResumableUpload
//
// No online host is required: a recording row is an index entry, not a session.

import org.openapitools.client.ApiClient;
import org.openapitools.client.ApiException;
import org.openapitools.client.ApiResponse;
import org.openapitools.client.api.AuthApi;
import org.openapitools.client.api.RecordingsApi;
import org.openapitools.client.model.AuthSessionRequest;
import org.openapitools.client.model.AuthSessionResponse;
import org.openapitools.client.model.PlaybackTicket;
import org.openapitools.client.model.SessionRecording;
import org.openapitools.client.model.SessionRecordingInput;

import java.io.File;
import java.io.IOException;
import java.net.URI;
import java.nio.file.Files;
import java.util.Arrays;
import java.util.List;
import java.util.Map;

public final class ResumableUpload {

    private static final int TOTAL = 1 << 20; // 1 MiB

    private ResumableUpload() {
    }

    public static void main(String[] args) throws Exception {
        String server = envOr("QD_SERVER", "http://127.0.0.1:8000");
        String apiKey = envOr("QD_API_KEY", "qd-local-dev-key");
        String identifier = System.getenv("QD_IDENTIFIER");
        String password = System.getenv("QD_PASSWORD");
        String deviceId = System.getenv("QD_SMOKE_DEVICE");
        if (identifier == null || password == null || deviceId == null) {
            System.err.println("QD_IDENTIFIER, QD_PASSWORD and QD_SMOKE_DEVICE are required");
            System.exit(1);
        }

        ApiClient client = new ApiClient();
        client.setBasePath(server);
        client.setApiKey(apiKey);

        AuthSessionResponse session = new AuthApi(client).createSession(
                new AuthSessionRequest().identifier(identifier).password(password));
        client.setBearerToken(session.getAccessToken());
        System.out.println("login ok");

        RecordingsApi recordings = new RecordingsApi(client);
        SessionRecording created = recordings.createSessionRecording(
                new SessionRecordingInput()
                        .deviceId(deviceId)
                        .deviceName("sdk-resumable-java")
                        .sessionId("sdk-resumable-java-" + ProcessHandle.current().pid())
                        .status(SessionRecordingInput.StatusEnum.RECORDING));
        long recordingId = created.getId();
        System.out.println("recording " + recordingId + " created for a " + TOTAL + "-byte upload");

        byte[] media = new byte[TOTAL];
        for (int i = 0; i < TOTAL; i++) {
            media[i] = (byte) ((i * 31 + 7) & 0xff);
        }

        try {
            long offset = append(recordings, recordingId, 0,
                    Arrays.copyOfRange(media, 0, 400000), false);
            require(offset == 400000, "first chunk left offset=" + offset + ", want 400000");
            System.out.println("chunk 1 staged, server reports offset " + offset);

            // The probe tells the server to ignore the body and stage nothing.
            offset = append(recordings, recordingId, offset, new byte[] {0}, true);
            require(offset == 400000, "probe moved the offset to " + offset + ", want 400000");
            System.out.println("probe (Upload-Probe: 1) answered offset " + offset + " without staging");

            try {
                recordings.appendRecordingArtifact(".mp4", 12345L, recordingId,
                        String.valueOf(TOTAL), null, null,
                        stage(Arrays.copyOfRange(media, 400000, 800000)));
                throw new IllegalStateException("a chunk with a wrong offset was accepted");
            } catch (ApiException conflict) {
                require(conflict.getCode() == 409,
                        "wrong offset answered " + conflict.getCode() + ", want 409");
                long conflictOffset = offsetHeader(conflict.getResponseHeaders());
                require(conflictOffset == 400000,
                        "409 reported offset=" + conflictOffset + ", want 400000");
                System.out.println("wrong offset refused with 409 and the authoritative offset "
                        + conflictOffset);
            }

            offset = append(recordings, recordingId, offset,
                    Arrays.copyOfRange(media, 400000, 800000), false);
            require(offset == 800000, "second chunk left offset=" + offset + ", want 800000");
            System.out.println("chunk 2 staged, server reports offset " + offset);

            // The last byte lands at the declared total, so the server finalizes
            // by itself; the 200 answer is the completed row.
            SessionRecording row = recordings.appendRecordingArtifact(".mp4", offset, recordingId,
                    String.valueOf(TOTAL), null, null,
                    stage(Arrays.copyOfRange(media, 800000, TOTAL)));
            require(row.getSizeBytes() == TOTAL,
                    "finalized size_bytes=" + row.getSizeBytes() + ", want " + TOTAL);
            System.out.println("final chunk finalized the upload (status=" + row.getStatus()
                    + ", size=" + row.getSizeBytes() + ")");

            PlaybackTicket ticket = recordings.createRecordingArtifactTicket(recordingId);
            String token = ticketToken(ticket.getUrl());
            File downloaded = recordings.downloadRecordingArtifactByTicket(token, recordingId);
            try {
                byte[] got = Files.readAllBytes(downloaded.toPath());
                require(Arrays.equals(got, media),
                        "downloaded " + got.length + " bytes, want " + media.length
                                + " with identical content");
                System.out.println("artifact read back through the ticket (" + got.length
                        + " bytes, identical)");
            } finally {
                Files.deleteIfExists(downloaded.toPath());
            }
            System.out.println("PASS");
        } finally {
            recordings.deleteSessionRecording(recordingId);
            System.out.println("recording " + recordingId + " deleted");
        }
    }

    /** PATCH one chunk (or probe) and return the offset the server reports. */
    private static long append(RecordingsApi recordings, long recordingId, long offset,
            byte[] data, boolean probe) throws ApiException, IOException {
        ApiResponse<SessionRecording> response = recordings.appendRecordingArtifactWithHttpInfo(
                ".mp4", offset, recordingId, String.valueOf(TOTAL), null,
                probe ? "1" : null, stage(data));
        require(response.getStatusCode() == 202,
                "append at offset " + offset + " answered " + response.getStatusCode() + ", want 202");
        return offsetHeader(response.getHeaders());
    }

    /** The generated signature takes a File, so chunks are staged on disk. */
    private static File stage(byte[] data) throws IOException {
        File file = File.createTempFile("qd-resumable-", ".part");
        file.deleteOnExit();
        Files.write(file.toPath(), data);
        return file;
    }

    private static long offsetHeader(Map<String, List<String>> headers) {
        if (headers != null) {
            for (Map.Entry<String, List<String>> entry : headers.entrySet()) {
                if (entry.getKey().equalsIgnoreCase("Upload-Offset") && !entry.getValue().isEmpty()) {
                    return Long.parseLong(entry.getValue().get(0));
                }
            }
        }
        throw new IllegalStateException("response carried no Upload-Offset header");
    }

    private static String ticketToken(String url) {
        String query = URI.create(url).getQuery();
        require(query != null, "ticket url " + url + " carries no query string");
        for (String pair : query.split("&")) {
            int eq = pair.indexOf('=');
            if (eq > 0 && pair.substring(0, eq).equals("t")) {
                return pair.substring(eq + 1);
            }
        }
        throw new IllegalStateException("ticket url " + url + " carries no t= parameter");
    }

    private static void require(boolean ok, String message) {
        if (!ok) {
            throw new IllegalStateException(message);
        }
    }

    private static String envOr(String key, String fallback) {
        String value = System.getenv(key);
        return value == null || value.isEmpty() ? fallback : value;
    }
}
