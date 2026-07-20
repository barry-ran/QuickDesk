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

// ==================== 进度模型 ====================

class FileTransferProgress {
  final String filename;
  final int bytes;
  final int totalBytes;
  final bool done;
  final String? error;
  final String? savedPath; // 下载完成时的落盘路径

  FileTransferProgress({
    required this.filename,
    required this.bytes,
    required this.totalBytes,
    this.done = false,
    this.error,
    this.savedPath,
  });

  double get fraction => totalBytes > 0 ? bytes / totalBytes : 0;
}

// ==================== 管理器 ====================

class FileTransferManager {
  final RTCPeerConnection Function() pcProvider;
  int _seq = 0;

  FileTransferManager(this.pcProvider);

  /// 上传本地文件到被控端。
  Stream<FileTransferProgress> upload(String filename, Uint8List bytes) {
    final ctrl = StreamController<FileTransferProgress>();
    final id = _seq++;
    final total = bytes.length;
    var cancelled = false;

    Future<void> run() async {
      RTCDataChannel channel;
      try {
        channel = await pcProvider().createDataChannel(
          'filetransfer-$id',
          createRemotingDataChannelInit(),
        );
      } catch (e) {
        ctrl.add(FileTransferProgress(
            filename: filename, bytes: 0, totalBytes: total, error: '$e'));
        await ctrl.close();
        return;
      }

      channel.onDataChannelState = (state) async {
        if (state != RTCDataChannelState.RTCDataChannelOpen) return;
        try {
          await channel.send(
              RTCDataChannelMessage.fromBinary(_encMetadata(filename, total)));
          var offset = 0;
          while (offset < total && !cancelled) {
            final end =
                (offset + _chunkSize) > total ? total : offset + _chunkSize;
            final chunk = Uint8List.sublistView(bytes, offset, end);
            await channel
                .send(RTCDataChannelMessage.fromBinary(_encData(chunk)));
            offset = end;
            ctrl.add(FileTransferProgress(
                filename: filename, bytes: offset, totalBytes: total));
            var guard = 0;
            while ((channel.bufferedAmount ?? 0) > _bufferHighWater &&
                guard < 500) {
              await Future<void>.delayed(const Duration(milliseconds: 20));
              guard++;
            }
          }
          if (!cancelled) {
            await channel.send(RTCDataChannelMessage.fromBinary(_encEnd()));
          }
        } catch (e) {
          ctrl.add(FileTransferProgress(
              filename: filename, bytes: 0, totalBytes: total, error: '$e'));
          await ctrl.close();
        }
      };

      channel.onMessage = (msg) async {
        if (!msg.isBinary) return;
        final ft = _decode(msg.binary);
        if (ft.success) {
          ctrl.add(FileTransferProgress(
              filename: filename, bytes: total, totalBytes: total, done: true));
          await channel.close();
          await ctrl.close();
        } else if (ft.errorType != null) {
          cancelled = true;
          ctrl.add(FileTransferProgress(
              filename: filename,
              bytes: 0,
              totalBytes: total,
              error: 'host error ${ft.errorType}'));
          await channel.close();
          await ctrl.close();
        }
      };
    }

    run();
    return ctrl.stream;
  }

  /// 从被控端下载文件，落盘到 [saveDir]/<对端文件名>。
  Stream<FileTransferProgress> download(String saveDir) {
    final ctrl = StreamController<FileTransferProgress>();
    final id = _seq++;

    Future<void> run() async {
      RTCDataChannel channel;
      try {
        channel = await pcProvider().createDataChannel(
          'filetransfer-$id',
          createRemotingDataChannelInit(),
        );
      } catch (e) {
        ctrl.add(FileTransferProgress(
            filename: '', bytes: 0, totalBytes: 0, error: '$e'));
        await ctrl.close();
        return;
      }

      var filename = 'download';
      var total = 0;
      var received = 0;
      IOSink? sink;
      String? savePath;

      channel.onDataChannelState = (state) async {
        if (state != RTCDataChannelState.RTCDataChannelOpen) return;
        try {
          await channel
              .send(RTCDataChannelMessage.fromBinary(_encRequestTransfer()));
        } catch (e) {
          ctrl.add(FileTransferProgress(
              filename: filename, bytes: 0, totalBytes: 0, error: '$e'));
          await ctrl.close();
        }
      };

      channel.onMessage = (msg) async {
        if (!msg.isBinary) return;
        final ft = _decode(msg.binary);
        if (ft.metaFilename != null || ft.metaSize != null) {
          filename = _sanitize(ft.metaFilename ?? filename);
          total = ft.metaSize ?? 0;
          // 文件名确定后再开写入流
          savePath = '$saveDir${Platform.pathSeparator}$filename';
          sink = File(savePath!).openWrite();
          ctrl.add(FileTransferProgress(
              filename: filename, bytes: 0, totalBytes: total));
        } else if (ft.data != null) {
          sink?.add(ft.data!);
          received += ft.data!.length;
          ctrl.add(FileTransferProgress(
              filename: filename, bytes: received, totalBytes: total));
        } else if (ft.end) {
          await channel.send(RTCDataChannelMessage.fromBinary(_encSuccess()));
          await sink?.flush();
          await sink?.close();
          ctrl.add(FileTransferProgress(
              filename: filename,
              bytes: received,
              totalBytes: total,
              done: true,
              savedPath: savePath));
          await channel.close();
          await ctrl.close();
        } else if (ft.errorType != null) {
          await sink?.close();
          ctrl.add(FileTransferProgress(
              filename: filename,
              bytes: received,
              totalBytes: total,
              error: 'host error ${ft.errorType}'));
          await channel.close();
          await ctrl.close();
        }
      };
    }

    run();
    return ctrl.stream;
  }

  static String _sanitize(String name) =>
      name.replaceAll(RegExp(r'[\\/:*?"<>|]'), '_');
}
