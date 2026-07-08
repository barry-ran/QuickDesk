import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';

import 'package:quickdesk_android/protocol/auth/spake2.dart';
import 'package:quickdesk_android/protocol/auth/spake2_authenticator.dart';
import 'package:quickdesk_android/protocol/proto/protobuf_messages.dart';
import 'package:quickdesk_android/protocol/signaling/jingle.dart';

void main() {
  group('SPAKE2', () {
    test('Alice/Bob key agreement + verification hash', () {
      const deviceId = '123456789';
      const accessCode = '424242';
      const clientJid = '123456789@quickdesk.local/chromoting_ftl_android_abc';
      const hostJid = '123456789@quickdesk.local/chromoting_ftl_quickdesk_host';

      final password = getSharedSecretHash(deviceId, deviceId + accessCode);

      final alice = Spake2Context(spake2RoleAlice, clientJid, hostJid);
      final bob = Spake2Context(spake2RoleBob, hostJid, clientJid);

      final aliceMsg = alice.generateMessage(password);
      final bobMsg = bob.generateMessage(password);

      final aliceKey = alice.processMessage(bobMsg);
      final bobKey = bob.processMessage(aliceMsg);

      expect(bytesToHex(aliceKey), bytesToHex(bobKey));
      expect(alice.verifyHash(bob.getOutgoingVerificationHash()), isTrue);
      expect(bob.verifyHash(alice.getOutgoingVerificationHash()), isTrue);
    });

    test('wrong access code fails verification', () {
      const deviceId = '123456789';
      const clientJid = 'c@x/1';
      const hostJid = 'h@x/2';

      final goodPw = getSharedSecretHash(deviceId, '${deviceId}424242');
      final badPw = getSharedSecretHash(deviceId, '${deviceId}000000');

      final alice = Spake2Context(spake2RoleAlice, clientJid, hostJid);
      final bob = Spake2Context(spake2RoleBob, hostJid, clientJid);

      final aliceMsg = alice.generateMessage(goodPw);
      final bobMsg = bob.generateMessage(badPw);
      alice.processMessage(bobMsg);
      bob.processMessage(aliceMsg);

      expect(alice.verifyHash(bob.getOutgoingVerificationHash()), isFalse);
    });

    test('client/host authenticator state machines complete handshake', () {
      const clientJid = 'c@quickdesk.local/chromoting_ftl_android_1';
      const hostJid = 'c@quickdesk.local/chromoting_ftl_quickdesk_host';
      final secret = getSharedSecretHash('c', 'c123456');

      final client = Spake2ClientAuthenticator(clientJid, hostJid, secret);
      final host = Spake2HostAuthenticator(hostJid, clientJid, secret);

      // client → host: supported-methods
      host.processMessage(client.getFirstNegotiationMessage());
      expect(host.state, AuthState.messageReady);

      // host → client: method + spake message
      client.processMessage(host.getNextMessage());
      expect(client.state, AuthState.messageReady);

      // client → host: spake message + verification hash（合并在同一条消息，
      // 与 JS 实现一致——client 处理完 host 的 spake 消息后 key 已就绪）
      final clientMsg = client.getNextMessage();
      expect(clientMsg.spakeMessage, isNotNull);
      expect(clientMsg.verificationHash, isNotNull);
      host.processMessage(clientMsg);
      // host 已验证 client 的 hash（内部 accepted），但自己的 hash 还没发出
      expect(host.state, AuthState.messageReady);

      // host → client: verification hash
      client.processMessage(host.getNextMessage());
      expect(client.state, AuthState.accepted);
      expect(host.state, AuthState.accepted);

      expect(client.authKey, isNotNull);
      expect(bytesToHex(client.authKey!), bytesToHex(host.authKey!));
    });
  });

  group('Protobuf', () {
    test('EventMessage mouse roundtrip', () {
      final encoded = encodeEventMessage(
        timestamp: 1234567890,
        mouseEvent: MouseEventMsg(
          x: 100,
          y: -5,
          button: MouseButton.left.value,
          buttonDown: true,
          wheelDeltaY: -120.5,
        ),
      );
      final decoded = decodeEventMessage(encoded);
      expect(decoded.timestamp, 1234567890);
      expect(decoded.mouseEvent!.x, 100);
      expect(decoded.mouseEvent!.y, -5);
      expect(decoded.mouseEvent!.button, 1);
      expect(decoded.mouseEvent!.buttonDown, isTrue);
      expect(decoded.mouseEvent!.wheelDeltaY, closeTo(-120.5, 0.001));
    });

    test('EventMessage key + text roundtrip', () {
      final encoded = encodeEventMessage(
        keyEvent: KeyEventMsg(pressed: true, usbKeycode: 0x070028),
        textEventText: '你好 QuickDesk',
      );
      final decoded = decodeEventMessage(encoded);
      expect(decoded.keyEvent!.pressed, isTrue);
      expect(decoded.keyEvent!.usbKeycode, 0x070028);
      expect(decoded.textEventText, '你好 QuickDesk');
    });

    test('ControlMessage roundtrip', () {
      final encoded = encodeControlMessage(
        clipboardEvent: ClipboardEventMsg(
          mimeType: 'text/plain; charset=UTF-8',
          data: Uint8List.fromList('hello'.codeUnits),
        ),
        capabilities: 'fileTransfer privacyScreen',
        audioControlEnable: true,
      );
      final decoded = decodeControlMessage(encoded);
      expect(decoded.clipboardEvent!.mimeType, 'text/plain; charset=UTF-8');
      expect(String.fromCharCodes(decoded.clipboardEvent!.data), 'hello');
      expect(decoded.capabilities!.capabilities, 'fileTransfer privacyScreen');
      expect(decoded.audioControlEnable, isTrue);
    });

    test('VideoLayout roundtrip', () {
      final layout = VideoLayoutMsg()
        ..supportsFullDesktopCapture = true
        ..primaryScreenId = 3;
      final track = VideoTrackLayout()
        ..mediaStreamId = 'screen_stream_3'
        ..positionX = 0
        ..positionY = 0
        ..width = 1920
        ..height = 1080
        ..screenId = 3;
      layout.videoTracks.add(track);

      final decoded = decodeControlMessage(encodeControlMessage(videoLayout: layout));
      expect(decoded.videoLayout!.videoTracks, hasLength(1));
      expect(decoded.videoLayout!.videoTracks.first.mediaStreamId, 'screen_stream_3');
      expect(decoded.videoLayout!.videoTracks.first.width, 1920);
      expect(decoded.videoLayout!.primaryScreenId, 3);
    });
  });

  group('Jingle', () {
    test('session-initiate build + parse roundtrip', () {
      final builder = JingleBuilder()
        ..localJid = 'dev@quickdesk.local/chromoting_ftl_android_1'
        ..remoteJid = 'dev@quickdesk.local/chromoting_ftl_quickdesk_host';
      builder.generateSessionId();

      final xml = builder.buildSessionInitiate(
        'v=0\r\no=- 1 1 IN IP4 0.0.0.0\r\n',
        AuthMessage(supportedMethods: 'spake2_curve25519'),
      );

      final parsed = JingleParser().parse(xml);
      expect(parsed, isNotNull);
      expect(parsed!.action, 'session-initiate');
      expect(parsed.sid, builder.sessionId);
      expect(parsed.initiator, builder.localJid);
      expect(parsed.sdp!.type, 'offer');
      expect(parsed.sdp!.sdp, contains('v=0'));
      expect(parsed.authMessage!.supportedMethods, 'spake2_curve25519');
    });

    test('transport-info candidate build + parse roundtrip', () {
      final builder = JingleBuilder()
        ..localJid = 'a@x/1'
        ..remoteJid = 'b@x/2';
      builder.generateSessionId();

      const candidateStr =
          'candidate:842163049 1 udp 1677729535 1.2.3.4 46154 typ srflx';
      final xml = builder.buildTransportInfo(IceCandidateInfo(
        candidate: candidateStr,
        sdpMid: '0',
        sdpMLineIndex: 0,
      ));

      final parsed = JingleParser().parse(xml);
      expect(parsed!.action, 'transport-info');
      expect(parsed.iceCandidates, hasLength(1));
      expect(parsed.iceCandidates.first.candidate, candidateStr);
      expect(parsed.iceCandidates.first.sdpMid, '0');
    });

    test('session-info auth message roundtrip', () {
      final builder = JingleBuilder()
        ..localJid = 'a@x/1'
        ..remoteJid = 'b@x/2';
      builder.generateSessionId();

      final xml = builder.buildSessionInfo(AuthMessage(
        method: 'spake2_curve25519',
        spakeMessage: 'c3Bha2U=',
        verificationHash: 'aGFzaA==',
      ));

      final parsed = JingleParser().parse(xml);
      expect(parsed!.action, 'session-info');
      expect(parsed.authMessage!.method, 'spake2_curve25519');
      expect(parsed.authMessage!.spakeMessage, 'c3Bha2U=');
      expect(parsed.authMessage!.verificationHash, 'aGFzaA==');
    });

    test('session-terminate + iq result', () {
      final builder = JingleBuilder()
        ..localJid = 'a@x/1'
        ..remoteJid = 'b@x/2';
      builder.generateSessionId();

      final parsed = JingleParser().parse(builder.buildSessionTerminate('success'));
      expect(parsed!.action, 'session-terminate');
      expect(parsed.terminateInfo!.reason, 'success');

      final iqResult = JingleParser().parse(builder.buildIqResult('iq-42', 'b@x/2'));
      expect(iqResult!.action, isNot('session-terminate'));
    });
  });
}
