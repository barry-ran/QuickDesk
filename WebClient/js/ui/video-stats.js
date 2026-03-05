/**
 * video-stats.js - Video statistics overlay
 * 
 * Displays latency, bandwidth, framerate, codec information.
 */

import { t } from '../i18n.js';

export class VideoStats {
    /**
     * @param {HTMLElement} overlayElement
     * @param {RTCPeerConnection} pc
     */
    constructor(overlayElement, pc) {
        this.overlay = overlayElement;
        this.pc = pc;
        this._visible = false;
        this._updateInterval = null;
        this._prevStats = null;
        this._prevTimestamp = 0;
    }

    show() {
        this._visible = true;
        this.overlay.style.display = 'block';
        this._startUpdate();
    }

    hide() {
        this._visible = false;
        this.overlay.style.display = 'none';
        this._stopUpdate();
    }

    toggle() {
        if (this._visible) {
            this.hide();
        } else {
            this.show();
        }
    }

    /** @private */
    _startUpdate() {
        if (this._updateInterval) return;
        this._updateInterval = setInterval(() => this._update(), 1000);
        this._update();
    }

    /** @private */
    _stopUpdate() {
        if (this._updateInterval) {
            clearInterval(this._updateInterval);
            this._updateInterval = null;
        }
    }

    /** @private */
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

            let fps = 0;
            let bitrate = 0;

            if (this._prevStats && timeDelta > 0) {
                const prevVideo = this._prevStats.video;
                if (prevVideo) {
                    const framesDelta = (videoStats.framesDecoded || 0) - (prevVideo.framesDecoded || 0);
                    fps = Math.round(framesDelta / timeDelta);

                    const bytesDelta = (videoStats.bytesReceived || 0) - (prevVideo.bytesReceived || 0);
                    bitrate = Math.round((bytesDelta * 8) / timeDelta / 1000);
                }
            }

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

    /** @private */
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
                    <span class="stats-label">${t('stats.rtt')}</span>
                    <span class="stats-value" style="color:${rttColor}">${rtt}ms</span>
                </div>
                <div class="stats-row">
                    <span class="stats-label">${t('stats.fps')}</span>
                    <span class="stats-value">${data.fps}</span>
                </div>
                <div class="stats-row">
                    <span class="stats-label">${t('stats.bitrate')}</span>
                    <span class="stats-value">${data.bitrate > 1000 ? (data.bitrate/1000).toFixed(1) + ' Mbps' : data.bitrate + ' kbps'}</span>
                </div>
                <div class="stats-row">
                    <span class="stats-label">${t('stats.resolution')}</span>
                    <span class="stats-value">${resolution}</span>
                </div>
                <div class="stats-row">
                    <span class="stats-label">${t('stats.codec')}</span>
                    <span class="stats-value">${codec}</span>
                </div>
                <div class="stats-row">
                    <span class="stats-label">${t('stats.decoder')}</span>
                    <span class="stats-value">${decoder}</span>
                </div>
                <div class="stats-row">
                    <span class="stats-label">${t('stats.jitter')}</span>
                    <span class="stats-value">${jitter}</span>
                </div>
                <div class="stats-row">
                    <span class="stats-label">${t('stats.packetsLost')}</span>
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
     * @param {RTCPeerConnection} pc 
     */
    setPeerConnection(pc) {
        this.pc = pc;
    }

    getCurrentStats() {
        return this._prevStats;
    }

    destroy() {
        this._stopUpdate();
        this.overlay.innerHTML = '';
    }
}
