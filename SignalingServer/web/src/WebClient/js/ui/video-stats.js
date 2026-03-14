/**
 * video-stats.js - 视频统计覆盖层
 * 
 * 参照 VideoStatsOverlay.qml
 * 显示延迟分解、带宽、帧率、编解码器信息
 */

export class VideoStats {
    /**
     * @param {HTMLElement} overlayElement - 覆盖层容器元素
     * @param {RTCPeerConnection} pc - WebRTC PeerConnection
     */
    constructor(overlayElement, pc) {
        this.overlay = overlayElement;
        this.pc = pc;
        this._visible = false;
        this._updateInterval = null;
        this._prevStats = null;
        this._prevTimestamp = 0;
    }

    /**
     * 显示统计覆盖层
     */
    show() {
        this._visible = true;
        this.overlay.style.display = 'block';
        this._startUpdate();
    }

    /**
     * 隐藏统计覆盖层
     */
    hide() {
        this._visible = false;
        this.overlay.style.display = 'none';
        this._stopUpdate();
    }

    /**
     * 切换显示状态
     */
    toggle() {
        if (this._visible) {
            this.hide();
        } else {
            this.show();
        }
    }

    /**
     * 开始更新
     * @private
     */
    _startUpdate() {
        if (this._updateInterval) return;
        this._updateInterval = setInterval(() => this._update(), 1000);
        this._update();
    }

    /**
     * 停止更新
     * @private
     */
    _stopUpdate() {
        if (this._updateInterval) {
            clearInterval(this._updateInterval);
            this._updateInterval = null;
        }
    }

    /**
     * 更新统计信息
     * @private
     */
    async _update() {
        if (!this.pc || this.pc.connectionState === 'closed') return;

        try {
            const stats = await this.pc.getStats();
            const now = Date.now();
            const timeDelta = this._prevTimestamp ? (now - this._prevTimestamp) / 1000 : 1;

            let videoStats = {};
            let audioStats = {};
            let candidatePair = null;

            stats.forEach(report => {
                if (report.type === 'inbound-rtp' && report.kind === 'video') {
                    videoStats = {
                        bytesReceived: report.bytesReceived,
                        framesDecoded: report.framesDecoded,
                        framesReceived: report.framesReceived,
                        framesDropped: report.framesDropped,
                        frameWidth: report.frameWidth,
                        frameHeight: report.frameHeight,
                        jitter: report.jitter,
                        packetsLost: report.packetsLost,
                        packetsReceived: report.packetsReceived,
                        decoderImplementation: report.decoderImplementation,
                        codec: null,
                    };

                    // 查找编解码器信息
                    if (report.codecId) {
                        const codecReport = stats.get(report.codecId);
                        if (codecReport) {
                            videoStats.codec = codecReport.mimeType;
                        }
                    }
                }

                if (report.type === 'inbound-rtp' && report.kind === 'audio') {
                    audioStats = {
                        bytesReceived: report.bytesReceived,
                        packetsLost: report.packetsLost,
                        jitter: report.jitter,
                    };
                }

                if (report.type === 'candidate-pair' && report.state === 'succeeded') {
                    candidatePair = {
                        currentRoundTripTime: report.currentRoundTripTime,
                        availableOutgoingBitrate: report.availableOutgoingBitrate,
                        bytesReceived: report.bytesReceived,
                        bytesSent: report.bytesSent,
                    };
                }
            });

            // 计算变化率
            let fps = 0;
            let bitrate = 0;

            if (this._prevStats && timeDelta > 0) {
                const prevVideo = this._prevStats.video;
                if (prevVideo) {
                    const framesDelta = (videoStats.framesDecoded || 0) - (prevVideo.framesDecoded || 0);
                    fps = Math.round(framesDelta / timeDelta);

                    const bytesDelta = (videoStats.bytesReceived || 0) - (prevVideo.bytesReceived || 0);
                    bitrate = Math.round((bytesDelta * 8) / timeDelta / 1000); // kbps
                }
            }

            // 渲染
            this._render({
                video: videoStats,
                audio: audioStats,
                network: candidatePair,
                fps,
                bitrate,
            });

            this._prevStats = { video: videoStats, audio: audioStats };
            this._prevTimestamp = now;

        } catch (e) {
            console.warn('[VideoStats] Failed to get stats:', e);
        }
    }

    /**
     * 渲染统计信息
     * @private
     */
    _render(data) {
        const rtt = data.network ? Math.round((data.network.currentRoundTripTime || 0) * 1000) : 0;
        const rttColor = rtt < 50 ? '#4caf50' : rtt < 100 ? '#ffc107' : '#f44336';

        const resolution = (data.video.frameWidth && data.video.frameHeight) 
            ? `${data.video.frameWidth}x${data.video.frameHeight}` : 'N/A';
        const codec = data.video.codec || 'N/A';
        const decoder = data.video.decoderImplementation || 'N/A';
        const jitter = data.video.jitter ? `${(data.video.jitter * 1000).toFixed(1)}ms` : 'N/A';
        const packetsLost = data.video.packetsLost || 0;

        this.overlay.innerHTML = `
            <div class="stats-grid">
                <div class="stats-row">
                    <span class="stats-label">RTT</span>
                    <span class="stats-value" style="color:${rttColor}">${rtt}ms</span>
                </div>
                <div class="stats-row">
                    <span class="stats-label">FPS</span>
                    <span class="stats-value">${data.fps}</span>
                </div>
                <div class="stats-row">
                    <span class="stats-label">Bitrate</span>
                    <span class="stats-value">${data.bitrate > 1000 ? (data.bitrate/1000).toFixed(1) + ' Mbps' : data.bitrate + ' kbps'}</span>
                </div>
                <div class="stats-row">
                    <span class="stats-label">Resolution</span>
                    <span class="stats-value">${resolution}</span>
                </div>
                <div class="stats-row">
                    <span class="stats-label">Codec</span>
                    <span class="stats-value">${codec}</span>
                </div>
                <div class="stats-row">
                    <span class="stats-label">Decoder</span>
                    <span class="stats-value">${decoder}</span>
                </div>
                <div class="stats-row">
                    <span class="stats-label">Jitter</span>
                    <span class="stats-value">${jitter}</span>
                </div>
                <div class="stats-row">
                    <span class="stats-label">Packets Lost</span>
                    <span class="stats-value">${packetsLost}</span>
                </div>
                ${rtt > 0 ? `
                <div class="stats-bar">
                    <div class="stats-bar-fill" style="width:${Math.min(rtt/2, 100)}%;background:${rttColor}"></div>
                </div>` : ''}
            </div>
        `;
    }

    /**
     * 设置 PeerConnection
     * @param {RTCPeerConnection} pc 
     */
    setPeerConnection(pc) {
        this.pc = pc;
    }

    /**
     * 获取当前统计数据快照
     * @returns {object|null}
     */
    getCurrentStats() {
        return this._prevStats;
    }

    /**
     * 清理
     */
    destroy() {
        this._stopUpdate();
        this.overlay.innerHTML = '';
    }
}
