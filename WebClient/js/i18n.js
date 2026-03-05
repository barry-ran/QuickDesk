/**
 * i18n.js - Lightweight internationalization module
 *
 * Supports zh-CN and en-US with auto-detection and localStorage persistence.
 */

const STORAGE_KEY = 'quickdesk_lang';

const zh = {
    // ========== index.html ==========
    'app.title': 'QuickDesk Web Client',
    'nav.remote': '远程控制',
    'nav.settings': '设置',
    'nav.about': '关于',
    'version.label': 'Web Client v1.0.0',

    // Remote page
    'remote.title': '远程控制',
    'connect.title': '连接远程设备',
    'connect.server': '信令服务器',
    'connect.deviceId': '设备 ID',
    'connect.deviceId.placeholder': '请输入远程设备 ID',
    'connect.accessCode': '访问码',
    'connect.accessCode.placeholder': '请输入访问码',
    'connect.button': '连接',
    'connect.inputRequired': '请输入设备 ID 和访问码',
    'connect.connecting': '正在连接 {deviceId}...',

    // History
    'history.title': '历史连接',
    'history.empty': '暂无历史连接记录',
    'history.deleted': '已删除历史记录',
    'history.connectCount': '连接 {count} 次',
    'history.fill': '填充',
    'history.delete': '删除',

    // Time
    'time.justNow': '刚刚',
    'time.minutesAgo': '{n} 分钟前',
    'time.hoursAgo': '{n} 小时前',
    'time.daysAgo': '{n} 天前',

    // Settings
    'settings.title': '设置',
    'settings.network': '网络设置',
    'settings.signalingServer': '信令服务器',
    'settings.signalingServerDesc': '设置默认的信令服务器地址，用于 WebRTC 连接协商。',
    'settings.signalingServer.placeholder': 'ws://qdsignaling.quickdesk.cc:8000',
    'settings.signalingServerSaved': '信令服务器地址已保存',
    'settings.video': '视频设置',
    'settings.videoCodec': '首选视频编码器',
    'settings.videoCodecDesc': '选择远程连接使用的视频编码器，实际使用取决于主机端支持情况。',
    'settings.codecChanged': '视频编码器已设置为 {codec}，下次连接时生效',

    // About
    'about.title': '关于',
    'about.desc': '首款 AI 原生远程桌面。内置 MCP Server，让任何 AI Agent 都能查看和操控远程电脑。底层基于 Chromium Remoting 技术，经过 Google 十余年打磨，性能、安全性和稳定性达到工业级水准。',
    'about.links': '链接',
    'about.sourceCode': '源代码',
    'about.basedOn': '基于 Chromium Remoting',
    'about.license': '开源协议',
    'about.licenseDesc': 'QuickDesk 采用 MIT License 开源协议。其中 quickdesk-remoting 组件基于 Chromium，采用 BSD 3-Clause License 开源协议。',

    // ========== remote.html ==========
    'remote.connecting': '连接中...',
    'remote.waitingVideo': '等待视频流...',
    'remote.clickToUnmute': '点击任意位置开启声音',
    'remote.clickToPlay': '点击任意位置开始播放',
    'remote.logs': 'Logs',
    'remote.clear': 'Clear',

    // Mobile toolbar
    'mobile.keyboard': 'Keyboard',
    'mobile.rightClick': 'Right Click',
    'mobile.logs': 'Logs',
    'mobile.zoom': 'Zoom',
    'mobile.fullscreen': 'Fullscreen',
    'mobile.close': 'Close',

    // ========== Floating toolbar ==========
    'menu.smartBoost': 'Smart Boost',
    'menu.targetFramerate': 'Target Framerate',
    'menu.resolution': 'Resolution',
    'menu.bitrate': 'Bitrate',
    'menu.fitWindow': 'Fit Window',
    'menu.videoStats': 'Video Stats',
    'menu.muteAudio': 'Mute Audio',
    'menu.unmuteAudio': 'Unmute Audio',
    'menu.screenshot': 'Screenshot',
    'menu.logs': 'Logs',
    'menu.sendCAD': 'Send Ctrl+Alt+Del',
    'menu.lockScreen': 'Lock Screen',
    'menu.uploadFile': 'Upload File',
    'menu.downloadFile': 'Download from Host',
    'menu.transfers': 'Transfers',
    'menu.transfersCount': 'Transfers ({count})',
    'menu.disconnect': 'Disconnect',
    'menu.original': 'Original',
    'menu.off': 'Off',
    'menu.office': 'Office',
    'menu.gaming': 'Gaming',

    // ========== remote-main.js log/status ==========
    'log.missingParams': '缺少连接参数 (device, code)',
    'status.missingParams': '缺少连接参数',
    'log.fetchingIce': '正在获取 ICE 配置...',
    'log.iceServers': 'ICE 配置: {count} 个服务器',
    'log.connectingTo': '连接到 {deviceId}...',
    'log.preferredCodec': '首选视频编码器: {codec}',
    'log.controlReady': 'Control DataChannel 就绪',
    'log.eventReady': 'Event DataChannel 就绪',
    'log.hostCaps': 'Host 能力: {caps}',
    'log.negotiatedCaps': 'Negotiated caps: SAS={sas} Lock={lock} FileTransfer={ft}',
    'log.uploadComplete': 'Upload complete: {filename}',
    'log.uploadFailed': 'Upload failed: {error}',
    'log.downloadStarted': 'Download started: {filename}',
    'log.downloadComplete': 'Download complete: {filename}',
    'log.downloadFailed': 'Download failed: {error}',
    'log.connectFailed': '连接失败: {error}',
    'status.connectFailed': '连接失败',
    'log.stateChange': '状态: {from} -> {to}',
    'status.connected': '已连接',
    'status.failed': '连接失败',
    'status.closed': '连接已关闭',
    'status.disconnected': '已断开',
    'log.trackReceived': '收到 {kind} 轨道, stream={stream}',
    'log.audioTrack': '音频轨道: enabled={enabled}, muted={muted}, readyState={state}',
    'log.ignoreStream': '忽略非主显示器视频流: {stream}',
    'log.audioMerged': '音频轨道已合并到视频流',
    'log.autoplayFailed': '视频自动播放失败: {error}',
    'log.videoResolution': '视频分辨率: {width}x{height}',
    'log.videoLayout': '=== 收到 VideoLayout: {count} 个显示器 ===',
    'log.selectMonitor': '选择显示器: {width}x{height}',
    'log.sentConfig': '已发送初始配置',
    'log.sentCAD': 'Sent Ctrl+Alt+Del',
    'log.sentLock': 'Sent Lock Screen',
    'log.uploadingFile': 'Uploading file: {name} ({size} bytes)',
    'log.downloadCancelled': 'Download cancelled by user',
    'log.savePickerError': 'Save picker error: {error}',
    'log.downloadStreaming': 'Requested file download (streaming to disk)',
    'log.downloadBuffered': 'Requested file download (buffered)',
    'log.targetFps': '目标帧率: {fps} FPS',
    'log.boostMode': '帧率增强: {mode}',
    'log.minBitrate': '最小码率: {bitrate} MiB',
    'log.resolution': '分辨率: {width}x{height}',
    'log.audioToggle': '音频: {state}',
    'log.audioOn': '开启',
    'log.audioOff': '关闭',
    'log.noVideoScreenshot': '无视频可截图',
    'log.screenshotSaved': '截图已保存',
    'drop.text': 'Drop files here to upload',
    'overlay.clickToUnmute': 'Click to enable audio',
    'overlay.clickToPlay': 'Click to start video',

    // ========== connection-ui.js ==========
    'connui.inputRequired': '请输入设备 ID 和访问码',
    'connui.connecting': '连接中...',
    'connui.connected': '已连接',
    'connui.disconnected': '未连接',
    'connui.failed': '连接失败',

    // ========== video-stats.js ==========
    'stats.rtt': 'RTT',
    'stats.fps': 'FPS',
    'stats.bitrate': 'Bitrate',
    'stats.resolution': 'Resolution',
    'stats.codec': 'Codec',
    'stats.decoder': 'Decoder',
    'stats.jitter': 'Jitter',
    'stats.packetsLost': 'Packets Lost',

    // Language selector
    'lang.label': '语言',
};

const en = {
    // ========== index.html ==========
    'app.title': 'QuickDesk Web Client',
    'nav.remote': 'Remote',
    'nav.settings': 'Settings',
    'nav.about': 'About',
    'version.label': 'Web Client v1.0.0',

    // Remote page
    'remote.title': 'Remote Control',
    'connect.title': 'Connect to Remote Device',
    'connect.server': 'Signaling Server',
    'connect.deviceId': 'Device ID',
    'connect.deviceId.placeholder': 'Enter remote device ID',
    'connect.accessCode': 'Access Code',
    'connect.accessCode.placeholder': 'Enter access code',
    'connect.button': 'Connect',
    'connect.inputRequired': 'Please enter Device ID and Access Code',
    'connect.connecting': 'Connecting to {deviceId}...',

    // History
    'history.title': 'Connection History',
    'history.empty': 'No connection history',
    'history.deleted': 'History record deleted',
    'history.connectCount': '{count} connections',
    'history.fill': 'Fill',
    'history.delete': 'Delete',

    // Time
    'time.justNow': 'just now',
    'time.minutesAgo': '{n} min ago',
    'time.hoursAgo': '{n} hr ago',
    'time.daysAgo': '{n} days ago',

    // Settings
    'settings.title': 'Settings',
    'settings.network': 'Network Settings',
    'settings.signalingServer': 'Signaling Server',
    'settings.signalingServerDesc': 'Set the default signaling server address for WebRTC connection negotiation.',
    'settings.signalingServer.placeholder': 'ws://qdsignaling.quickdesk.cc:8000',
    'settings.signalingServerSaved': 'Signaling server address saved',
    'settings.video': 'Video Settings',
    'settings.videoCodec': 'Preferred Video Codec',
    'settings.videoCodecDesc': 'Choose the video codec for remote connections. Actual codec depends on host support.',
    'settings.codecChanged': 'Video codec set to {codec}, effective on next connection',

    // About
    'about.title': 'About',
    'about.desc': 'The first AI-native remote desktop. Built-in MCP Server lets any AI Agent view and control remote computers. Powered by Chromium Remoting technology, refined by Google for over a decade, delivering industrial-grade performance, security, and stability.',
    'about.links': 'Links',
    'about.sourceCode': 'Source Code',
    'about.basedOn': 'Based on Chromium Remoting',
    'about.license': 'License',
    'about.licenseDesc': 'QuickDesk is open-sourced under the MIT License. The quickdesk-remoting component is based on Chromium and licensed under the BSD 3-Clause License.',

    // ========== remote.html ==========
    'remote.connecting': 'Connecting...',
    'remote.waitingVideo': 'Waiting for video...',
    'remote.clickToUnmute': 'Click to enable audio',
    'remote.clickToPlay': 'Click to start video',
    'remote.logs': 'Logs',
    'remote.clear': 'Clear',

    // Mobile toolbar
    'mobile.keyboard': 'Keyboard',
    'mobile.rightClick': 'Right Click',
    'mobile.logs': 'Logs',
    'mobile.zoom': 'Zoom',
    'mobile.fullscreen': 'Fullscreen',
    'mobile.close': 'Close',

    // ========== Floating toolbar ==========
    'menu.smartBoost': 'Smart Boost',
    'menu.targetFramerate': 'Target Framerate',
    'menu.resolution': 'Resolution',
    'menu.bitrate': 'Bitrate',
    'menu.fitWindow': 'Fit Window',
    'menu.videoStats': 'Video Stats',
    'menu.muteAudio': 'Mute Audio',
    'menu.unmuteAudio': 'Unmute Audio',
    'menu.screenshot': 'Screenshot',
    'menu.logs': 'Logs',
    'menu.sendCAD': 'Send Ctrl+Alt+Del',
    'menu.lockScreen': 'Lock Screen',
    'menu.uploadFile': 'Upload File',
    'menu.downloadFile': 'Download from Host',
    'menu.transfers': 'Transfers',
    'menu.transfersCount': 'Transfers ({count})',
    'menu.disconnect': 'Disconnect',
    'menu.original': 'Original',
    'menu.off': 'Off',
    'menu.office': 'Office',
    'menu.gaming': 'Gaming',

    // ========== remote-main.js log/status ==========
    'log.missingParams': 'Missing connection parameters (device, code)',
    'status.missingParams': 'Missing parameters',
    'log.fetchingIce': 'Fetching ICE configuration...',
    'log.iceServers': 'ICE config: {count} server(s)',
    'log.connectingTo': 'Connecting to {deviceId}...',
    'log.preferredCodec': 'Preferred video codec: {codec}',
    'log.controlReady': 'Control DataChannel ready',
    'log.eventReady': 'Event DataChannel ready',
    'log.hostCaps': 'Host capabilities: {caps}',
    'log.negotiatedCaps': 'Negotiated caps: SAS={sas} Lock={lock} FileTransfer={ft}',
    'log.uploadComplete': 'Upload complete: {filename}',
    'log.uploadFailed': 'Upload failed: {error}',
    'log.downloadStarted': 'Download started: {filename}',
    'log.downloadComplete': 'Download complete: {filename}',
    'log.downloadFailed': 'Download failed: {error}',
    'log.connectFailed': 'Connection failed: {error}',
    'status.connectFailed': 'Connection failed',
    'log.stateChange': 'State: {from} -> {to}',
    'status.connected': 'Connected',
    'status.failed': 'Connection failed',
    'status.closed': 'Connection closed',
    'status.disconnected': 'Disconnected',
    'log.trackReceived': 'Received {kind} track, stream={stream}',
    'log.audioTrack': 'Audio track: enabled={enabled}, muted={muted}, readyState={state}',
    'log.ignoreStream': 'Ignoring non-primary video stream: {stream}',
    'log.audioMerged': 'Audio track merged to video stream',
    'log.autoplayFailed': 'Video autoplay failed: {error}',
    'log.videoResolution': 'Video resolution: {width}x{height}',
    'log.videoLayout': '=== Received VideoLayout: {count} monitor(s) ===',
    'log.selectMonitor': 'Selected monitor: {width}x{height}',
    'log.sentConfig': 'Initial config sent',
    'log.sentCAD': 'Sent Ctrl+Alt+Del',
    'log.sentLock': 'Sent Lock Screen',
    'log.uploadingFile': 'Uploading file: {name} ({size} bytes)',
    'log.downloadCancelled': 'Download cancelled by user',
    'log.savePickerError': 'Save picker error: {error}',
    'log.downloadStreaming': 'Requested file download (streaming to disk)',
    'log.downloadBuffered': 'Requested file download (buffered)',
    'log.targetFps': 'Target framerate: {fps} FPS',
    'log.boostMode': 'Framerate boost: {mode}',
    'log.minBitrate': 'Min bitrate: {bitrate} MiB',
    'log.resolution': 'Resolution: {width}x{height}',
    'log.audioToggle': 'Audio: {state}',
    'log.audioOn': 'on',
    'log.audioOff': 'off',
    'log.noVideoScreenshot': 'No video to capture',
    'log.screenshotSaved': 'Screenshot saved',
    'drop.text': 'Drop files here to upload',
    'overlay.clickToUnmute': 'Click to enable audio',
    'overlay.clickToPlay': 'Click to start video',

    // ========== connection-ui.js ==========
    'connui.inputRequired': 'Please enter Device ID and Access Code',
    'connui.connecting': 'Connecting...',
    'connui.connected': 'Connected',
    'connui.disconnected': 'Not connected',
    'connui.failed': 'Connection failed',

    // ========== video-stats.js ==========
    'stats.rtt': 'RTT',
    'stats.fps': 'FPS',
    'stats.bitrate': 'Bitrate',
    'stats.resolution': 'Resolution',
    'stats.codec': 'Codec',
    'stats.decoder': 'Decoder',
    'stats.jitter': 'Jitter',
    'stats.packetsLost': 'Packets Lost',

    // Language selector
    'lang.label': 'Language',
};

const dictionaries = { 'zh-CN': zh, 'en-US': en };

let currentLocale = null;

function detectLocale() {
    const saved = localStorage.getItem(STORAGE_KEY);
    if (saved && dictionaries[saved]) return saved;
    return navigator.language.startsWith('zh') ? 'zh-CN' : 'en-US';
}

function ensureLocale() {
    if (!currentLocale) {
        currentLocale = detectLocale();
    }
}

/**
 * Translate a key with optional parameter substitution.
 * Parameters are specified as {name} in the dictionary string.
 *
 * @param {string} key
 * @param {Object} [params]
 * @returns {string}
 */
export function t(key, params) {
    ensureLocale();
    const dict = dictionaries[currentLocale] || en;
    let text = dict[key] ?? en[key] ?? key;
    if (params) {
        for (const [k, v] of Object.entries(params)) {
            text = text.replaceAll(`{${k}}`, v);
        }
    }
    return text;
}

/**
 * Set the active locale and persist to localStorage.
 * @param {string} locale - 'zh-CN' or 'en-US'
 */
export function setLocale(locale) {
    if (!dictionaries[locale]) return;
    currentLocale = locale;
    localStorage.setItem(STORAGE_KEY, locale);
}

/**
 * Get the currently active locale.
 * @returns {string}
 */
export function getLocale() {
    ensureLocale();
    return currentLocale;
}

/**
 * @returns {string[]}
 */
export function getSupportedLocales() {
    return ['zh-CN', 'en-US'];
}

/**
 * Walk all elements with [data-i18n] and replace textContent.
 * Elements with [data-i18n-placeholder] get their placeholder replaced.
 * Elements with [data-i18n-title] get their title replaced.
 */
export function applyI18n() {
    ensureLocale();
    document.querySelectorAll('[data-i18n]').forEach(el => {
        const key = el.getAttribute('data-i18n');
        if (key) el.textContent = t(key);
    });
    document.querySelectorAll('[data-i18n-placeholder]').forEach(el => {
        const key = el.getAttribute('data-i18n-placeholder');
        if (key) el.placeholder = t(key);
    });
    document.querySelectorAll('[data-i18n-title]').forEach(el => {
        const key = el.getAttribute('data-i18n-title');
        if (key) el.title = t(key);
    });
}
