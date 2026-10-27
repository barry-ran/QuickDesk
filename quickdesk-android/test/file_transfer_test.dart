import 'dart:async';
import 'dart:io';
import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';
import 'package:quickdesk_android/protocol/file_transfer.dart';
import 'package:quickdesk_android/protocol/proto/wire.dart';

class FakeFileChannel implements FileTransferChannel {
  @override
  bool isOpen = true;
  int bufferedAmount = 0;
  @override
  Future<int> bufferedBytes() async => bufferedAmount;
  @override
  void Function()? onOpen;
  @override
  void Function(Uint8List)? onData;
  @override
  void Function()? onClosed;
  final sent = <Uint8List>[];
  bool closed = false;

  @override
  Future<void> send(Uint8List data) async {
    if (closed) throw StateError('closed');
    sent.add(data);
  }

  @override
  Future<void> close() async {
    closed = true;
  }

  void reply(Uint8List data) => onData?.call(data);
}

Uint8List emptyField(int number) => lengthDelimitedField(number, Uint8List(0));
Uint8List metadata(String name, int size) => lengthDelimitedField(1,
    concatBytes([lengthDelimitedField(1, name), varintField(2, size)]));
Uint8List payload(List<int> bytes) =>
    lengthDelimitedField(2, lengthDelimitedField(1, Uint8List.fromList(bytes)));
int field(Uint8List message) => ProtobufReader(message).readTag().fieldNumber;

Future<void> waitUntil(bool Function() condition) async {
  final end = DateTime.now().add(const Duration(seconds: 5));
  while (!condition()) {
    if (DateTime.now().isAfter(end)) fail('condition timed out');
    await Future<void>.delayed(const Duration(milliseconds: 2));
  }
}

void main() {
  late Directory dir;
  final managers = <FileTransferManager>[];
  setUp(() async {
    dir = await Directory.systemTemp.createTemp('quickdesk-file-test-');
  });
  tearDown(() async {
    for (final manager in managers) { manager.dispose(); }
    managers.clear();
    // Let cleanup close owned files before removing the test root.
    await Future<void>.delayed(const Duration(milliseconds: 30));
    if (await dir.exists()) await dir.delete(recursive: true);
  });

  Future<File> local(String name, [int size = 1]) async {
    final file = File('${dir.path}/$name');
    await file.parent.create(recursive: true);
    await file.writeAsBytes(List.filled(size, 7));
    return file;
  }

  FileTransferManager manager(List<FakeFileChannel> channels) {
    final value = FileTransferManager.withChannelFactory((_) async {
      final channel = FakeFileChannel();
      channels.add(channel);
      return channel;
    });
    managers.add(value);
    return value;
  }

  test('uploads stream bounded chunks and wait for success before next file', () async {
    final channels = <FakeFileChannel>[];
    final queue = manager(channels);
    final first = await local('a/same.txt', 20000);
    final second = await local('b/same.txt');
    queue.enqueueUpload('same.txt', first.path);
    queue.enqueueUpload('same.txt', second.path);
    await waitUntil(() => channels.isNotEmpty &&
        channels.first.sent.any((message) => field(message) == 3));
    expect(channels.length, 1);
    expect(queue.tasks[1].status, FileTransferStatus.queued);
    final data = channels.first.sent.where((message) => field(message) == 2);
    expect(data.length, 3);
    expect(data.every((message) => message.length <= 8200), isTrue);
    channels.first.reply(emptyField(4));
    await waitUntil(() => channels.length == 2);
    expect(queue.tasks.first.status, FileTransferStatus.completed);
    final oldCallback = channels.first.onData;
    oldCallback?.call(emptyField(4));
    expect(queue.tasks[1].status, FileTransferStatus.transferring);
  });

  test('duplicate pending path deduplicates while same names stay distinct', () async {
    final channels = <FakeFileChannel>[];
    final queue = manager(channels);
    final a = await local('a/same');
    final b = await local('b/same');
    final id = queue.enqueueUpload('same', a.path);
    expect(queue.enqueueUpload('same', a.path), id);
    queue.enqueueUpload('same', b.path);
    expect(queue.tasks.length, 2);
  });

  test('queued cancellation never opens a channel', () async {
    final channels = <FakeFileChannel>[];
    final queue = manager(channels);
    final a = await local('first');
    final b = await local('second');
    queue.enqueueUpload('first', a.path);
    final second = queue.enqueueUpload('second', b.path);
    queue.cancel(second);
    await waitUntil(() => channels.isNotEmpty && channels.first.sent.any((m) => field(m) == 3));
    channels.first.reply(emptyField(4));
    await waitUntil(() => queue.tasks.first.finished);
    expect(queue.tasks[1].status, FileTransferStatus.cancelled);
    expect(channels.length, 1);
  });

  test('cancel while channel creation pending closes late channel', () async {
    final created = Completer<FileTransferChannel>();
    var requested = false;
    final queue = FileTransferManager.withChannelFactory((_) {
      requested = true;
      return created.future;
    });
    managers.add(queue);
    final file = await local('pending');
    final id = queue.enqueueUpload('pending', file.path);
    await waitUntil(() => requested);
    queue.cancel(id);
    final channel = FakeFileChannel();
    created.complete(channel);
    await waitUntil(() => channel.closed);
    expect(channel.sent, isEmpty);
    expect(queue.tasks.single.status, FileTransferStatus.cancelled);
  });

  test('active cancellation ignores stale success and frees dedicated channel', () async {
    final channels = <FakeFileChannel>[];
    final queue = manager(channels);
    final file = await local('active');
    final id = queue.enqueueUpload('active', file.path);
    await waitUntil(() => channels.isNotEmpty && channels.first.sent.any((m) => field(m) == 3));
    final callback = channels.first.onData;
    queue.cancel(id);
    callback?.call(emptyField(4));
    await waitUntil(() => channels.first.closed);
    expect(queue.tasks.single.status, FileTransferStatus.cancelled);
  });

  test('download serializes writes before end and never overwrites existing file', () async {
    final channels = <FakeFileChannel>[];
    final queue = manager(channels);
    final original = await local('same.txt', 4);
    queue.enqueueDownload(dir.path);
    await waitUntil(() => channels.isNotEmpty && channels.first.sent.isNotEmpty);
    final channel = channels.first;
    channel.reply(metadata('same.txt', 3));
    channel.reply(payload([1, 2]));
    channel.reply(payload([3]));
    channel.reply(emptyField(3));
    await waitUntil(() => queue.tasks.single.finished);
    final task = queue.tasks.single;
    expect(task.status, FileTransferStatus.completed);
    expect(task.savedPath, isNot(original.path));
    expect(await File(task.savedPath!).readAsBytes(), [1, 2, 3]);
    expect(await original.readAsBytes(), [7, 7, 7, 7]);
    expect(channel.sent.any((message) => field(message) == 4), isTrue);
  });

  test('truncated download fails without success or visible final file', () async {
    final channels = <FakeFileChannel>[];
    final queue = manager(channels);
    queue.enqueueDownload(dir.path);
    await waitUntil(() => channels.isNotEmpty && channels.first.sent.isNotEmpty);
    channels.first.reply(metadata('../bad.txt', 5));
    channels.first.reply(payload([1]));
    channels.first.reply(emptyField(3));
    await waitUntil(() => channels.first.closed);
    expect(queue.tasks.single.status, FileTransferStatus.failed);
    expect(queue.tasks.single.savedPath, isNull);
    expect(channels.first.sent.any((message) => field(message) == 4), isFalse);
    await waitUntil(() => dir.listSync().isEmpty);
  });

  test('directory upload fails and does not open channel', () async {
    final channels = <FakeFileChannel>[];
    final queue = manager(channels);
    queue.enqueueUpload('directory', dir.path);
    await waitUntil(() => queue.tasks.single.finished);
    expect(queue.tasks.single.status, FileTransferStatus.failed);
    expect(queue.tasks.single.error, contains('Folders'));
    expect(channels, isEmpty);
  });

  test('session disposal blocks new tasks and ignores late channel events', () async {
    final channels = <FakeFileChannel>[];
    final queue = manager(channels);
    final file = await local('session');
    queue.enqueueUpload('session', file.path);
    await waitUntil(() => channels.isNotEmpty && channels.first.sent.isNotEmpty);
    final callback = channels.first.onData;
    queue.dispose();
    callback?.call(emptyField(4));
    await waitUntil(() => channels.first.closed);
    expect(queue.tasks.single.status, FileTransferStatus.cancelled);
    expect(() => queue.enqueueDownload(dir.path), throwsStateError);
  });

  test('empty uploads still require remote success', () async {
    final channels = <FakeFileChannel>[];
    final queue = manager(channels);
    final file = await local('empty', 0);
    queue.enqueueUpload('empty', file.path);
    await waitUntil(() => channels.isNotEmpty && channels.first.sent.any((m) => field(m) == 3));
    expect(queue.tasks.single.finished, isFalse);
    channels.first.reply(emptyField(4));
    await waitUntil(() => queue.tasks.single.status == FileTransferStatus.completed);
  });

  test('backpressure timeout fails rather than sending beyond the window', () async {
    final channel = FakeFileChannel()..bufferedAmount = 1000000;
    final queue = FileTransferManager.withChannelFactory((_) async => channel,
        idleTimeout: const Duration(milliseconds: 50));
    managers.add(queue);
    final file = await local('backpressure', 20000);
    queue.enqueueUpload('backpressure', file.path);
    await waitUntil(() => channel.closed);
    expect(queue.tasks.single.status, FileTransferStatus.failed);
    expect(channel.sent.where((message) => field(message) == 2), isEmpty);
  });

  test('download retains 64-bit size counters without allocating content', () async {
    final channels = <FakeFileChannel>[];
    final queue = manager(channels);
    final id = queue.enqueueDownload(dir.path);
    await waitUntil(() => channels.isNotEmpty && channels.first.sent.isNotEmpty);
    channels.first.reply(metadata('large', 0x100000007));
    await waitUntil(() => queue.tasks.single.totalBytes > 0);
    expect(queue.tasks.single.totalBytes, 0x100000007);
    queue.cancel(id);
    await waitUntil(() => channels.first.closed);
    await waitUntil(() => dir.listSync().isEmpty);
    expect(queue.tasks.single.savedPath, isNull);
  });

  test('filename normalization removes traversal and staging collisions', () {
    expect(FileTransferManager.sanitizeDownloadName('../a/b\\c'), '.._a_b_c');
    expect(FileTransferManager.sanitizeDownloadName('..'), 'download');
    expect(FileTransferManager.sanitizeDownloadName('payload.part'), 'download-payload.part');
  });
}
