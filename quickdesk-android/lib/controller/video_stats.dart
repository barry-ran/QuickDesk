/// video_stats.dart - 连接性能统计采集
///
/// 对照 WebClient/js/ui/video-stats.js：周期性 getStats，从 inbound-rtp(video)、
/// candidate-pair、local/remote-candidate 报告里算出分辨率/编码/帧率/码率/RTT/
/// 丢包/路由类型，供远程页的统计叠层展示。
library;

import 'dart:async';

import 'package:flutter_webrtc/flutter_webrtc.dart';

class VideoStatsData {
  String resolution = '—';
  String codec = '—';
  String decoder = '—';
  int fps = 0;
  int bitrateKbps = 0;
  int rttMs = 0;
  double jitterMs = 0;
  int packetsLost = 0;
  int framesDropped = 0;
  int packetRate = 0;

  String routeType = '—'; // P2P (Direct) / P2P (STUN) / Relay (TURN)
  String protocol = '—';
  String localAddr = '—';
  String remoteAddr = '—';
}

class VideoStatsCollector {
  final Future<List<StatsReport>> Function() fetch;
  final void Function(VideoStatsData data) onUpdate;

  Timer? _timer;
  int _prevBytes = 0;
  int _prevFrames = 0;
  int _prevPackets = 0;
  double _prevTs = 0;

  VideoStatsCollector({required this.fetch, required this.onUpdate});

  void start() {
    if (_timer != null) return;
    _timer = Timer.periodic(const Duration(seconds: 1), (_) => _tick());
    _tick();
  }

  void stop() {
    _timer?.cancel();
    _timer = null;
    _prevBytes = 0;
    _prevFrames = 0;
    _prevPackets = 0;
    _prevTs = 0;
  }

  bool get running => _timer != null;

  Future<void> _tick() async {
    final reports = await fetch();
    if (reports.isEmpty) return;

    final byId = <String, StatsReport>{};
    for (final r in reports) {
      byId[r.id] = r;
    }

    final data = VideoStatsData();

    Map<dynamic, dynamic>? inboundVideo;
    Map<dynamic, dynamic>? candidatePair;
    double nowTs = 0;

    for (final r in reports) {
      final v = r.values;
      final kind = (v['kind'] ?? v['mediaType'])?.toString();
      if (r.type == 'inbound-rtp' && kind == 'video') {
        inboundVideo = v;
        nowTs = r.timestamp;
      }
      if (r.type == 'candidate-pair' && (v['state'] == 'succeeded')) {
        if (candidatePair == null || v['nominated'] == true) {
          candidatePair = v;
        }
      }
    }

    if (inboundVideo != null) {
      final w = _int(inboundVideo['frameWidth']);
      final h = _int(inboundVideo['frameHeight']);
      if (w > 0 && h > 0) data.resolution = '$w × $h';
      data.framesDropped = _int(inboundVideo['framesDropped']);
      data.packetsLost = _int(inboundVideo['packetsLost']);
      final jitter = _double(inboundVideo['jitter']);
      data.jitterMs = jitter * 1000;
      data.decoder = (inboundVideo['decoderImplementation'] ?? '—').toString();

      final codecId = inboundVideo['codecId']?.toString();
      if (codecId != null && byId[codecId] != null) {
        data.codec = (byId[codecId]!.values['mimeType'] ?? '—').toString();
      }

      final bytes = _int(inboundVideo['bytesReceived']);
      final frames = _int(inboundVideo['framesDecoded']);
      final packets = _int(inboundVideo['packetsReceived']);
      final dt = _prevTs > 0 ? (nowTs - _prevTs) / 1000.0 : 1.0;
      if (dt > 0 && _prevTs > 0) {
        data.fps = ((frames - _prevFrames) / dt).round();
        data.bitrateKbps = (((bytes - _prevBytes) * 8) / dt / 1000).round();
        data.packetRate = ((packets - _prevPackets) / dt).round();
      }
      _prevBytes = bytes;
      _prevFrames = frames;
      _prevPackets = packets;
      _prevTs = nowTs;
    }

    if (candidatePair != null) {
      data.rttMs = (_double(candidatePair['currentRoundTripTime']) * 1000).round();
      final localId = candidatePair['localCandidateId']?.toString();
      final remoteId = candidatePair['remoteCandidateId']?.toString();
      final local = localId != null ? byId[localId]?.values : null;
      final remote = remoteId != null ? byId[remoteId]?.values : null;
      final localType = local?['candidateType']?.toString() ?? '';
      final remoteType = remote?['candidateType']?.toString() ?? '';
      data.routeType = _routeLabel(localType, remoteType);
      data.protocol = (local?['protocol'] ?? '—').toString().toUpperCase();
      if (local != null) {
        data.localAddr = '${_addr(local)} ($localType)';
      }
      if (remote != null) {
        data.remoteAddr = '${_addr(remote)} ($remoteType)';
      }
    }

    onUpdate(data);
  }

  String _routeLabel(String localType, String remoteType) {
    if (localType == 'relay' || remoteType == 'relay') return 'Relay (TURN)';
    if (localType == 'srflx' ||
        localType == 'prflx' ||
        remoteType == 'srflx' ||
        remoteType == 'prflx') {
      return 'P2P (STUN)';
    }
    if (localType == 'host' && remoteType == 'host') return 'P2P (Direct)';
    return '—';
  }

  String _addr(Map<dynamic, dynamic> c) {
    final addr = (c['address'] ?? c['ip'])?.toString();
    final port = c['port'];
    if (addr == null || addr.isEmpty) return '—';
    return port != null ? '$addr:$port' : addr;
  }

  int _int(Object? v) {
    if (v is int) return v;
    if (v is double) return v.round();
    if (v is String) return int.tryParse(v) ?? 0;
    return 0;
  }

  double _double(Object? v) {
    if (v is double) return v;
    if (v is int) return v.toDouble();
    if (v is String) return double.tryParse(v) ?? 0;
    return 0;
  }
}
