/// connection_store.dart - 主控端连接历史 / 收藏（本地）
///
/// M1/M2 未接入用户登录（服务器 /v1/me/* 需账号），故历史与收藏先落本地
/// shared_preferences。登录能力落地后可迁移到服务器同步。
library;

import 'dart:convert';

import 'package:shared_preferences/shared_preferences.dart';

class ConnectionEntry {
  final String deviceId;
  String name;
  bool favorite;
  int lastConnectedAt; // epoch ms，0 表示仅收藏未连过

  ConnectionEntry({
    required this.deviceId,
    this.name = '',
    this.favorite = false,
    this.lastConnectedAt = 0,
  });

  Map<String, dynamic> toJson() => {
        'deviceId': deviceId,
        'name': name,
        'favorite': favorite,
        'lastConnectedAt': lastConnectedAt,
      };

  factory ConnectionEntry.fromJson(Map<String, dynamic> j) => ConnectionEntry(
        deviceId: (j['deviceId'] ?? '').toString(),
        name: (j['name'] ?? '').toString(),
        favorite: j['favorite'] == true,
        lastConnectedAt: (j['lastConnectedAt'] as num?)?.toInt() ?? 0,
      );
}

class ConnectionStore {
  static const _key = 'connection_history';
  static const _maxHistory = 20;

  Future<List<ConnectionEntry>> load() async {
    final prefs = await SharedPreferences.getInstance();
    final raw = prefs.getString(_key);
    if (raw == null || raw.isEmpty) return [];
    try {
      final list = jsonDecode(raw) as List<dynamic>;
      return list
          .map((e) => ConnectionEntry.fromJson(e as Map<String, dynamic>))
          .toList();
    } catch (_) {
      return [];
    }
  }

  Future<void> _save(List<ConnectionEntry> entries) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(_key, jsonEncode(entries.map((e) => e.toJson()).toList()));
  }

  /// 记录一次成功连接：更新时间戳，置顶；保留收藏标记
  Future<void> recordConnection(String deviceId, {String name = ''}) async {
    final entries = await load();
    final existing = entries.where((e) => e.deviceId == deviceId).firstOrNull;
    if (existing != null) {
      existing.lastConnectedAt = DateTime.now().millisecondsSinceEpoch;
      if (name.isNotEmpty) existing.name = name;
    } else {
      entries.add(ConnectionEntry(
        deviceId: deviceId,
        name: name,
        lastConnectedAt: DateTime.now().millisecondsSinceEpoch,
      ));
    }
    _trimNonFavorites(entries);
    await _save(entries);
  }

  Future<void> toggleFavorite(String deviceId) async {
    final entries = await load();
    final existing = entries.where((e) => e.deviceId == deviceId).firstOrNull;
    if (existing != null) {
      existing.favorite = !existing.favorite;
    } else {
      entries.add(ConnectionEntry(deviceId: deviceId, favorite: true));
    }
    await _save(entries);
  }

  Future<void> rename(String deviceId, String name) async {
    final entries = await load();
    final existing = entries.where((e) => e.deviceId == deviceId).firstOrNull;
    if (existing != null) {
      existing.name = name;
      await _save(entries);
    }
  }

  Future<void> remove(String deviceId) async {
    final entries = await load();
    entries.removeWhere((e) => e.deviceId == deviceId);
    await _save(entries);
  }

  /// 收藏优先、其次按最近连接时间排序
  Future<List<ConnectionEntry>> loadSorted() async {
    final entries = await load();
    entries.sort((a, b) {
      if (a.favorite != b.favorite) return a.favorite ? -1 : 1;
      return b.lastConnectedAt.compareTo(a.lastConnectedAt);
    });
    return entries;
  }

  void _trimNonFavorites(List<ConnectionEntry> entries) {
    final nonFav = entries.where((e) => !e.favorite).toList()
      ..sort((a, b) => b.lastConnectedAt.compareTo(a.lastConnectedAt));
    if (nonFav.length <= _maxHistory) return;
    for (final e in nonFav.skip(_maxHistory)) {
      entries.remove(e);
    }
  }
}

extension _FirstOrNull<E> on Iterable<E> {
  E? get firstOrNull {
    final it = iterator;
    return it.moveNext() ? it.current : null;
  }
}
