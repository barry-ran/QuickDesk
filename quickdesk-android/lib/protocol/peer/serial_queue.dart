/// serial_queue.dart - 异步任务串行队列
///
/// 信令消息必须逐条处理完（含 await）再处理下一条，防止 async 处理交叉
/// 导致状态机乱序。client_session 与 host_session 的每个对端各持一份。
library;

class SerialTaskQueue<T> {
  final Future<void> Function(T item) _process;
  final void Function(Object error)? onError;

  final List<T> _queue = [];
  bool _processing = false;

  SerialTaskQueue(this._process, {this.onError});

  void add(T item) {
    _queue.add(item);
    if (!_processing) _drain();
  }

  Future<void> _drain() async {
    if (_queue.isEmpty) {
      _processing = false;
      return;
    }
    _processing = true;
    final item = _queue.removeAt(0);
    try {
      await _process(item);
    } catch (e) {
      onError?.call(e);
    }
    await _drain();
  }

  void clear() {
    _queue.clear();
    _processing = false;
  }
}
