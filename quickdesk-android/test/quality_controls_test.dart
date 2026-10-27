import 'dart:typed_data';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_webrtc/flutter_webrtc.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:quickdesk_android/core/quality_settings.dart';
import 'package:quickdesk_android/controller/remote/quality_sheet.dart';
import 'package:quickdesk_android/l10n/app_strings.dart';
import 'package:quickdesk_android/protocol/datachannel_handler.dart';
import 'package:quickdesk_android/protocol/host_session.dart';
import 'package:quickdesk_android/protocol/proto/protobuf_messages.dart';

class FakeControlChannel implements RTCDataChannel {
  final List<Uint8List> sent = [];
  @override
  String get label => 'control';
  @override
  int get id => 1;
  @override
  RTCDataChannelState get state => RTCDataChannelState.RTCDataChannelOpen;
  @override
  set onDataChannelState(void Function(RTCDataChannelState)? callback) {}
  @override
  set onMessage(void Function(RTCDataChannelMessage)? callback) {}
  @override
  Future<void> send(RTCDataChannelMessage message) async {
    sent.add(message.binary);
  }

  @override
  Future<void> close() async {}
  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}

void main() {
  setUp(() {
    SharedPreferences.setMockInitialValues({});
    L10n.locale = AppLocale.en;
  });

  test('quality wire vectors match WebClient control.proto field numbers', () {
    expect(encodeControlMessage(targetFramerate: 30), [26, 4, 8, 1, 40, 30]);
    expect(encodeControlMessage(preferredMinBitrateBps: 0), [114, 2, 8, 0]);
    expect(encodeControlMessage(audioControlEnable: false), [42, 2, 8, 0]);
    expect(
      encodeControlMessage(
        clientResolution: const ClientResolutionMsg(1280, 720, screenId: 3),
      ),
      [18, 12, 8, 128, 10, 16, 208, 5, 40, 96, 48, 96, 56, 3],
    );
  });

  test('control readiness, range checks and dispose prevent sends', () async {
    final handler = DataChannelHandler();
    final channel = FakeControlChannel();
    handler.sendTargetFramerate(30);
    expect(channel.sent, isEmpty);
    handler.handleDataChannel(channel);
    handler.sendTargetFramerate(60);
    handler.sendPreferredMinBitrate(0);
    handler.sendClientResolution(const ClientResolutionMsg(1920, 1080));
    expect(channel.sent.length, 3);
    expect(() => handler.sendTargetFramerate(0), throwsArgumentError);
    expect(() => handler.sendPreferredMinBitrate(-1), throwsArgumentError);
    expect(
      () => handler.sendClientResolution(const ClientResolutionMsg(0, 720)),
      throwsArgumentError,
    );
    await handler.dispose();
    handler.sendTargetFramerate(30);
    expect(channel.sent.length, 3);
  });

  test('codec preference preserves supported fallbacks and invalid storage defaults', () async {
    final codecs = ['video/VP8', 'video/rtx', 'video/H264', 'video/red'];
    expect(preferCodec(codecs, 'h264', (value) => value), [
      'video/H264',
      'video/VP8',
      'video/rtx',
      'video/red',
    ]);
    expect(preferCodec(codecs, 'AV1', (value) => value), codecs);
    expect(codecs.first, 'video/VP8');
    await QualitySettings.saveCodec('VP9');
    expect(await QualitySettings.loadCodec(), 'VP9');
    expect(() => QualitySettings.saveCodec('unknown'), throwsArgumentError);
    SharedPreferences.setMockInitialValues({
      'preferred_video_codec': 'invalid',
    });
    expect(await QualitySettings.loadCodec(), '');
  });

  test('unknown and empty host capabilities never enable quality requests', () {
    expect(canRequestDesktopQuality(''), isFalse);
    expect(canRequestDesktopQuality('unknown fileTransferFake'), isFalse);
    expect(canRequestDesktopQuality('fileTransfer virtualDisplay'), isTrue);
  });

  test(
    'unknown client disconnect is idempotent and snapshots immutable',
    () async {
      final session = HostSession(
        signalingUrl: 'wss://example.invalid',
        deviceId: '1',
        sharedSecretHash: Uint8List(32),
        hostCertificate: '',
        screenStreamProvider: () => throw StateError('No capture expected'),
        screenWidth: 1,
        screenHeight: 1,
      );
      expect(session.connectedClients, isEmpty);
      expect(
        () => session.connectedClients.add(
          ConnectedHostClient(clientId: 'x', connectedAt: DateTime.now()),
        ),
        throwsUnsupportedError,
      );
      await session.disconnectClient('unknown');
      await session.disconnectClient('unknown');
      expect(session.peerCount, 0);
      await session.stop();
      session.dispose();
    },
  );

  testWidgets(
    'unknown host disables requests but leaves local mute and codec enabled',
    (tester) async {
      bool? muted;
      var requests = 0;
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: QualitySheet(
              canRequest: false,
              muted: false,
              onMute: (value) => muted = value,
              onFramerate: (value) => requests++,
              onBitrate: (value) => requests++,
              onResolution: (width, height) => requests++,
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      final selectors = tester.widgetList<DropdownButtonFormField<int>>(
        find.byType(DropdownButtonFormField<int>),
      );
      expect(selectors.every((field) => field.onChanged == null), isTrue);
      await tester.tap(find.byType(Switch));
      await tester.pumpAndSettle();
      expect(muted, true);
      expect(requests, 0);
      expect(find.text('Preferred codec (next connection)'), findsOneWidget);
      expect(tester.takeException(), isNull);
    },
  );
}
