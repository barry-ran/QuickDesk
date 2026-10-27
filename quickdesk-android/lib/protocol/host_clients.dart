/// Minimal lifecycle contract for individually disconnectable host clients.
abstract interface class HostClientConnection {
  DateTime? get connectedAt;
  bool get isConnected;
  Future<void> close({
    required bool sendTerminate,
    String terminateReason = 'success',
  });
}

class ConnectedHostClient {
  final String clientId;
  final DateTime connectedAt;
  const ConnectedHostClient({
    required this.clientId,
    required this.connectedAt,
  });
}

/// The map is rechecked after asynchronous close so a later connection with the
/// same client ID cannot be removed by an older close completion.
Future<bool> disconnectHostClient(
  Map<String, HostClientConnection> clients,
  String clientId, {
  DateTime? connectedAt,
}) async {
  final client = clients[clientId];
  if (client == null ||
      !client.isConnected ||
      (connectedAt != null && client.connectedAt != connectedAt)) {
    return false;
  }
  await client.close(sendTerminate: true);
  if (!identical(clients[clientId], client)) return false;
  clients.remove(clientId);
  return true;
}
