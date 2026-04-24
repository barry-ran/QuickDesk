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
import { t, applyI18n, getLocale, setLocale } from './i18n.js';
import { userApi } from './api/user-api.js';

class RemoteDesktopApp {
    constructor() {
        this.session = null;
        this.dcHandler = null;
        this.mouseHandler = null;
        this.keyboardHandler = null;
        this.touchHandler = null;
        this.clipboardHandler = null;
        this._isMobile = this._detectMobile();
        this._connectStartTime = 0;
        this._deviceId = '';
        this._connectionRecorded = false;
        this.cursorRenderer = null;
        this.videoStats = null;
        this.floatingToolbar = null;

        this._receivedStreams = null;
        this._selectedStreamId = null;
        this._pendingAudioTrack = null;
        this._remoteWidth = 0;
        this._remoteHeight = 0;
        this._currentDisplayList = [];
        this._activeDisplayIndex = 0;
        this._supportsVirtualDisplay = false;
    }

    /**
     * 检测是否为移动设备（手机/平板）
     * 仅靠 touch 能力检测不可靠，Windows 桌面也可能报 maxTouchPoints > 0
     * 需要结合 UA、屏幕尺寸和指针能力综合判断
     */
    _detectMobile() {
        const ua = navigator.userAgent || '';
        // Check UA for common mobile identifiers
        if (/Android|iPhone|iPad|iPod/i.test(ua)) {
            // iPad with desktop UA: navigator.platform or maxTouchPoints
            return true;
        }
        // iPad on iOS 13+ uses desktop UA but has touch
        if (navigator.platform === 'MacIntel' && navigator.maxTouchPoints > 1) {
            return true;
        }
        // Primary pointer is coarse (finger) and no fine pointer (mouse) available
        if (window.matchMedia) {
            const coarse = window.matchMedia('(pointer: coarse)').matches;
            const fine = window.matchMedia('(any-pointer: fine)').matches;
            if (coarse && !fine) {
                return true;
            }
        }
        // Small screen with touch capability is likely a phone
        if (navigator.maxTouchPoints > 0 && window.innerWidth <= 768) {
            return true;
        }
        return false;
    }

    async init() {
        this._initRemoteLangSelector();
        applyI18n();

        const params = new URLSearchParams(window.location.search);
        const serverUrl = params.get('server') || 'ws://localhost:8000';
        const deviceId = params.get('device');
        const accessCode = params.get('code');
        const preferredVideoCodec = params.get('codec') || '';

        if (!deviceId || !accessCode) {
            this._log(t('log.missingParams'), 'error');
            this._setConnectionState('failed', t('status.missingParams'));
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

        this._log(t('log.fetchingIce'));
        const fetcher = new IceConfigFetcher(serverUrl);
        const iceServers = await fetcher.getIceServers();
        this._log(t('log.iceServers', { count: iceServers.length }));

        await this._connect(serverUrl, deviceId, accessCode, iceServers, preferredVideoCodec);
    }

    _initRemoteLangSelector() {
        const select = document.getElementById('remoteLangSelect');
        if (!select) return;
        select.value = getLocale();
        select.addEventListener('change', () => {
            setLocale(select.value);
            applyI18n();
            if (this.floatingToolbar) {
                this.floatingToolbar.refreshI18n();
            }
        });
    }

    async _connect(serverUrl, deviceId, accessCode, iceServers, preferredVideoCodec) {
        this._log(t('log.connectingTo', { deviceId }));
        this._deviceId = deviceId;
        this._connectStartTime = Date.now();
        this._connectionRecorded = false;

        // Set up userApi base URL for connection recording
        userApi.setBaseUrl(serverUrl);
        if (preferredVideoCodec) {
            this._log(t('log.preferredCodec', { codec: preferredVideoCodec }));
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
                if (!this.dcHandler._pc && this.session.pc) {
                    this.dcHandler.setPeerConnection(this.session.pc);
                }
            });

            this.dcHandler.addEventListener('cursorShape', (e) => {
                if (this.cursorRenderer) this.cursorRenderer.updateCursor(e.detail);
            });

            this.dcHandler.addEventListener('videoLayout', (e) => {
                this._onVideoLayout(e.detail);
            });

            this.dcHandler.addEventListener('controlReady', () => {
                this._log(t('log.controlReady'));
                this._sendInitialConfig();
            });

            this.dcHandler.addEventListener('eventReady', () => {
                this._log(t('log.eventReady'));
            });

            this.dcHandler.addEventListener('capabilities', (e) => {
                this._log(t('log.hostCaps', { caps: e.detail.capabilities || '(empty)' }));
            });

            this.dcHandler.addEventListener('hostCapabilities', (e) => {
                const { supportsSendAttentionSequence, supportsLockWorkstation, supportsFileTransfer, supportsPrivacyScreen, supportsVirtualDisplay } = e.detail;
                this._log(t('log.negotiatedCaps', { sas: supportsSendAttentionSequence, lock: supportsLockWorkstation, ft: supportsFileTransfer }));
                this._supportsVirtualDisplay = supportsVirtualDisplay;
                if (this.floatingToolbar) {
                    this.floatingToolbar.setActionSupport(
                        supportsSendAttentionSequence, supportsLockWorkstation, supportsFileTransfer, supportsPrivacyScreen, supportsVirtualDisplay);
                }
                if (supportsVirtualDisplay) {
                    this._sendVirtualDisplayCommand({ action: 'query' });
                }
            });

            this.dcHandler.addEventListener('extensionMessage', (e) => {
                this._onExtensionMessage(e.detail);
            });

            this.dcHandler.addEventListener('fileTransferStarted', (e) => {
                this._addTransferItem(e.detail.transferId, e.detail.filename, e.detail.totalBytes);
            });

            this.dcHandler.addEventListener('fileTransferProgress', (e) => {
                this._updateTransferProgress(e.detail.transferId, e.detail.bytesSent, e.detail.totalBytes);
            });

            this.dcHandler.addEventListener('fileTransferComplete', (e) => {
                this._updateTransferStatus(e.detail.transferId, 'complete');
                this._log(t('log.uploadComplete', { filename: e.detail.filename }));
            });

            this.dcHandler.addEventListener('fileTransferError', (e) => {
                this._updateTransferStatus(e.detail.transferId, 'error', e.detail.errorMessage);
                this._log(t('log.uploadFailed', { error: e.detail.errorMessage }));
            });

            this.dcHandler.addEventListener('fileDownloadStarted', (e) => {
                this._addTransferItem(e.detail.transferId, e.detail.filename, e.detail.totalBytes, 'download');
                this._log(t('log.downloadStarted', { filename: e.detail.filename }));
            });

            this.dcHandler.addEventListener('fileDownloadProgress', (e) => {
                this._updateTransferProgress(e.detail.transferId, e.detail.bytesReceived, e.detail.totalBytes);
            });

            this.dcHandler.addEventListener('fileDownloadComplete', (e) => {
                this._updateTransferStatus(e.detail.transferId, 'complete');
                this._log(t('log.downloadComplete', { filename: e.detail.filename }));
                if (e.detail.blob && !e.detail.streaming) {
                    this._triggerBrowserDownload(e.detail.blob, e.detail.filename);
                }
            });

            this.dcHandler.addEventListener('fileDownloadError', (e) => {
                this._updateTransferStatus(e.detail.transferId, 'error', e.detail.errorMessage);
                this._log(t('log.downloadFailed', { error: e.detail.errorMessage }));
            });

            await this.session.connect(deviceId, accessCode);

        } catch (error) {
            this._log(t('log.connectFailed', { error: error.message }), 'error');
            this._setConnectionState('failed', error.message);
        }
    }

    _onSessionStateChange(detail) {
        const { oldState, newState } = detail;
        this._log(t('log.stateChange', { from: oldState, to: newState }));

        switch (newState) {
            case SessionState.CONNECTED:
                this._setConnectionState('connected', t('status.connected'));
                this._recordConnectionResult('success');
                if (this.floatingToolbar) this.floatingToolbar.setVisible(true);
                if (this.videoStats) this.videoStats.setPeerConnection(this.session.pc);
                this._startConnBarStats();
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
                this._setConnectionState('failed', t('status.failed'));
                this._recordConnectionResult('failed', 'Connection failed');
                break;
            case SessionState.CLOSED:
                this._setConnectionState('failed', t('status.closed'));
                this._recordConnectionResult('success');
                break;
        }
    }

    _recordConnectionResult(status, errorMsg) {
        if (this._connectionRecorded && status === 'success') return; // Don't double-record success
        if (!userApi.isLoggedIn() || !this._deviceId) return;
        this._connectionRecorded = true;

        const duration = Math.round((Date.now() - this._connectStartTime) / 1000);
        userApi.recordConnection(this._deviceId, duration, status, errorMsg || '').catch(() => {});
    }

    _onTrack(detail) {
        const { track, streams } = detail;
        const streamId = (streams && streams.length > 0) ? streams[0].id : '';
        this._log(t('log.trackReceived', { kind: track.kind, stream: streamId }));

        if (!this._receivedStreams) this._receivedStreams = new Map();

        if (track.kind === 'video' && streams && streams.length > 0) {
            this._receivedStreams.set(streams[0].id, streams[0]);
        }

        if (track.kind === 'audio') {
            this._log(t('log.audioTrack', { enabled: track.enabled, muted: track.muted, state: track.readyState }));
            this._pendingAudioTrack = track;
            return;
        }

        if (track.kind === 'video' && this._selectedStreamId && streamId !== this._selectedStreamId) {
            this._log(t('log.ignoreStream', { stream: streamId }));
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
            this._log(t('log.audioMerged'));
        }
        combinedStream.addTrack(track);
        video.srcObject = combinedStream;

        video.play().catch(e => {
            this._log(t('log.autoplayFailed', { error: e.message }), 'warning');
            video.muted = true;
            video.play().then(() => {
                this._showClickToUnmute(video);
            }).catch(() => {
                this._showClickToPlay(video);
            });
        });

        video.onloadedmetadata = () => {
            this._log(t('log.videoResolution', { width: video.videoWidth, height: video.videoHeight }));
            document.getElementById('noVideo')?.style.setProperty('display', 'none');
            if (this.floatingToolbar) {
                this.floatingToolbar.setRemoteResolution(video.videoWidth, video.videoHeight);
            }
            this._setupInputHandlers();
        };
    }

    _showClickToUnmute(video) {
        const overlay = document.getElementById('unmuteOverlay');
        if (!overlay) return;
        overlay.textContent = t('overlay.clickToUnmute');
        overlay.style.display = 'block';
        const handler = () => {
            video.muted = false;
            overlay.style.display = 'none';
            overlay.removeEventListener('click', handler);
            document.removeEventListener('click', handler);
        };
        overlay.addEventListener('click', handler);
        document.addEventListener('click', handler, { once: true });
    }

    _showClickToPlay(video) {
        const overlay = document.getElementById('unmuteOverlay');
        if (!overlay) return;
        overlay.textContent = t('overlay.clickToPlay');
        overlay.style.display = 'block';
        const handler = () => {
            video.muted = false;
            video.play().catch(() => {});
            overlay.style.display = 'none';
            overlay.removeEventListener('click', handler);
            document.removeEventListener('click', handler);
        };
        overlay.addEventListener('click', handler);
        document.addEventListener('click', handler, { once: true });
    }

    _onVideoLayout(layout) {
        if (!layout.videoTracks || layout.videoTracks.length === 0) return;

        this._log(t('log.videoLayout', { count: layout.videoTracks.length }));

        this._currentDisplayList = layout.videoTracks;

        let selectedTrack = null;
        let selectedIndex = 0;

        if (this._selectedStreamId) {
            for (let i = 0; i < layout.videoTracks.length; i++) {
                if (layout.videoTracks[i].mediaStreamId === this._selectedStreamId) {
                    selectedTrack = layout.videoTracks[i];
                    selectedIndex = i;
                    break;
                }
            }
        }

        if (!selectedTrack) {
            if (layout.primaryScreenId !== undefined) {
                for (let i = 0; i < layout.videoTracks.length; i++) {
                    if (layout.videoTracks[i].screenId === layout.primaryScreenId) {
                        selectedTrack = layout.videoTracks[i];
                        selectedIndex = i;
                        break;
                    }
                }
            }
            if (!selectedTrack) {
                selectedTrack = layout.videoTracks[0];
                selectedIndex = 0;
            }
        }

        this._log(t('log.selectMonitor', { width: selectedTrack.width, height: selectedTrack.height }));
        this._remoteWidth = selectedTrack.width;
        this._remoteHeight = selectedTrack.height;
        this._activeDisplayIndex = selectedIndex;

        const resEl = document.getElementById('connResolution');
        if (resEl) resEl.textContent = `${selectedTrack.width}x${selectedTrack.height}`;

        if (selectedTrack.mediaStreamId) {
            this._selectedStreamId = selectedTrack.mediaStreamId;
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
            this.mouseHandler.setRemoteResolution(selectedTrack.width, selectedTrack.height);
        }
        if (this.touchHandler) {
            this.touchHandler.setRemoteResolution(selectedTrack.width, selectedTrack.height);
        }

        if (this.floatingToolbar) {
            this.floatingToolbar.updateDisplayList(layout.videoTracks, selectedIndex);
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

        this._setupDragDrop(videoContainer || video);
        this._setupPasteUpload(videoContainer || video);

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
        this.dcHandler.sendCapabilities(
            'sendAttentionSequenceAction lockWorkstationAction fileTransfer privacyScreen virtualDisplay');
        this.dcHandler.sendAudioControl({ enable: true });
        this._log(t('log.sentConfig'));
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
            case 'sendAttentionSequence':
                if (this.dcHandler) {
                    this.dcHandler.sendAction('sendAttentionSequence');
                    this._log(t('log.sentCAD'));
                }
                break;
            case 'lockWorkstation':
                if (this.dcHandler) {
                    this.dcHandler.sendAction('lockWorkstation');
                    this._log(t('log.sentLock'));
                }
                break;
            case 'enablePrivacyScreen':
                if (this.dcHandler) {
                    this.dcHandler.sendAction('enablePrivacyScreen');
                    this._log(t('log.privacyScreenEnabled'));
                }
                break;
            case 'disablePrivacyScreen':
                if (this.dcHandler) {
                    this.dcHandler.sendAction('disablePrivacyScreen');
                    this._log(t('log.privacyScreenDisabled'));
                }
                break;
            case 'uploadFile':
                this._triggerFileUpload();
                break;
            case 'downloadFile':
                this._onDownloadFromHost();
                break;
            case 'showTransfers':
                this._toggleTransferPanel();
                break;
            case 'selectDisplay':
                this._selectDisplay(detail.index);
                break;
            case 'createVirtualDisplay':
                this._sendVirtualDisplayCommand({
                    action: 'create',
                    width: detail.width,
                    height: detail.height,
                    refreshRate: detail.refreshRate,
                });
                break;
            case 'removeVirtualDisplay':
                this._sendVirtualDisplayCommand({
                    action: 'remove',
                    index: detail.index,
                });
                break;
            case 'removeAllVirtualDisplays':
                this._sendVirtualDisplayCommand({ action: 'removeAll' });
                break;
        }
    }

    // ==================== Display Switching ====================

    _selectDisplay(index) {
        if (!this._currentDisplayList || index < 0 || index >= this._currentDisplayList.length) return;

        const track = this._currentDisplayList[index];
        if (!track.mediaStreamId) return;

        this._selectedStreamId = track.mediaStreamId;
        this._activeDisplayIndex = index;
        this._remoteWidth = track.width;
        this._remoteHeight = track.height;

        if (this._receivedStreams) {
            const targetStream = this._receivedStreams.get(this._selectedStreamId);
            if (targetStream) {
                const video = document.getElementById('remoteVideo');
                if (video) {
                    const combinedStream = new MediaStream();
                    if (this._pendingAudioTrack && this._pendingAudioTrack.readyState === 'live') {
                        combinedStream.addTrack(this._pendingAudioTrack);
                    }
                    for (const vt of targetStream.getVideoTracks()) {
                        combinedStream.addTrack(vt);
                    }
                    video.srcObject = combinedStream;
                }
            }
        }

        if (this.mouseHandler) this.mouseHandler.setRemoteResolution(track.width, track.height);
        if (this.touchHandler) this.touchHandler.setRemoteResolution(track.width, track.height);

        const resEl = document.getElementById('connResolution');
        if (resEl) resEl.textContent = `${track.width}x${track.height}`;

        if (this.floatingToolbar) {
            this.floatingToolbar.updateDisplayList(this._currentDisplayList, index);
        }

        this._log(t('log.switchDisplay', { index }));
    }

    // ==================== Virtual Display ====================

    _sendVirtualDisplayCommand(command) {
        if (!this.dcHandler || !this._supportsVirtualDisplay) return;
        this.dcHandler.sendExtensionMessage('qd-virtual-display', JSON.stringify(command));
    }

    _onExtensionMessage(msg) {
        if (!msg || msg.type !== 'qd-virtual-display') return;

        let state;
        try {
            state = JSON.parse(msg.data);
        } catch (e) {
            return;
        }

        const type = state.type || '';
        if (type === 'created') {
            this._log(t('log.vdCreated', { index: state.index, resolution: `${state.width}x${state.height}` }));
            this._sendVirtualDisplayCommand({ action: 'query' });
        } else if (type === 'removed') {
            this._log(t('log.vdRemoved', { index: state.index }));
            this._sendVirtualDisplayCommand({ action: 'query' });
        } else if (type === 'removedAll') {
            this._log(t('log.vdRemovedAll'));
            if (this.floatingToolbar) this.floatingToolbar.updateVirtualDisplays([]);
        } else if (type === 'state') {
            const displays = state.displays || [];
            const active = displays.filter(d => d.active);
            this._log(t('log.vdQuery', { count: active.length }));
            if (this.floatingToolbar) this.floatingToolbar.updateVirtualDisplays(displays);
        } else if (type === 'error') {
            this._log(t('log.vdError', { message: state.message || 'Unknown' }), 'warning');
        }
    }

    // ==================== File Transfer ====================

    _setupDragDrop(target) {
        if (!target) return;
        let dragCounter = 0;

        target.addEventListener('dragenter', (e) => {
            e.preventDefault();
            e.stopPropagation();
            dragCounter++;
            if (dragCounter === 1) this._showDropOverlay(target);
        });

        target.addEventListener('dragover', (e) => {
            e.preventDefault();
            e.stopPropagation();
        });

        target.addEventListener('dragleave', (e) => {
            e.preventDefault();
            e.stopPropagation();
            dragCounter--;
            if (dragCounter <= 0) {
                dragCounter = 0;
                this._hideDropOverlay(target);
            }
        });

        target.addEventListener('drop', (e) => {
            e.preventDefault();
            e.stopPropagation();
            dragCounter = 0;
            this._hideDropOverlay(target);
            if (e.dataTransfer?.files?.length > 0) {
                this._uploadFiles(e.dataTransfer.files);
            }
        });
    }

    _showDropOverlay(target) {
        if (this._dropOverlay) return;
        const overlay = document.createElement('div');
        overlay.className = 'drop-overlay';
        overlay.innerHTML = `<div class="drop-overlay-content">
            <div class="drop-icon">📤</div>
            <div class="drop-text">${t('drop.text')}</div>
        </div>`;
        target.style.position = 'relative';
        target.appendChild(overlay);
        this._dropOverlay = overlay;
    }

    _hideDropOverlay(target) {
        if (this._dropOverlay) {
            this._dropOverlay.remove();
            this._dropOverlay = null;
        }
    }

    _uploadFiles(fileList) {
        if (!this.dcHandler || !fileList || fileList.length === 0) return;
        for (const file of fileList) {
            this._log(t('log.uploadingFile', { name: file.name, size: file.size }));
            this.dcHandler.startFileUpload(file);
        }
        this._showTransferPanel();
    }

    _setupPasteUpload(target) {
        if (!target) return;
        target.addEventListener('paste', (e) => {
            const files = e.clipboardData?.files;
            if (files && files.length > 0) {
                e.preventDefault();
                this._uploadFiles(files);
            }
        });
    }

    async _onDownloadFromHost() {
        if (!this.dcHandler) return;

        let fileHandle = null;
        if (window.showSaveFilePicker) {
            try {
                fileHandle = await window.showSaveFilePicker({
                    suggestedName: 'download',
                });
            } catch (err) {
                if (err.name === 'AbortError') {
                    this._log(t('log.downloadCancelled'));
                    return;
                }
                this._log(t('log.savePickerError', { error: err.message }), 'warning');
            }
        }

        this.dcHandler.startFileDownload(fileHandle);
        this._log(fileHandle
            ? t('log.downloadStreaming')
            : t('log.downloadBuffered'));
    }

    _triggerFileUpload() {
        const input = document.createElement('input');
        input.type = 'file';
        input.multiple = true;
        input.style.display = 'none';
        input.addEventListener('change', () => {
            this._uploadFiles(input.files);
            input.remove();
        });
        document.body.appendChild(input);
        input.click();
    }

    // ==================== Transfer Panel ====================

    _ensureTransferPanel() {
        if (this._transferPanel) return;

        const panel = document.createElement('div');
        panel.className = 'transfer-panel';
        panel.innerHTML = `
            <div class="transfer-panel-header">
                <span class="transfer-panel-title">File Transfers</span>
                <span class="transfer-panel-close" title="Close">&times;</span>
            </div>
            <div class="transfer-panel-list"></div>
            <div class="transfer-panel-footer">
                <button class="transfer-panel-clear">Clear Completed</button>
            </div>
        `;
        panel.style.display = 'none';

        panel.querySelector('.transfer-panel-close').addEventListener('click', () => {
            this._hideTransferPanel();
        });
        panel.querySelector('.transfer-panel-clear').addEventListener('click', () => {
            this._clearCompletedTransfers();
        });

        const container = document.getElementById('remoteContainer') || document.body;
        container.appendChild(panel);
        this._transferPanel = panel;
        this._transferItems = new Map();
    }

    _showTransferPanel() {
        this._ensureTransferPanel();
        this._transferPanel.style.display = 'flex';
    }

    _hideTransferPanel() {
        if (this._transferPanel) {
            this._transferPanel.style.display = 'none';
        }
    }

    _toggleTransferPanel() {
        this._ensureTransferPanel();
        if (this._transferPanel.style.display === 'none') {
            this._showTransferPanel();
        } else {
            this._hideTransferPanel();
        }
    }

    _addTransferItem(transferId, filename, totalBytes, direction = 'upload') {
        this._ensureTransferPanel();
        const list = this._transferPanel.querySelector('.transfer-panel-list');
        const item = document.createElement('div');
        item.className = 'transfer-item';
        item.dataset.transferId = transferId;
        item.dataset.direction = direction;
        const dirIcon = direction === 'download' ? '📥' : '📤';
        item.innerHTML = `
            <div class="transfer-item-row">
                <span class="transfer-item-icon">${dirIcon}</span>
                <span class="transfer-item-name" title="${filename}">${filename}</span>
                <span class="transfer-item-pct">0%</span>
                <span class="transfer-item-cancel" title="Cancel">&times;</span>
            </div>
            <div class="transfer-item-bar"><div class="transfer-item-fill"></div></div>
        `;
        item.querySelector('.transfer-item-cancel').addEventListener('click', () => {
            if (this.dcHandler) {
                if (direction === 'download') {
                    this.dcHandler.cancelFileDownload(transferId);
                } else {
                    this.dcHandler.cancelFileUpload(transferId);
                }
            }
        });
        list.appendChild(item);
        this._transferItems.set(transferId, item);
        this._updateActiveCount();
        this._showTransferPanel();
    }

    _updateTransferProgress(transferId, bytesSent, totalBytes) {
        const item = this._transferItems?.get(transferId);
        if (!item) return;
        const pct = totalBytes > 0 ? Math.round(bytesSent / totalBytes * 100) : 0;
        const pctEl = item.querySelector('.transfer-item-pct');
        if (pctEl) pctEl.textContent = `${pct}%`;
        const fill = item.querySelector('.transfer-item-fill');
        if (fill) fill.style.width = `${pct}%`;
    }

    _updateTransferStatus(transferId, status, errorMessage) {
        const item = this._transferItems?.get(transferId);
        if (!item) return;

        const pctEl = item.querySelector('.transfer-item-pct');
        const cancelEl = item.querySelector('.transfer-item-cancel');
        const fill = item.querySelector('.transfer-item-fill');
        const iconEl = item.querySelector('.transfer-item-icon');

        if (status === 'complete') {
            if (pctEl) pctEl.textContent = '✅';
            if (cancelEl) cancelEl.style.display = 'none';
            if (fill) { fill.style.width = '100%'; fill.classList.add('complete'); }
            if (iconEl) iconEl.textContent = '✅';
            item.classList.add('transfer-complete');
        } else {
            const msg = errorMessage || 'Failed';
            if (pctEl) pctEl.textContent = '❌';
            if (cancelEl) cancelEl.style.display = 'none';
            if (fill) fill.classList.add('error');
            if (iconEl) iconEl.textContent = '❌';
            item.classList.add('transfer-error');
            const errEl = document.createElement('div');
            errEl.className = 'transfer-item-error';
            errEl.textContent = msg;
            item.appendChild(errEl);
        }
        this._updateActiveCount();
    }

    _clearCompletedTransfers() {
        if (!this._transferItems) return;
        for (const [id, item] of this._transferItems) {
            if (item.classList.contains('transfer-complete') || item.classList.contains('transfer-error')) {
                item.remove();
                this._transferItems.delete(id);
            }
        }
        this._updateActiveCount();
    }

    _updateActiveCount() {
        if (!this._transferItems || !this.floatingToolbar) return;
        let count = 0;
        for (const [, item] of this._transferItems) {
            if (!item.classList.contains('transfer-complete') && !item.classList.contains('transfer-error')) {
                count++;
            }
        }
        this.floatingToolbar.updateTransferCount(count);
    }

    _triggerBrowserDownload(blob, filename) {
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = filename;
        a.style.display = 'none';
        document.body.appendChild(a);
        a.click();
        setTimeout(() => {
            document.body.removeChild(a);
            URL.revokeObjectURL(url);
        }, 100);
    }

    _handleSettingChange(detail) {
        if (!this.dcHandler) return;

        switch (detail.setting) {
            case 'framerate':
                this.dcHandler.sendVideoControl({ enable: true, targetFramerate: detail.value });
                this._log(t('log.targetFps', { fps: detail.value }));
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
                this._log(t('log.boostMode', { mode: detail.value }));
                break;
            }
            case 'bitrate':
                this.dcHandler.sendPeerConnectionParameters({
                    preferredMinBitrateBps: detail.value,
                });
                this._log(t('log.minBitrate', { bitrate: Math.round(detail.value / (1024 * 1024)) }));
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
                this._log(t('log.resolution', { width, height }));
                break;
            }
            case 'audio':
                this.dcHandler.sendAudioControl({ enable: detail.value });
                this._log(t('log.audioToggle', { state: detail.value ? t('log.audioOn') : t('log.audioOff') }));
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
            this._log(t('log.noVideoScreenshot'), 'warning');
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
            this._log(t('log.screenshotSaved'));
        }, 'image/png');
    }

    _disconnect() {
        if (this.session) {
            this.session.disconnect();
        }
        this._cleanupInputHandlers();
        this._setConnectionState('failed', t('status.disconnected'));
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
        if (state !== 'connected') {
            this._stopConnBarStats();
            const routeEl = document.getElementById('connRoute');
            const pingEl = document.getElementById('connPing');
            if (routeEl) routeEl.style.display = 'none';
            if (pingEl) pingEl.style.display = 'none';
        }
    }

    _startConnBarStats() {
        if (this._connBarInterval) return;
        this._connBarInterval = setInterval(() => this._updateConnBarStats(), 2000);
        this._updateConnBarStats();
    }

    _stopConnBarStats() {
        if (this._connBarInterval) {
            clearInterval(this._connBarInterval);
            this._connBarInterval = null;
        }
    }

    async _updateConnBarStats() {
        if (!this.session?.pc || this.session.pc.connectionState === 'closed') return;
        try {
            const stats = await this.session.pc.getStats();
            let rtt = 0;
            let localId = null;
            let remoteId = null;

            stats.forEach(report => {
                if (report.type === 'candidate-pair' && report.state === 'succeeded') {
                    if (!localId || report.nominated) {
                        rtt = Math.round((report.currentRoundTripTime || 0) * 1000);
                        localId = report.localCandidateId;
                        remoteId = report.remoteCandidateId;
                    }
                }
            });

            let localType = '';
            let remoteType = '';
            if (localId) {
                const lc = stats.get(localId);
                if (lc) localType = lc.candidateType || '';
            }
            if (remoteId) {
                const rc = stats.get(remoteId);
                if (rc) remoteType = rc.candidateType || '';
            }

            // Route type
            const routeEl = document.getElementById('connRoute');
            const routeDot = document.getElementById('connRouteDot');
            const routeLabel = document.getElementById('connRouteLabel');
            if (routeEl && localType) {
                let label, color;
                if (localType === 'relay' || remoteType === 'relay') {
                    label = 'Relay'; color = '#FFA726';
                } else if (localType === 'srflx' || localType === 'prflx' ||
                           remoteType === 'srflx' || remoteType === 'prflx') {
                    label = 'STUN'; color = '#66BB6A';
                } else {
                    label = 'P2P'; color = '#66BB6A';
                }
                routeDot.style.background = color;
                routeLabel.textContent = label;
                routeEl.style.display = 'inline-flex';
            }

            // Ping
            const pingEl = document.getElementById('connPing');
            const pingDot = document.getElementById('connPingDot');
            const pingValue = document.getElementById('connPingValue');
            if (pingEl && rtt > 0) {
                const pingColor = rtt < 50 ? '#4caf50' : rtt < 100 ? '#ffc107' : '#f44336';
                pingDot.style.background = pingColor;
                pingValue.textContent = rtt + ' ms';
                pingEl.style.display = 'inline-flex';
            }
        } catch (e) {
            // ignore
        }
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
