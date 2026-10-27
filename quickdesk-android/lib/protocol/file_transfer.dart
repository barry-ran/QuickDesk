/// file_transfer.dart - 文件传输（主控端）
///
/// 对照 WebClient/js/protocol/datachannel-handler.js 的实现，走 remoting 的
/// file_transfer.proto，通过按传输独立的 `filetransfer-<n>` DataChannel 传送：
///
///   FileTransfer {
///     Metadata metadata = 1;        // { string filename=1; int64 size=2; }
///     Data     data     = 2;        // { bytes data=1; }
///     Empty    end      = 3;
///     Empty    success  = 4;
///     Empty    request_transfer = 5;
///     Error    error    = 6;        // { ErrorType type=1; }
///   }
///
/// 上传（client→host）：建通道 → metadata → data* → end → 等 success
/// 下载（host→client）：建通道 → request_transfer → metadata → data* → end → 回 success
library;

import 'dart:async';
import 'dart:io';
import 'dart:typed_data';

import 'package:flutter/foundation.dart';
import 'package:flutter_webrtc/flutter_webrtc.dart';

import 'datachannel_config.dart';
import 'proto/wire.dart';

const int _chunkSize = 8192;
const int _bufferHighWater = _chunkSize * 8;

// ==================== 编码 ====================

Uint8List _encMetadata(String filename, int size) => lengthDelimitedField(
    1,
    concatBytes([
      lengthDelimitedField(1, filename),
      varintField(2, size),
    ]));

Uint8List _encData(Uint8List data) =>
    lengthDelimitedField(2, lengthDelimitedField(1, data));

Uint8List _encEnd() => lengthDelimitedField(3, Uint8List(0));
Uint8List _encSuccess() => lengthDelimitedField(4, Uint8List(0));
Uint8List _encRequestTransfer() => lengthDelimitedField(5, Uint8List(0));

// ==================== 解码 ====================

class _FtMessage {
  String? metaFilename;
  int? metaSize;
  Uint8List? data;
  bool end = false;
  bool success = false;
  int? errorType;
}

_FtMessage _decode(Uint8List data) {
  final msg = _FtMessage();
  final reader = ProtobufReader(data);
  while (reader.hasMore) {
    final tag = reader.readTag();
    if (tag.wireType != 2) {
      reader.skipField(tag.wireType);
      continue;
    }
    switch (tag.fieldNumber) {
      case 1:
        _decodeMetadata(reader.readBytes(), msg);
        break;
      case 2:
        msg.data = _decodeDataField(reader.readBytes());
        break;
      case 3:
        reader.skipField(tag.wireType);
        msg.end = true;
        break;
      case 4:
        reader.skipField(tag.wireType);
        msg.success = true;
        break;
      case 6:
        final errReader = ProtobufReader(reader.readBytes());
        msg.errorType = 0;
        while (errReader.hasMore) {
          final et = errReader.readTag();
          if (et.fieldNumber == 1 && et.wireType == 0) {
            msg.errorType = errReader.readVarint();
          } else {
            errReader.skipField(et.wireType);
          }
        }
        break;
      default:
        reader.skipField(tag.wireType);
    }
  }
  return msg;
}

void _decodeMetadata(Uint8List data, _FtMessage msg) {
  final reader = ProtobufReader(data);
  while (reader.hasMore) {
    final tag = reader.readTag();
    if (tag.fieldNumber == 1 && tag.wireType == 2) {
      msg.metaFilename = reader.readString();
    } else if (tag.fieldNumber == 2 && tag.wireType == 0) {
      msg.metaSize = reader.readVarint();
    } else {
      reader.skipField(tag.wireType);
    }
  }
}

Uint8List _decodeDataField(Uint8List data) {
  final reader = ProtobufReader(data);
  while (reader.hasMore) {
    final tag = reader.readTag();
    if (tag.fieldNumber == 1 && tag.wireType == 2) {
      return reader.readBytes();
    }
    reader.skipField(tag.wireType);
  }
  return Uint8List(0);
}

// ==================== Transport seam and task model ====================

/// Existing remoting wire messages, on one dedicated channel per task.
/// Injectable so lifecycle and filesystem behavior are tested without WebRTC.
abstract class FileTransferChannel {
  bool get isOpen;
  int get bufferedAmount;
  set onOpen(void Function()? callback);
  set onData(void Function(Uint8List)? callback);
  set onClosed(void Function()? callback);
  Future<void> send(Uint8List data);
  Future<void> close();
}

class _RtcFileTransferChannel implements FileTransferChannel {
  final RTCDataChannel channel;
  void Function()? _open;
  void Function(Uint8List)? _data;
  void Function()? _closed;

  _RtcFileTransferChannel(this.channel) {
    channel.onDataChannelState = (state) {
      if (state == RTCDataChannelState.RTCDataChannelOpen) _open?.call();
      if (state == RTCDataChannelState.RTCDataChannelClosed) _closed?.call();
    };
    channel.onMessage = (message) {
      if (message.isBinary) _data?.call(message.binary);
    };
  }

  @override
  bool get isOpen => channel.state == RTCDataChannelState.RTCDataChannelOpen;
  @override
  int get bufferedAmount => channel.bufferedAmount ?? 0;
  @override
  set onOpen(void Function()? callback) => _open = callback;
  @override
  set onData(void Function(Uint8List)? callback) => _data = callback;
  @override
  set onClosed(void Function()? callback) => _closed = callback;
  @override
  Future<void> send(Uint8List data) =>
      channel.send(RTCDataChannelMessage.fromBinary(data));
  @override
  Future<void> close() => channel.close();
}

enum FileTransferStatus { queued, transferring, completed, cancelled, failed }

class FileTransferTask {
  final String id;
  final bool download;
  final String source;
  String filename;
  FileTransferStatus status = FileTransferStatus.queued;
  int bytes = 0;
  int totalBytes = 0;
  String? error;
  String? savedPath;

  FileTransferTask(this.id, this.filename, this.source, {this.download = false});

  bool get finished =>
      status == FileTransferStatus.completed ||
      status == FileTransferStatus.cancelled ||
      status == FileTransferStatus.failed;
  double get fraction =>
      totalBytes > 0 ? (bytes / totalBytes).clamp(0.0, 1.0).toDouble() : 0;
}

class FileTransferManager extends ChangeNotifier {
  final Future<FileTransferChannel> Function(String) _createChannel;
  final Duration idleTimeout;
  final List<FileTransferTask> _tasks = [];
  final Map<String, void Function()> _cancelHooks = {};
  int _seq = 0;
  bool _disposed = false;
  bool _running = false;

  FileTransferManager(RTCPeerConnection Function() pcProvider)
      : this.withChannelFactory((label) async => _RtcFileTransferChannel(
            await pcProvider().createDataChannel(
                label, createRemotingDataChannelInit())));

  FileTransferManager.withChannelFactory(this._createChannel,
      {this.idleTimeout = const Duration(seconds: 60)});

  List<FileTransferTask> get tasks => List.unmodifiable(_tasks);

  String enqueueUpload(String filename, String path) {
    if (_disposed) throw StateError('This session is closed');
    final source = File(path).absolute.path;
    for (final task in _tasks) {
      if (!task.finished && !task.download && task.source == source) return task.id;
    }
    return _enqueue(filename, source, false);
  }

  String enqueueDownload(String saveDir) {
    if (_disposed) throw StateError('This session is closed');
    // One outstanding remote chooser at a time.
    for (final task in _tasks) {
      if (!task.finished && task.download) return task.id;
    }
    return _enqueue('Download', saveDir, true);
  }

  String _enqueue(String filename, String source, bool download) {
    if (_tasks.where((task) => !task.finished).length >= 256) {
      throw StateError('The transfer queue is full');
    }
    final task = FileTransferTask('filetransfer-${_seq++}', filename, source,
        download: download);
    _tasks.add(task);
    _changed();
    _pump();
    return task.id;
  }

  void cancel(String id) {
    for (final task in _tasks) {
      if (task.id != id || task.finished) continue;
      task.status = FileTransferStatus.cancelled;
      task.error = task.download
          ? 'Download stopped locally; temporary data will be removed.'
          : 'Stopped locally. A partial file may remain on the remote host.';
      _cancelHooks[id]?.call();
      _changed();
      return;
    }
  }

  void _changed() {
    if (!_disposed) notifyListeners();
  }

  void _pump() {
    if (_disposed || _running) return;
    final pending = _tasks.where((task) => task.status == FileTransferStatus.queued);
    if (pending.isEmpty) return;
    _running = true;
    final task = pending.first;
    task.status = FileTransferStatus.transferring;
    _changed();
    unawaited(_run(task).whenComplete(() {
      _running = false;
      _pump();
    }));
  }

  Future<void> _run(FileTransferTask task) async {
    FileTransferChannel? channel;
    RandomAccessFile? input;
    RandomAccessFile? output;
    Directory? staging;
    final finished = Completer<void>();
    var inbound = Future<void>.value();
    Timer? timer;
    String? failure;
    bool ended = false;
    bool opened = false;
    bool metadataReceived = false;
    bool sentEnd = false;
    bool succeeded = false;

    void stop([Object? error]) {
      if (ended) return;
      ended = true;
      if (error != null) {
        failure = '$error';
        if (!task.finished) {
          task.status = FileTransferStatus.failed;
          task.error = failure;
          _changed();
        }
      }
      timer?.cancel();
      finished.complete();
    }

    void touch() {
      timer?.cancel();
      timer = Timer(idleTimeout, () => stop('File transfer timed out'));
    }

    bool getStopped() => ended || _disposed || task.finished;

    _cancelHooks[task.id] = () => stop();
    touch();
    try {
      if (!task.download) {
        if (await FileSystemEntity.type(task.source, followLinks: true) !=
            FileSystemEntityType.file) {
          throw StateError('Folders are not supported. Select a regular file.');
        }
        final openedInput = await File(task.source).open();
        input = openedInput;
        task.totalBytes = await openedInput.length();
      }
      if (getStopped()) return;
      channel = await _createChannel(task.id);
      if (getStopped()) return;
      final transport = channel;

      Future<void> onOpen() async {
        if (opened || getStopped()) return;
        opened = true;
        try {
          if (task.download) {
            await transport.send(_encRequestTransfer());
            touch();
            return;
          }
          await transport.send(_encMetadata(task.filename, task.totalBytes));
          while (task.bytes < task.totalBytes && !getStopped()) {
            while (transport.bufferedAmount > _bufferHighWater && !getStopped()) {
              await Future<void>.delayed(const Duration(milliseconds: 20));
            }
            if (getStopped()) return;
            final remaining = task.totalBytes - task.bytes;
            final chunk = await input!.read(
                remaining < _chunkSize ? remaining : _chunkSize);
            if (getStopped()) return;
            if (chunk.isEmpty) throw StateError('The local file changed during upload');
            await transport.send(_encData(chunk));
            if (getStopped()) return;
            task.bytes += chunk.length;
            touch();
            _changed();
          }
          if (!getStopped()) {
            sentEnd = true;
            await transport.send(_encEnd());
            touch();
          }
        } catch (error) {
          stop(error);
        }
      }

      Future<void> onData(Uint8List bytes) async {
        if (getStopped()) return;
        final message = _decode(bytes);
        touch();
        if (message.errorType != null) {
          stop('Host error ${message.errorType}');
          return;
        }
        if (!task.download) {
          if (message.success && sentEnd) {
            succeeded = true;
            stop();
          }
          return;
        }
        if (message.metaFilename != null || message.metaSize != null) {
          if (metadataReceived || message.metaSize == null || message.metaSize! < 0) {
            throw StateError('Invalid or repeated download metadata');
          }
          metadataReceived = true;
          task.filename = sanitizeDownloadName(message.metaFilename ?? 'download');
          task.totalBytes = message.metaSize!;
          await Directory(task.source).create(recursive: true);
          staging = await Directory(task.source).createTemp('quickdesk-');
          output = await File('${staging!.path}${Platform.pathSeparator}payload.part').open(
              mode: FileMode.write);
          _changed();
        } else if (message.data != null) {
          if (!metadataReceived || output == null ||
              task.bytes + message.data!.length > task.totalBytes) {
            throw StateError('Invalid download data or size');
          }
          await output!.writeFrom(message.data!);
          if (getStopped()) return;
          task.bytes += message.data!.length;
          _changed();
        } else if (message.end) {
          if (!metadataReceived || output == null || task.bytes != task.totalBytes) {
            throw StateError('Incomplete download');
          }
          await output!.flush();
          await output!.close();
          output = null;
          if (getStopped()) return;
          final destination = '${staging!.path}${Platform.pathSeparator}${task.filename}';
          await File('${staging!.path}${Platform.pathSeparator}payload.part').rename(destination);
          if (getStopped()) return;
          task.savedPath = destination;
          await transport.send(_encSuccess());
          if (getStopped()) return;
          succeeded = true;
          stop();
        }
      }

      transport.onOpen = () => unawaited(onOpen());
      transport.onClosed = () => stop('File channel closed before completion');
      transport.onData = (data) {
        // A single ordered chain prevents end/flush racing preceding writes.
        inbound = inbound.then((_) => onData(data)).catchError((Object error) {
          stop(error);
        });
      };
      if (transport.isOpen) unawaited(onOpen());
      await finished.future;
    } catch (error) {
      failure = '$error';
      ended = true;
    } finally {
      ended = true;
      timer?.cancel();
      _cancelHooks.remove(task.id);
      if (channel != null) {
        channel.onOpen = null;
        channel.onData = null;
        channel.onClosed = null;
        try { await channel.close(); } catch (_) {}
      }
      await inbound;
      try { await input?.close(); } catch (_) {}
      try { await output?.close(); } catch (_) {}
      if (!succeeded && staging != null) {
        // Only this task's newly-created private directory is removed.
        try { await staging!.delete(recursive: true); } catch (_) {}
        task.savedPath = null;
      }
      if (!task.finished) {
        task.status = succeeded ? FileTransferStatus.completed : FileTransferStatus.failed;
        task.error = succeeded ? null : (failure ?? 'File transfer stopped');
      }
      _changed();
    }
  }

  /// Peer-supplied filenames never become paths; each download has its own directory.
  static String sanitizeDownloadName(String name) {
    var safe = name.replaceAll(RegExp(r'[\\/:*?"<>|\x00-\x1f]'), '_').trim();
    if (safe.isEmpty || safe == '.' || safe == '..') safe = 'download';
    if (safe.length > 180) safe = safe.substring(0, 180);
    // Do not collide with our private staging filename.
    if (safe == 'payload.part') safe = 'download-payload.part';
    return safe;
  }

  @override
  void dispose() {
    if (_disposed) return;
    _disposed = true;
    for (final task in _tasks) {
      if (!task.finished) {
        task.status = FileTransferStatus.cancelled;
        task.error = 'Session closed';
        _cancelHooks[task.id]?.call();
      }
    }
    super.dispose();
  }
}
