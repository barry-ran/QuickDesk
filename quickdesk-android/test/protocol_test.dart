import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';
import 'package:xml/xml.dart';

import 'package:quickdesk_android/protocol/auth/spake2.dart';
import 'package:quickdesk_android/protocol/auth/spake2_authenticator.dart';
import 'package:quickdesk_android/protocol/datachannel_config.dart';
import 'package:quickdesk_android/protocol/proto/protobuf_messages.dart';
import 'package:quickdesk_android/protocol/signaling/jingle.dart';
import 'package:quickdesk_android/protocol/signaling/sdp_signature.dart';

void main() {
  group('DataChannel', () {
    test('matches Chromium in-band channel defaults', () {
      final init = createRemotingDataChannelInit();

      expect(init.ordered, isTrue);
      expect(init.negotiated, isFalse);
      expect(init.id, -1);
      expect(init.protocol, isEmpty);
      expect(init.maxRetransmitTime, -1);
      expect(init.maxRetransmits, -1);
    });
  });

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

    test('matches Chromium 140 BoringSSL fixed byte vector', () {
      const deviceId = '123456789';
      const accessCode = '424243';
      const clientJid =
          'client_01234567-89ab-4cde-8f01-23456789abcd@quickdesk.local/'
          'chromoting_ftl_quickdesk_host';
      const hostJid = '123456789@quickdesk.local/chromoting_ftl_quickdesk_host';
      const expectedClientMessage =
          '8d57e26fd5c4bc842ebaa235e39ebf1aadceea1d4dc436d91362beeb9a8934c9';
      const expectedHostMessage =
          'caf5b3dc942dc21a66cefbab23fe9f684a42fd2dfa06fb35d59f29019c7ff7c6';
      const expectedAuthKey =
          'e6cf084f085b5d13552e564130c70136d4af595f3d09f099d7f0583fc02e9c6c'
          'fac4bbb5d4818aa0bac3fa249caa7bd1d028db2e48e49772a56fb0d0282db72f';

      final password = getSharedSecretHash(deviceId, deviceId + accessCode);
      final passwordData = passwordToScalar(password);
      expect(
        scReduce(passwordData.hash).toRadixString(16),
        '578e3738c0adbc5fd589554c9ba2ec1a996ea7582585248bda1d119310f5ffc',
      );
      expect(
        passwordData.scalar.toRadixString(16),
        '4578e3738c0adbc5fd589554c9ba2ec1fd12d1f00e36c5a21deb5d82a4e6afb0',
      );
      expect(passwordData.scalar & BigInt.from(7), BigInt.zero);

      final client = Spake2Context(spake2RoleAlice, clientJid, hostJid);
      final host = Spake2Context(spake2RoleBob, hostJid, clientJid);
      final clientMessage = client.generateMessage(
        password,
        privateKeyOverride: BigInt.parse(
          '123456789abcdef123456789abcdef',
          radix: 16,
        ),
      );
      final hostMessage = host.generateMessage(
        password,
        privateKeyOverride: BigInt.parse(
          'fedcba9876543210fedcba987654321',
          radix: 16,
        ),
      );

      expect(bytesToHex(clientMessage), expectedClientMessage);
      expect(bytesToHex(hostMessage), expectedHostMessage);
      expect(bytesToHex(client.processMessage(hostMessage)), expectedAuthKey);
      expect(bytesToHex(host.processMessage(clientMessage)), expectedAuthKey);
      expect(
        bytesToHex(client.getOutgoingVerificationHash()),
        '5592d1521d3cdbb7a2caf217309585f5bfa516b3b72883862613e9959093c78f',
      );
      expect(
        bytesToHex(host.getOutgoingVerificationHash()),
        '30f60ffad1342128f95dd8bbbd48e0096e8df610fe634f9246ac34f12f214297',
      );

      // SPAKE2 将实际完整 JID 写入 transcript。即使曲线消息相同，使用
      // 过期 initiator 而非 IQ `from` 也会派生不同的认证密钥。
      final wrongIdentityHost = Spake2Context(
        spake2RoleBob,
        hostJid,
        'client_stale@quickdesk.local/chromoting_ftl_quickdesk_host',
      );
      final wrongIdentityHostMessage = wrongIdentityHost.generateMessage(
        password,
        privateKeyOverride: BigInt.parse(
          'fedcba9876543210fedcba987654321',
          radix: 16,
        ),
      );
      expect(bytesToHex(wrongIdentityHostMessage), expectedHostMessage);
      expect(
        bytesToHex(wrongIdentityHost.processMessage(clientMessage)),
        isNot(expectedAuthKey),
      );
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

    test('host accepts a directly selected SPAKE2 method', () {
      final host = Spake2HostAuthenticator(
        'host@example.com/chromoting_ftl_host',
        'client@example.com/chromoting_ftl_client',
        Uint8List(32),
        certificate: 'Y2VydA==',
      );

      host.processMessage(AuthMessage(method: kSpake2Method));
      expect(host.state, AuthState.messageReady);
      expect(host.getNextMessage().method, kSpake2Method);
    });

    test('client/host authenticator state machines complete handshake', () {
      const clientJid = 'c@quickdesk.local/chromoting_ftl_android_1';
      const hostJid = 'c@quickdesk.local/chromoting_ftl_quickdesk_host';
      final secret = getSharedSecretHash('c', 'c123456');

      final client = Spake2ClientAuthenticator(clientJid, hostJid, secret);
      final host = Spake2HostAuthenticator(
        hostJid,
        clientJid,
        secret,
        certificate: 'ZHVtbXktZGVyLWNlcnQ=',
      );

      // client → host: supported-methods
      host.processMessage(client.getFirstNegotiationMessage());
      expect(host.state, AuthState.messageReady);

      // host → client: method + certificate + spake message
      final hostFirstMessage = host.getNextMessage();
      expect(hostFirstMessage.certificate, 'ZHVtbXktZGVyLWNlcnQ=');
      client.processMessage(hostFirstMessage);
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

  group('SDP signature', () {
    test('normalizes exactly like Chromium SdpMessage', () {
      const sdp = ' v=0\r\n\r\n o=- 1 1 IN IP4 0.0.0.0 \r\n'
          'a=group:BUNDLE 0\n';

      expect(
        normalizeSdpForSignature(sdp),
        'v=0\no=- 1 1 IN IP4 0.0.0.0\na=group:BUNDLE 0\n',
      );
    });

    test('signs and verifies normalized SDP bytes', () {
      final authKey = Uint8List.fromList(List<int>.generate(64, (i) => i));
      const sdp = 'v=0\r\no=- 1 1 IN IP4 0.0.0.0\r\n';

      final signature = signSdp(authKey, sdp, 'offer');
      expect(signature, 'sqPgIQOE5P6X1JCZcYIn1fOgBxOTfXBnbyitRSK9Lzk=');
      expect(verifySdpSignature(authKey, sdp, 'offer', signature), isTrue);
      expect(verifySdpSignature(authKey, sdp, 'answer', signature), isFalse);
      expect(verifySdpSignature(authKey, sdp, 'offer', 'not-base64'), isFalse);
    });
  });

  group('Protobuf', () {
    test('matches Chromium 140 naked DataChannel byte vectors', () {
      final event = encodeEventMessage(
        timestamp: 1234567890,
        mouseEvent: MouseEventMsg(
          x: -5,
          y: 100,
          button: MouseButton.left.value,
          buttonDown: true,
          wheelDeltaX: 1.25,
          wheelDeltaY: -120.5,
          wheelTicksX: 2.5,
          wheelTicksY: -3.75,
        ),
      );
      expect(
        bytesToHex(event),
        '08d285d8cc04222508fbffffffffffffffff011064280130013d0000a03f'
        '450000f1c24d0000204055000070c0',
      );

      final layout = VideoLayoutMsg()
        ..supportsFullDesktopCapture = false
        ..primaryScreenId = 0;
      layout.videoTracks.add(VideoTrackLayout()
        ..mediaStreamId = 'screen_stream_0'
        ..positionX = -10
        ..positionY = 20
        ..width = 1080
        ..height = 2400
        ..xDpi = 420
        ..yDpi = 420
        ..screenId = 0
        ..displayName = 'Android');
      final control = encodeControlMessage(
        capabilities: 'rateLimitResize videoLayout',
        videoLayout: layout,
      );
      expect(
        bytesToHex(control),
        '321d0a1b726174654c696d6974526573697a6520766964656f4c61796f7574'
        '523b0a350a0f73637265656e5f73747265616d5f3010f6ffffffffffffffff01'
        '181420b80828e01230a40338a40340004a07416e64726f696410001800',
      );

      // WebrtcDataStreamAdapter 直接发送 protobuf payload；若误加传统通道的
      // 4 字节大端长度头，首字节会是 0 而不再是字段 tag 0x08/0x32。
      expect(event.first, 0x08);
      expect(control.first, 0x32);
    });

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
      expect(decoded.capabilities, 'fileTransfer privacyScreen');
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

      final decoded =
          decodeControlMessage(encodeControlMessage(videoLayout: layout));
      expect(decoded.videoLayout!.videoTracks, hasLength(1));
      expect(decoded.videoLayout!.videoTracks.first.mediaStreamId,
          'screen_stream_3');
      expect(decoded.videoLayout!.videoTracks.first.width, 1920);
      expect(decoded.videoLayout!.primaryScreenId, 3);
    });
  });

  group('Jingle', () {
    test('Chromium auth-only session-accept selects WebRTC', () {
      final builder = JingleBuilder()
        ..localJid = 'host@quickdesk.local/chromoting_ftl_quickdesk_host'
        ..remoteJid = 'client@quickdesk.local/chromoting_ftl_desktop'
        ..sessionId = '123456789';

      final xml = builder.buildSessionAccept(
        null,
        AuthMessage(
          method: 'spake2_curve25519',
          spakeMessage: 'c3Bha2U=',
          certificate: 'Y2VydA==',
        ),
      );
      final doc = XmlDocument.parse(xml);
      final root = doc.rootElement;
      final parsed = JingleParser().parse(xml)!;
      final transport = doc.descendantElements.firstWhere(
        (element) =>
            element.name.local == 'transport' &&
            element.getAttribute('xmlns') == nsWebrtcTransport,
      );

      expect(root.name.local, 'iq');
      expect(root.name.prefix, 'cli');
      expect(root.getAttribute('xmlns:cli'), nsJabberClient);
      expect(parsed.action, 'session-accept');
      expect(parsed.sdp, isNull);
      expect(parsed.authMessage!.certificate, 'Y2VydA==');
      expect(transport.childElements, isEmpty);

      // 对照 Chromium 140 JingleMessage::ParseXml：外层 WebRTC transport
      // 的存在会把 CandidateSessionConfig.webrtc_supported 设为 true；
      // description 内无需旧 ICE 协议的 standard-ice/channel 配置。
      final description = doc.descendantElements.firstWhere(
        (element) =>
            element.name.local == 'description' &&
            element.getAttribute('xmlns') == nsChromoting,
      );
      expect(
        description.childElements
            .where((element) => element.name.local == 'standard-ice'),
        isEmpty,
      );
      expect(
        description.childElements.where(
          (element) =>
              element.name.local == 'control' ||
              element.name.local == 'event' ||
              element.name.local == 'video' ||
              element.name.local == 'audio',
        ),
        isEmpty,
      );
    });

    test('outgoing IQ IDs are consecutive and responses reuse request ID', () {
      final builder = JingleBuilder()
        ..localJid = 'host@x/1'
        ..remoteJid = 'client@x/2'
        ..sessionId = '42';

      final first = JingleParser().parse(builder.buildSessionInfo(
        AuthMessage(method: 'spake2_curve25519'),
      ))!;
      final response = JingleParser().parse(
        builder.buildIqResult('remote-prefix_17', builder.remoteJid),
      )!;
      final second = JingleParser().parse(builder.buildTransportInfoSdp(
        'v=0\r\n',
        'offer',
        signature: 'c2ln',
      ))!;

      final firstParts = first.iqId.split('_');
      final secondParts = second.iqId.split('_');
      expect(firstParts, hasLength(2));
      expect(secondParts, hasLength(2));
      expect(firstParts.first, secondParts.first);
      expect(int.parse(firstParts.last), 1);
      expect(int.parse(secondParts.last), 2);
      expect(response.iqId, 'remote-prefix_17');
    });

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
        certificate: 'Y2VydA==',
      ));

      final parsed = JingleParser().parse(xml);
      expect(parsed!.action, 'session-info');
      expect(parsed.authMessage!.method, 'spake2_curve25519');
      expect(parsed.authMessage!.spakeMessage, 'c3Bha2U=');
      expect(parsed.authMessage!.verificationHash, 'aGFzaA==');
      expect(parsed.authMessage!.certificate, 'Y2VydA==');
    });

    test('session-terminate + iq result', () {
      final builder = JingleBuilder()
        ..localJid = 'a@x/1'
        ..remoteJid = 'b@x/2';
      builder.generateSessionId();

      final parsed =
          JingleParser().parse(builder.buildSessionTerminate('success'));
      expect(parsed!.action, 'session-terminate');
      expect(parsed.terminateInfo!.reason, 'success');

      final iqResult =
          JingleParser().parse(builder.buildIqResult('iq-42', 'b@x/2'));
      expect(iqResult!.action, isNot('session-terminate'));
    });
  });
}
