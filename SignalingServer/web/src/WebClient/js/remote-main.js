/**
 * remote-main.js - 远程桌面页面入口
 *
 * 从 URL 参数获取连接信息，建立远程桌面会话
 * 由 index.html 通过 window.open 打开
 */

import { Session, SessionState } from './protocol/session.js';
import { DataChannelHandler } from './protocol/datachannel-handler.js';
import { MouseHandler } from './input/mouse-handler.js';
import { KeyboardHandler } from './input/keyboard-handler.js';
import { TouchHandler } from './input/touch-handler.js';
import { ClipboardHandler } from './input/clipboard-handler.js';
import { CursorRenderer } from './ui/cursor-renderer.js';
import { MouseButton } from './protocol/protobuf-messages.js';
import { VideoStats } from './ui/video-stats.js';
import { FloatingToolbar } from './ui/floating-toolbar.js';
import { IceConfigFetcher } from './ice-config-fetcher.js';
import { IceServerStorage } from './storage/ice-server-storage.js';

class RemoteDesktopApp {
    constructor() {
        this.session = null;
        this.dcHandler = null;
        this.mouseHandler = null;
        this.keyboardHandler = null;
        this.touchHandler = null;
        this.clipboardHandler = null;
        this._isMobile = 'ontouchstart' in window || navigator.maxTouchPoints > 0;
        this.cursorRenderer = null;
        this.videoStats = null;
        this.floatingToolbar = null;

        this._receivedStreams = null;
        this._selectedStreamId = null;
        this._pendingAudioTrack = null;
        this._remoteWidth = 0;
        this._remoteHeight = 0;
    }

    async init() {
        const params = new URLSearchParams(window.location.search);
        const serverUrl = params.get('server') || 'ws://localhost:8000';
        const deviceId = params.get('device');
        const accessCode = params.get('code');
        const preferredVideoCodec = params.get('codec') || '';

        if (!deviceId || !accessCode) {
            this._log('缺少连接参数 (device, code)', 'error');
            this._setConnectionState('failed', '缺少连接参数');
            return;
        }

        document.title = `QuickDesk - ${deviceId}`;
        document.getElementById('connDevice').textContent = deviceId;

        const remoteContainer = document.getElementById('remoteContainer');
        if (remoteContainer) {
            this.floatingToolbar = new FloatingToolbar(remoteContainer);
            this.floatingToolbar.addEventListener('action', (e) => this._handleToolbarAction(e.detail));
            this.floatingToolbar.addEventListener('settingChange', (e) => this._handleSettingChange(e.detail));
            this.floatingToolbar.setVisible(false);
        }

        if (this._isMobile) {
            this._setupMobileToolbar();
        }

        this._initLogDrawer();

        const statsOverlay = document.getElementById('statsOverlay');
        if (statsOverlay) {
            this.videoStats = new VideoStats(statsOverlay, null);
        }

        this._log('正在获取 ICE 配置...');
        const fetcher = new IceConfigFetcher(serverUrl);
        const userIceServers = IceServerStorage.getAll();
        const iceServers = await fetcher.getIceServers(userIceServers);
        this._log(`ICE 配置: ${iceServers.length} 个服务器`);

        await this._connect(serverUrl, deviceId, accessCode, iceServers, preferredVideoCodec);
    }

    async _connect(serverUrl, deviceId, accessCode, iceServers, preferredVideoCodec) {
        this._log(`连接到 ${deviceId}...`);
        if (preferredVideoCodec) {
            this._log(`首选视频编码器: ${preferredVideoCodec}`);
        }

        try {
            this.session = new Session({
                signalingUrl: serverUrl,
                iceServers,
                preferredVideoCodec,
            });

            this.dcHandler = new DataChannelHandler();

            this.session.addEventListener('stateChange', (e) => {
                this._onSessionStateChange(e.detail);
            });

            this.session.addEventListener('log', (e) => {
                this._log(e.detail.message, e.detail.level);
            });

            this.session.addEventListener('track', (e) => {
                this._onTrack(e.detail);
            });

            this.session.addEventListener('datachannel', (e) => {
                this.dcHandler.handleDataChannel(e.detail.channel);
            });

            this.dcHandler.addEventListener('cursorShape', (e) => {
                if (this.cursorRenderer) this.cursorRenderer.updateCursor(e.detail);
            });

            this.dcHandler.addEventListener('videoLayout', (e) => {
                this._onVideoLayout(e.detail);
            });

            this.dcHandler.addEventListener('controlReady', () => {
                this._log('Control DataChannel 就绪');
                this._sendInitialConfig();
            });

            this.dcHandler.addEventListener('eventReady', () => {
                this._log('Event DataChannel 就绪');
            });

            this.dcHandler.addEventListener('capabilities', (e) => {
                this._log(`Host 能力: ${e.detail.capabilities || '(empty)'}`);
            });

            await this.session.connect(deviceId, accessCode);

        } catch (error) {
            this._log(`连接失败: ${error.message}`, 'error');
            this._setConnectionState('failed', error.message);
        }
    }

    _onSessionStateChange(detail) {
        const { oldState, newState } = detail;
        this._log(`状态: ${oldState} -> ${newState}`);

        switch (newState) {
            case SessionState.CONNECTED:
                this._setConnectionState('connected', '已连接');
                if (this.floatingToolbar) this.floatingToolbar.setVisible(true);
                if (this.videoStats) this.videoStats.setPeerConnection(this.session.pc);
                // Notify opener about successful connection
                if (window.opener && !window.opener.closed) {
                    try {
                        window.opener.postMessage({
                            type: 'quickdesk-connected',
                            deviceId: this.session.deviceId,
                            serverUrl: this.session.signalingUrl,
                        }, '*');
                    } catch (e) { /* cross-origin */ }
                }
                break;
            case SessionState.FAILED:
                this._setConnectionState('failed', '连接失败');
                break;
            case SessionState.CLOSED:
                this._setConnectionState('failed', '连接已关闭');
                break;
        }
    }

    _onTrack(detail) {
        const { track, streams } = detail;
        const streamId = (streams && streams.length > 0) ? streams[0].id : '';
        this._log(`收到 ${track.kind} 轨道, stream=${streamId}`);

        if (!this._receivedStreams) this._receivedStreams = new Map();

        if (track.kind === 'video' && streams && streams.length > 0) {
            this._receivedStreams.set(streams[0].id, streams[0]);
        }

        if (track.kind === 'audio') {
            this._log(`音频轨道: enabled=${track.enabled}, muted=${track.muted}, readyState=${track.readyState}`);
            this._pendingAudioTrack = track;
            return;
        }

        if (track.kind === 'video' && this._selectedStreamId && streamId !== this._selectedStreamId) {
            this._log(`忽略非主显示器视频流: ${streamId}`);
            return;
        }

        if (track.kind === 'video') {
            this._setupVideoPlayback(track, streams);
        }
    }

    _setupVideoPlayback(track, streams) {
        const video = document.getElementById('remoteVideo');
        if (!video) return;

        const combinedStream = new MediaStream();
        if (this._pendingAudioTrack && this._pendingAudioTrack.readyState === 'live') {
            combinedStream.addTrack(this._pendingAudioTrack);
            this._log('音频轨道已合并到视频流');
        }
        combinedStream.addTrack(track);
        video.srcObject = combinedStream;

        video.play().catch(e => this._log(`视频自动播放失败: ${e.message}`));

        video.onloadedmetadata = () => {
            this._log(`视频分辨率: ${video.videoWidth}x${video.videoHeight}`);
            document.getElementById('noVideo')?.style.setProperty('display', 'none');
            if (this.floatingToolbar) {
                this.floatingToolbar.setRemoteResolution(video.videoWidth, video.videoHeight);
            }
            this._setupInputHandlers();
        };
    }

    _onVideoLayout(layout) {
        if (!layout.videoTracks || layout.videoTracks.length === 0) return;

        this._log(`=== 收到 VideoLayout: ${layout.videoTracks.length} 个显示器 ===`);

        let primaryTrack = null;
        if (layout.primaryScreenId !== undefined) {
            for (let i = 0; i < layout.videoTracks.length; i++) {
                const t = layout.videoTracks[i];
                if (t.screenId === layout.primaryScreenId) {
                    primaryTrack = t;
                    break;
                }
            }
        }
        if (!primaryTrack) primaryTrack = layout.videoTracks[0];

        this._log(`选择显示器: ${primaryTrack.width}x${primaryTrack.height}`);
        this._remoteWidth = primaryTrack.width;
        this._remoteHeight = primaryTrack.height;

        const resEl = document.getElementById('connResolution');
        if (resEl) resEl.textContent = `${primaryTrack.width}x${primaryTrack.height}`;

        if (primaryTrack.mediaStreamId && !this._selectedStreamId) {
            this._selectedStreamId = primaryTrack.mediaStreamId;
            if (this._receivedStreams) {
                const targetStream = this._receivedStreams.get(this._selectedStreamId);
                if (targetStream) {
                    const video = document.getElementById('remoteVideo');
                    if (video && video.srcObject !== targetStream) {
                        video.srcObject = targetStream;
                    }
                }
            }
        }

        if (this.mouseHandler) {
            this.mouseHandler.setRemoteResolution(primaryTrack.width, primaryTrack.height);
        }
        if (this.touchHandler) {
            this.touchHandler.setRemoteResolution(primaryTrack.width, primaryTrack.height);
        }
    }

    _setupInputHandlers() {
        const video = document.getElementById('remoteVideo');
        const videoContainer = document.getElementById('videoContainer');
        if (!video || !this.dcHandler) return;

        this._cleanupInputHandlers();

        const w = this._remoteWidth || video.videoWidth || 1920;
        const h = this._remoteHeight || video.videoHeight || 1080;

        if (this._isMobile) {
            this.touchHandler = new TouchHandler(videoContainer || video, video, this.dcHandler);
            this.touchHandler.setRemoteResolution(w, h);
            this.touchHandler.enable();
        } else {
            this.mouseHandler = new MouseHandler(video, this.dcHandler);
            this.mouseHandler.setRemoteResolution(w, h);
            this.mouseHandler.enable();

            this.keyboardHandler = new KeyboardHandler(videoContainer || video, this.dcHandler);
            this.keyboardHandler.enable();
        }

        this.clipboardHandler = new ClipboardHandler(this.dcHandler);
        this.clipboardHandler.enable();

        this.cursorRenderer = new CursorRenderer(videoContainer || video);

        if (this._isMobile) {
            this._enableMobileKeyboard();
        } else {
            (videoContainer || video).focus();
        }
    }

    _cleanupInputHandlers() {
        if (this.mouseHandler) { this.mouseHandler.destroy(); this.mouseHandler = null; }
        if (this.keyboardHandler) { this.keyboardHandler.destroy(); this.keyboardHandler = null; }
        if (this.touchHandler) { this.touchHandler.destroy(); this.touchHandler = null; }
        this._mobileKbInput = null;
        if (this.clipboardHandler) { this.clipboardHandler.destroy(); this.clipboardHandler = null; }
        if (this.cursorRenderer) { this.cursorRenderer.destroy(); this.cursorRenderer = null; }
    }

    _sendInitialConfig() {
        this.dcHandler.sendCapabilities('');
        this.dcHandler.sendAudioControl({ enable: true });
        this._log('已发送初始配置');
    }

    _handleToolbarAction(detail) {
        switch (detail.action) {
            case 'disconnect':
                this._disconnect();
                break;
            case 'fitWindow': {
                const video = document.getElementById('remoteVideo');
                if (video) video.style.objectFit = 'contain';
                break;
            }
            case 'screenshot':
                this._screenshot();
                break;
            case 'toggleLogs':
                this._toggleLogDrawer();
                break;
        }
    }

    _handleSettingChange(detail) {
        if (!this.dcHandler) return;

        switch (detail.setting) {
            case 'framerate':
                this.dcHandler.sendVideoControl({ enable: true, targetFramerate: detail.value });
                this._log(`目标帧率: ${detail.value} FPS`);
                break;
            case 'framerateBoost': {
                const boostConfig = {
                    off: { enabled: false },
                    office: { enabled: true, captureIntervalMs: 30, boostDurationMs: 300 },
                    gaming: { enabled: true, captureIntervalMs: 15, boostDurationMs: 500 },
                };
                this.dcHandler.sendVideoControl({
                    framerateBoost: boostConfig[detail.value] || { enabled: false },
                });
                this._log(`帧率增强: ${detail.value}`);
                break;
            }
            case 'bitrate':
                this.dcHandler.sendPeerConnectionParameters({
                    preferredMinBitrateBps: detail.value,
                });
                this._log(`最小码率: ${Math.round(detail.value / (1024 * 1024))} MiB`);
                break;
            case 'resolution': {
                let width, height;
                if (detail.value === 'original') {
                    if (this.floatingToolbar?._originalWidth > 0) {
                        width = this.floatingToolbar._originalWidth;
                        height = this.floatingToolbar._originalHeight;
                    } else return;
                } else {
                    const parts = detail.value.split('x');
                    width = parseInt(parts[0]);
                    height = parseInt(parts[1]);
                }
                this.dcHandler.sendClientResolution({ widthPixels: width, heightPixels: height, xDpi: 96, yDpi: 96 });
                this._log(`分辨率: ${width}x${height}`);
                break;
            }
            case 'audio':
                this.dcHandler.sendAudioControl({ enable: detail.value });
                this._log(`音频: ${detail.value ? '开启' : '关闭'}`);
                break;
            case 'stats':
                if (this.videoStats) {
                    detail.value ? this.videoStats.show() : this.videoStats.hide();
                }
                break;
        }
    }

    _screenshot() {
        const video = document.getElementById('remoteVideo');
        if (!video || !video.videoWidth) {
            this._log('无视频可截图', 'warning');
            return;
        }
        const canvas = document.createElement('canvas');
        canvas.width = video.videoWidth;
        canvas.height = video.videoHeight;
        canvas.getContext('2d').drawImage(video, 0, 0);
        canvas.toBlob((blob) => {
            const url = URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = `quickdesk_screenshot_${Date.now()}.png`;
            a.click();
            URL.revokeObjectURL(url);
            this._log('截图已保存');
        }, 'image/png');
    }

    _disconnect() {
        if (this.session) {
            this.session.disconnect();
        }
        this._cleanupInputHandlers();
        this._setConnectionState('failed', '已断开');
        setTimeout(() => {
            if (this._isMobile) {
                if (window.history.length > 1) {
                    window.history.back();
                } else {
                    window.location.href = 'index.html';
                }
            } else {
                window.close();
            }
        }, 500);
    }

    _setConnectionState(state, text) {
        const dot = document.getElementById('connDot');
        const statusEl = document.getElementById('connStatus');
        if (dot) {
            dot.className = 'conn-dot';
            if (state === 'connected') dot.classList.add('connected');
            else if (state === 'failed') dot.classList.add('failed');
        }
        if (statusEl) statusEl.textContent = text;
    }

    _initLogDrawer() {
        document.getElementById('logCloseBtn')?.addEventListener('click', () => this._toggleLogDrawer(false));
        document.getElementById('logClearBtn')?.addEventListener('click', () => {
            const container = document.getElementById('logContainer');
            if (container) container.innerHTML = '';
        });
    }

    _toggleLogDrawer(forceState) {
        const drawer = document.getElementById('logDrawer');
        if (!drawer) return;
        const open = forceState !== undefined ? forceState : !drawer.classList.contains('open');
        drawer.classList.toggle('open', open);
        if (open) {
            const container = document.getElementById('logContainer');
            if (container) container.scrollTop = container.scrollHeight;
        }
    }

    _setupMobileToolbar() {
        const toolbar = document.getElementById('mobileToolbar');
        if (!toolbar) return;
        toolbar.style.display = 'block';

        const keyInput = document.getElementById('mobileKeyInput');

        document.getElementById('btnKeyboard')?.addEventListener('click', () => {
            if (!keyInput) return;
            keyInput.style.pointerEvents = 'auto';
            keyInput.focus();
            keyInput.style.pointerEvents = 'none';
        });

        document.getElementById('btnRightClick')?.addEventListener('click', () => {
            if (!this.touchHandler || !this.dcHandler) return;
            const x = Math.round(this.touchHandler._cursorX);
            const y = Math.round(this.touchHandler._cursorY);
            this.dcHandler.sendMouseEvent({ x, y, button: MouseButton.BUTTON_RIGHT, buttonDown: true });
            this.dcHandler.sendMouseEvent({ x, y, button: MouseButton.BUTTON_RIGHT, buttonDown: false });
        });

        document.getElementById('btnLogs')?.addEventListener('click', () => {
            this._toggleLogDrawer();
        });

        document.getElementById('btnZoomReset')?.addEventListener('click', () => {
            if (this.touchHandler) this.touchHandler.resetZoom();
        });

        document.getElementById('btnFullscreen')?.addEventListener('click', () => {
            if (document.fullscreenElement) {
                document.exitFullscreen();
            } else {
                document.documentElement.requestFullscreen?.();
            }
        });

        document.getElementById('btnDisconnect')?.addEventListener('click', () => {
            this._disconnect();
        });
    }

    _enableMobileKeyboard() {
        const keyInput = document.getElementById('mobileKeyInput');
        if (!keyInput || !this.dcHandler || this._mobileKbInput) return;
        this._mobileKbInput = keyInput;
        this._isComposing = false;
        this._inputTimer = null;
        this._lastSentValue = '';

        keyInput.addEventListener('compositionstart', () => {
            this._isComposing = true;
        });

        keyInput.addEventListener('compositionend', (e) => {
            this._isComposing = false;
            clearTimeout(this._inputTimer);
            keyInput.value = '';
            if (e.data) {
                this._sendTextAsKeys(e.data);
            }
        });

        keyInput.addEventListener('input', () => {
            if (this._isComposing) return;
            clearTimeout(this._inputTimer);
            this._inputTimer = setTimeout(() => {
                const text = keyInput.value;
                if (!text) return;
                keyInput.value = '';
                this._sendTextAsKeys(text);
            }, 60);
        });

        keyInput.addEventListener('keydown', (e) => {
            if (e.key === 'Backspace' || e.key === 'Enter') {
                e.preventDefault();
                const usb = e.key === 'Backspace' ? 0x07002A : 0x070028;
                this.dcHandler.sendKeyEvent({ pressed: true, usbKeycode: usb });
                this.dcHandler.sendKeyEvent({ pressed: false, usbKeycode: usb });
            }
        });

        this._setupKeyboardResize();
    }

    _sendTextAsKeys(text) {
        let asciiBuffer = '';
        for (const ch of text) {
            if (CHAR_TO_USB[ch] || CHAR_TO_USB[ch.toLowerCase()]) {
                asciiBuffer += ch;
            } else {
                if (asciiBuffer) {
                    for (const c of asciiBuffer) this._sendCharAsKey(c);
                    asciiBuffer = '';
                }
                // Non-ASCII (中文等): use TextEvent protocol
                this.dcHandler.sendTextEvent(ch);
            }
        }
        if (asciiBuffer) {
            for (const c of asciiBuffer) this._sendCharAsKey(c);
        }
    }

    _setupKeyboardResize() {
        const vv = window.visualViewport;
        if (!vv) return;
        let lastHeight = vv.height;

        vv.addEventListener('resize', () => {
            const h = vv.height;
            if (Math.abs(h - lastHeight) < 50) return;
            lastHeight = h;
            document.body.style.height = `${h}px`;
            requestAnimationFrame(() => {
                window.scrollTo(0, 0);
                if (this.touchHandler?.isZoomed) {
                    this.touchHandler._autoPanToFollow();
                }
            });
        });
    }

    _sendCharAsKey(ch) {
        const usb = CHAR_TO_USB[ch] || CHAR_TO_USB[ch.toLowerCase()];
        if (!usb) return;
        const needShift = (ch >= 'A' && ch <= 'Z') || SHIFT_CHARS.has(ch);
        const SHIFT_USB = 0x0700E1;
        if (needShift) this.dcHandler.sendKeyEvent({ pressed: true, usbKeycode: SHIFT_USB });
        this.dcHandler.sendKeyEvent({ pressed: true, usbKeycode: usb });
        this.dcHandler.sendKeyEvent({ pressed: false, usbKeycode: usb });
        if (needShift) this.dcHandler.sendKeyEvent({ pressed: false, usbKeycode: SHIFT_USB });
    }

    _log(message, level = 'info') {
        const container = document.getElementById('logContainer');
        if (container) {
            const time = new Date().toLocaleTimeString();
            const entry = document.createElement('div');
            entry.className = `log-entry log-${level}`;
            entry.innerHTML = `<span class="log-time">[${time}]</span> ${message}`;
            container.appendChild(entry);
            container.scrollTop = container.scrollHeight;
            while (container.children.length > 500) {
                container.removeChild(container.firstChild);
            }
        }
        console.log(`[RemoteDesktop] ${message}`);
    }
}

// Character → USB HID keycode mapping for mobile virtual keyboard input
const P = 0x070000;
const CHAR_TO_USB = {
    'a':P|0x04,'b':P|0x05,'c':P|0x06,'d':P|0x07,'e':P|0x08,'f':P|0x09,
    'g':P|0x0A,'h':P|0x0B,'i':P|0x0C,'j':P|0x0D,'k':P|0x0E,'l':P|0x0F,
    'm':P|0x10,'n':P|0x11,'o':P|0x12,'p':P|0x13,'q':P|0x14,'r':P|0x15,
    's':P|0x16,'t':P|0x17,'u':P|0x18,'v':P|0x19,'w':P|0x1A,'x':P|0x1B,
    'y':P|0x1C,'z':P|0x1D,
    '1':P|0x1E,'2':P|0x1F,'3':P|0x20,'4':P|0x21,'5':P|0x22,'6':P|0x23,
    '7':P|0x24,'8':P|0x25,'9':P|0x26,'0':P|0x27,
    ' ':P|0x2C,'-':P|0x2D,'=':P|0x2E,'[':P|0x2F,']':P|0x30,'\\':P|0x31,
    ';':P|0x33,"'":P|0x34,'`':P|0x35,',':P|0x36,'.':P|0x37,'/':P|0x38,
    '\t':P|0x2B,
};
// Shift symbols → base key mapping
const SHIFT_CHAR_MAP = {
    '~':'`','!':'1','@':'2','#':'3','$':'4','%':'5','^':'6','&':'7',
    '*':'8','(':'9',')':'0','_':'-','+':'=','{':'[','}':']','|':'\\',
    ':':';','"':"'",'<':',','>':'.','?':'/',
};
for (const [shifted, base] of Object.entries(SHIFT_CHAR_MAP)) {
    if (CHAR_TO_USB[base]) CHAR_TO_USB[shifted] = CHAR_TO_USB[base];
}
const SHIFT_CHARS = new Set(Object.keys(SHIFT_CHAR_MAP));

document.addEventListener('DOMContentLoaded', () => {
    const app = new RemoteDesktopApp();
    app.init();
    window.remoteDesktopApp = app;
});
