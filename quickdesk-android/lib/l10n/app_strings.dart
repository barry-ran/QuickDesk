/// app_strings.dart - 轻量 i18n（中/英），无需代码生成
///
/// 全局 `L10n.t(key, {params})` 取当前语言文案；`{name}` 占位用 params 替换。
/// 语言切换见 locale_controller.dart（切换后重建整棵 MaterialApp）。
library;

enum AppLocale { zh, en }

class L10n {
  static AppLocale locale = AppLocale.zh;

  static String t(String key, [Map<String, Object?>? params]) {
    final table = locale == AppLocale.zh ? _zh : _en;
    var s = table[key] ?? _zh[key] ?? key;
    if (params != null) {
      params.forEach((k, v) {
        s = s.replaceAll('{$k}', '${v ?? ''}');
      });
    }
    return s;
  }

  static const Map<String, String> _zh = {
    // 通用
    'app.name': 'QuickDesk',
    'common.back': '返回',
    'common.cancel': '取消',
    'common.copy': '复制',
    'common.copied': '已复制 {label}',

    // 首页 / 模式
    'home.controller': '主控',
    'home.host': '被控',
    'home.titleController': 'QuickDesk · 主控',
    'home.titleHost': 'QuickDesk · 被控',
    'home.language': '语言',

    // 连接页
    'connect.title': '连接远程设备',
    'connect.deviceId': '设备 ID',
    'connect.deviceIdHint': '9 位数字',
    'connect.accessCode': '访问码',
    'connect.connect': '连接',
    'connect.needIdAndCode': '请输入设备 ID 和访问码',
    'connect.serverSettings': '服务器设置',
    'connect.serverUrl': '信令服务器地址',
    'connect.apiKey': 'API Key（可选）',
    'connect.recentAndFav': '最近与收藏',
    'connect.favorite': '收藏',
    'connect.unfavorite': '取消收藏',
    'connect.errDeviceNotFound': '设备不存在',
    'connect.errHostOffline': '对方设备离线',
    'connect.errInvalidCode': '访问码错误',
    'connect.errTooManyAttempts': '尝试次数过多，请稍后再试',
    'connect.errGeneric': '连接失败: {code}',

    // 远程页
    'remote.connecting': '连接中...',
    'remote.stateConnecting': '连接信令服务器...',
    'remote.stateInitiating': '发起会话...',
    'remote.stateAccepting': '等待对方接受...',
    'remote.stateAuthenticating': '认证中...',
    'remote.stateFailed': '连接失败',
    'remote.stateClosed': '连接已断开',
    'remote.waitingFrame': '已连接，等待远程画面…',
    'remote.switchDisplay': '切换显示器',
    'remote.display': '显示器 {n}',
    'remote.rotate': '切换横竖屏',
    'remote.resetZoom': '重置缩放',
    'remote.leftHold': '左键按住中 · 再点松开',
    'remote.keyboard': '键盘',
    'remote.sendClipboard': '把本机剪贴板发送到远端',
    'remote.clipboardSent': '已发送剪贴板',
    'remote.stats': '性能统计',
    'remote.disconnect': '断开连接',
    'remote.textInputHint': '输入文字发送到远端...',
    // 统计
    'stats.title': '连接统计',
    'stats.resolution': '分辨率',
    'stats.codec': '编码',
    'stats.fps': '帧率',
    'stats.bitrate': '码率',
    'stats.rtt': '延迟',
    'stats.jitter': '抖动',
    'stats.packetsLost': '丢包',
    'stats.framesDropped': '丢帧',
    'stats.decoder': '解码器',
    'stats.route': '线路',
    'stats.protocol': '协议',
    'stats.localAddr': '本地地址',
    'stats.remoteAddr': '远端地址',
    // 文件传输
    'file.transfer': '文件传输',
    'file.upload': '发送文件到远端',
    'file.download': '从远端下载文件',
    'file.uploading': '正在上传',
    'file.downloading': '正在下载',
    'file.done': '完成',
    'file.failed': '传输失败',
    'file.savedTo': '已保存到 {path}',

    // 被控页
    'host.statusTitle': '被控状态',
    'host.controlling': '被控中（{n} 个连接）',
    'host.online': '在线待命',
    'host.offline': '未连接',
    'host.notStarted': '未开启',
    'host.preparing': '准备中',
    'host.deviceId': '设备 ID',
    'host.accessCode': '访问码',
    'host.accessCodeUnset': '未设置',
    'host.regenCode': '重新生成访问码',
    'host.start': '开启被控',
    'host.stop': '停止被控',
    'host.hint': '开启后，其他设备可用上方「设备 ID + 访问码」控制本机。'
        '被控依赖无障碍服务注入点击/滑动，屏幕采集需系统授权。',
    'host.inputMethod': '输入方式：{label}',
    'host.backendShizuku': '增强档（Shizuku）',
    'host.backendShizukuDesc': '真实触摸事件，支持流畅拖拽与全键盘',
    'host.backendA11y': '标准档（无障碍）',
    'host.backendA11yDesc': '通过无障碍手势注入点击/滑动',
    'host.enableShizuku': '启用 Shizuku 增强',
    'host.shizukuHint': '提示：安装并启动 Shizuku 后可解锁增强档（更流畅、支持完整键盘）。',
    'host.a11yTitle': '需要开启无障碍服务',
    'host.a11yDesc': '被控时需要「QuickDesk 被控输入」无障碍服务来执行远程点击与滑动。'
        '点击下方按钮前往系统设置开启，然后重新点「开启被控」。',
    'host.a11yGo': '前往无障碍设置',
    // 被控端 controller 状态消息
    'host.msgCheckA11y': '检查无障碍权限',
    'host.msgNeedA11y': '请先开启 QuickDesk 无障碍服务',
    'host.msgProvisioning': '注册设备中',
    'host.msgRequestCapture': '请授权屏幕采集',
    'host.msgConnecting': '连接信令服务器',
    'host.msgOnline': '已上线，等待连接',
    'host.msgSignalDropped': '信令连接断开',
    'host.msgServerError': '服务器错误: {code}',
    'host.msgStartFailed': '启动失败: {e}',
  };

  static const Map<String, String> _en = {
    'app.name': 'QuickDesk',
    'common.back': 'Back',
    'common.cancel': 'Cancel',
    'common.copy': 'Copy',
    'common.copied': 'Copied {label}',

    'home.controller': 'Control',
    'home.host': 'Share',
    'home.titleController': 'QuickDesk · Control',
    'home.titleHost': 'QuickDesk · Share',
    'home.language': 'Language',

    'connect.title': 'Connect to a device',
    'connect.deviceId': 'Device ID',
    'connect.deviceIdHint': '9 digits',
    'connect.accessCode': 'Access code',
    'connect.connect': 'Connect',
    'connect.needIdAndCode': 'Enter device ID and access code',
    'connect.serverSettings': 'Server settings',
    'connect.serverUrl': 'Signaling server URL',
    'connect.apiKey': 'API Key (optional)',
    'connect.recentAndFav': 'Recent & favorites',
    'connect.favorite': 'Favorite',
    'connect.unfavorite': 'Unfavorite',
    'connect.errDeviceNotFound': 'Device not found',
    'connect.errHostOffline': 'The device is offline',
    'connect.errInvalidCode': 'Wrong access code',
    'connect.errTooManyAttempts': 'Too many attempts, try again later',
    'connect.errGeneric': 'Connection failed: {code}',

    'remote.connecting': 'Connecting...',
    'remote.stateConnecting': 'Connecting to signaling...',
    'remote.stateInitiating': 'Starting session...',
    'remote.stateAccepting': 'Waiting for the host...',
    'remote.stateAuthenticating': 'Authenticating...',
    'remote.stateFailed': 'Connection failed',
    'remote.stateClosed': 'Disconnected',
    'remote.waitingFrame': 'Connected, waiting for remote video…',
    'remote.switchDisplay': 'Switch display',
    'remote.display': 'Display {n}',
    'remote.rotate': 'Toggle orientation',
    'remote.resetZoom': 'Reset zoom',
    'remote.leftHold': 'Left button held · tap to release',
    'remote.keyboard': 'Keyboard',
    'remote.sendClipboard': 'Send local clipboard to remote',
    'remote.clipboardSent': 'Clipboard sent',
    'remote.stats': 'Statistics',
    'remote.disconnect': 'Disconnect',
    'remote.textInputHint': 'Type to send to remote...',
    'stats.title': 'Connection stats',
    'stats.resolution': 'Resolution',
    'stats.codec': 'Codec',
    'stats.fps': 'FPS',
    'stats.bitrate': 'Bitrate',
    'stats.rtt': 'Latency',
    'stats.jitter': 'Jitter',
    'stats.packetsLost': 'Packets lost',
    'stats.framesDropped': 'Frames dropped',
    'stats.decoder': 'Decoder',
    'stats.route': 'Route',
    'stats.protocol': 'Protocol',
    'stats.localAddr': 'Local address',
    'stats.remoteAddr': 'Remote address',
    'file.transfer': 'File transfer',
    'file.upload': 'Send file to remote',
    'file.download': 'Download file from remote',
    'file.uploading': 'Uploading',
    'file.downloading': 'Downloading',
    'file.done': 'Done',
    'file.failed': 'Transfer failed',
    'file.savedTo': 'Saved to {path}',

    'host.statusTitle': 'Share status',
    'host.controlling': 'Controlled ({n} peer(s))',
    'host.online': 'Online, idle',
    'host.offline': 'Not connected',
    'host.notStarted': 'Off',
    'host.preparing': 'Preparing',
    'host.deviceId': 'Device ID',
    'host.accessCode': 'Access code',
    'host.accessCodeUnset': 'Not set',
    'host.regenCode': 'Regenerate access code',
    'host.start': 'Start sharing',
    'host.stop': 'Stop sharing',
    'host.hint': 'Once started, other devices can control this phone with the '
        'Device ID + access code above. Input uses the accessibility service; '
        'screen capture requires system permission.',
    'host.inputMethod': 'Input: {label}',
    'host.backendShizuku': 'Enhanced (Shizuku)',
    'host.backendShizukuDesc': 'Real touch events, smooth drag and full keyboard',
    'host.backendA11y': 'Standard (Accessibility)',
    'host.backendA11yDesc': 'Injects taps/swipes via accessibility gestures',
    'host.enableShizuku': 'Enable Shizuku',
    'host.shizukuHint': 'Tip: install & start Shizuku to unlock the enhanced backend '
        '(smoother, full keyboard).',
    'host.a11yTitle': 'Accessibility service required',
    'host.a11yDesc': 'Sharing needs the "QuickDesk input" accessibility service to '
        'perform remote taps and swipes. Tap below to enable it in system settings, '
        'then tap "Start sharing" again.',
    'host.a11yGo': 'Open accessibility settings',
    'host.msgCheckA11y': 'Checking accessibility permission',
    'host.msgNeedA11y': 'Please enable the QuickDesk accessibility service first',
    'host.msgProvisioning': 'Registering device',
    'host.msgRequestCapture': 'Please allow screen capture',
    'host.msgConnecting': 'Connecting to signaling server',
    'host.msgOnline': 'Online, waiting for connections',
    'host.msgSignalDropped': 'Signaling connection dropped',
    'host.msgServerError': 'Server error: {code}',
    'host.msgStartFailed': 'Start failed: {e}',
  };
}
