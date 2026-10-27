import 'dart:async';

import 'package:flutter_test/flutter_test.dart';
import 'package:quickdesk_android/protocol/host_clients.dart';

class FakeHostClient implements HostClientConnection {
  @override
  final DateTime connectedAt;
  @override
  bool isConnected = true;
  final Completer<void> finished = Completer<void>();
  bool? sentTerminate;
  FakeHostClient(this.connectedAt);
  @override
  Future<void> close({
    required bool sendTerminate,
    String terminateReason = 'success',
  }) {
    isConnected = false; // production peer likewise stops input synchronously
    sentTerminate = sendTerminate;
    return finished.future;
  }
}

void main() {
  test(
    'kick closes only matching connected peer and sends terminate',
    () async {
      final a = FakeHostClient(DateTime(2026));
      final b = FakeHostClient(DateTime(2027));
      final peers = <String, HostClientConnection>{'a': a, 'b': b};
      final pending = disconnectHostClient(
        peers,
        'a',
        connectedAt: a.connectedAt,
      );
      expect(a.isConnected, false);
      expect(a.sentTerminate, true);
      expect(b.isConnected, true);
      expect(b.sentTerminate, isNull);
      a.finished.complete();
      expect(await pending, true);
      expect(peers.keys, ['b']);
      expect(await disconnectHostClient(peers, 'a'), false);
    },
  );

  test('stale confirmation and old close cannot remove replacement', () async {
    final old = FakeHostClient(DateTime(2026));
    final replacement = FakeHostClient(DateTime(2027));
    final peers = <String, HostClientConnection>{'same': replacement};
    expect(
      await disconnectHostClient(peers, 'same', connectedAt: old.connectedAt),
      false,
    );
    expect(replacement.sentTerminate, isNull);
    peers['same'] = old;
    final pending = disconnectHostClient(
      peers,
      'same',
      connectedAt: old.connectedAt,
    );
    peers['same'] = replacement;
    old.finished.complete();
    expect(await pending, false);
    expect(peers['same'], same(replacement));
    expect(replacement.isConnected, true);
  });

  test('already closing peers do not receive another termination', () async {
    final peer = FakeHostClient(DateTime(2026));
    final peers = <String, HostClientConnection>{'one': peer};
    final pending = disconnectHostClient(peers, 'one');
    expect(await disconnectHostClient(peers, 'one'), false);
    peer.finished.complete();
    expect(await pending, true);
  });
}
