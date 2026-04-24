/**
 * floating-toolbar.js - Floating toolbar
 * 
 * Matches FloatingToolButton.qml menu structure.
 * Draggable floating button + dropdown menu (with submenus).
 */

import { t } from '../i18n.js';

export class FloatingToolbar extends EventTarget {
    /**
     * @param {HTMLElement} container
     */
    constructor(container) {
        super();
        this.container = container;
        this._element = null;
        this._menuElement = null;
        this._activeSubmenu = null;
        this._isDragging = false;
        this._dragOffset = { x: 0, y: 0 };
        this._menuVisible = false;

        this.settings = {
            targetFramerate: 30,
            framerateBoostMode: 'office',
            preferredMinBitrate: 10485760,
            audioEnabled: true,
            statsVisible: false,
        };

        this._supportsSAS = false;
        this._supportsLock = false;
        this._supportsFileTransfer = false;
        this._supportsPrivacyScreen = false;
        this._supportsVirtualDisplay = false;
        this._privacyScreenActive = false;

        this._displayList = [];
        this._activeDisplayIndex = -1;
        this._virtualDisplays = [];

        this._remoteWidth = 0;
        this._remoteHeight = 0;
        this._originalWidth = 0;
        this._originalHeight = 0;

        this._create();
    }

    setRemoteResolution(width, height) {
        this._remoteWidth = width;
        this._remoteHeight = height;
        if (!this._originalWidth && width > 0) {
            this._originalWidth = width;
            this._originalHeight = height;
        }
    }

    /** @private */
    _create() {
        this._element = document.createElement('div');
        this._element.className = 'floating-btn';
        this._element.innerHTML = '<svg viewBox="0 0 24 24" width="24" height="24" fill="white"><path d="M12 8c1.1 0 2-.9 2-2s-.9-2-2-2-2 .9-2 2 .9 2 2 2zm0 2c-1.1 0-2 .9-2 2s.9 2 2 2 2-.9 2-2-.9-2-2-2zm0 6c-1.1 0-2 .9-2 2s.9 2 2 2 2-.9 2-2-.9-2-2-2z"/></svg>';

        this._menuElement = document.createElement('div');
        this._menuElement.className = 'floating-menu';
        this._menuElement.style.display = 'none';
        this._buildMenu();

        this.container.appendChild(this._element);
        this.container.appendChild(this._menuElement);

        this._element.addEventListener('click', (e) => {
            if (!this._isDragging) this._toggleMenu();
        });
        this._element.addEventListener('mousedown', (e) => {
            e.preventDefault();
            this._startDrag(e);
        });

        this._menuElement.addEventListener('mousedown', (e) => e.preventDefault());
        document.addEventListener('mousemove', (e) => this._onDrag(e));
        document.addEventListener('mouseup', () => this._endDrag());

        this._element.addEventListener('touchstart', (e) => {
            e.preventDefault();
            this._startDrag(e.touches[0]);
        }, { passive: false });
        document.addEventListener('touchmove', (e) => {
            if (this._maybeDragging) this._onDrag(e.touches[0]);
        }, { passive: true });
        document.addEventListener('touchend', () => {
            if (this._maybeDragging) {
                this._endDrag();
                if (!this._isDragging) this._toggleMenu();
            }
        });

        document.addEventListener('click', (e) => {
            if (!this._element.contains(e.target) && 
                !this._menuElement.contains(e.target) &&
                !(this._activeSubmenu && this._activeSubmenu.contains(e.target))) {
                this._hideMenu();
            }
        });
    }

    /** @private */
    _buildMenu() {
        this._menuElement.innerHTML = '';

        const items = [
            { textKey: 'menu.smartBoost', icon: '⚡', hasSubmenu: true, action: 'submenu-boost' },
            { textKey: 'menu.targetFramerate', icon: '🎯', hasSubmenu: true, action: 'submenu-framerate' },
            { textKey: 'menu.resolution', icon: '🖥️', hasSubmenu: true, action: 'submenu-resolution' },
            { textKey: 'menu.bitrate', icon: '📶', hasSubmenu: true, action: 'submenu-bitrate' },
            { type: 'separator' },
            { textKey: 'menu.fitWindow', icon: '⛶', action: 'fitWindow' },
            { textKey: 'menu.videoStats', icon: '📊', action: 'toggleStats' },
            { textKey: '', icon: '🔊', action: 'toggleAudio', id: 'audioMenuItem' },
            { textKey: 'menu.screenshot', icon: '📷', action: 'screenshot' },
            { textKey: 'menu.logs', icon: '📋', action: 'toggleLogs' },
            { type: 'separator', id: 'actionSeparator', hidden: true },
            { textKey: 'menu.sendCAD', icon: '⌨', action: 'sendAttentionSequence', id: 'sasMenuItem', hidden: true },
            { textKey: 'menu.lockScreen', icon: '🔒', action: 'lockWorkstation', id: 'lockMenuItem', hidden: true },
            { textKey: 'menu.privacyScreen', icon: '👁', action: 'togglePrivacyScreen', id: 'privacyScreenMenuItem', hidden: true },
            { type: 'separator', id: 'uploadSeparator', hidden: true },
            { textKey: 'menu.uploadFile', icon: '📤', action: 'uploadFile', id: 'uploadMenuItem', hidden: true },
            { textKey: 'menu.downloadFile', icon: '📥', action: 'downloadFile', id: 'downloadMenuItem', hidden: true },
            { textKey: 'menu.transfers', icon: '📊', action: 'showTransfers', id: 'transfersMenuItem', hidden: true },
            { type: 'separator', id: 'vdSeparator', hidden: true },
            { textKey: 'menu.virtualDisplay', icon: '🖥️', hasSubmenu: true, action: 'submenu-virtualDisplay', id: 'vdMenuItem', hidden: true },
            { type: 'separator' },
            { textKey: 'menu.disconnect', icon: '✕', action: 'disconnect', destructive: true },
        ];

        for (const item of items) {
            if (item.type === 'separator') {
                const sep = document.createElement('div');
                sep.className = 'menu-separator';
                if (item.id) sep.id = item.id;
                if (item.hidden) sep.style.display = 'none';
                this._menuElement.appendChild(sep);
                continue;
            }

            const el = document.createElement('div');
            el.className = 'menu-item' + (item.destructive ? ' destructive' : '');
            if (item.id) el.id = item.id;
            if (item.hidden) el.style.display = 'none';

            if (item.action === 'toggleAudio') {
                const label = this.settings.audioEnabled ? t('menu.muteAudio') : t('menu.unmuteAudio');
                el.innerHTML = `<span class="menu-icon">${this.settings.audioEnabled ? '🔊' : '🔇'}</span><span class="menu-text">${label}</span>`;
            } else {
                el.innerHTML = `<span class="menu-icon">${item.icon}</span><span class="menu-text">${t(item.textKey)}</span>`;
            }

            if (item.hasSubmenu) {
                el.innerHTML += '<span class="menu-arrow">›</span>';
            }

            el.addEventListener('click', (e) => {
                e.stopPropagation();
                this._handleMenuClick(item.action, el);
            });

            this._menuElement.appendChild(el);
        }

        this._createSubmenus();
    }

    /** @private */
    _createSubmenus() {
        this._submenus = {};

        this._submenus.boost = this._createSubmenu([
            { text: t('menu.off'), value: 'off', checkGroup: 'boost' },
            { text: t('menu.office'), value: 'office', checkGroup: 'boost' },
            { text: t('menu.gaming'), value: 'gaming', checkGroup: 'boost' },
        ]);

        this._submenus.framerate = this._createSubmenu([
            { text: '60 FPS', value: 60, checkGroup: 'framerate' },
            { text: '30 FPS', value: 30, checkGroup: 'framerate' },
            { text: '15 FPS', value: 15, checkGroup: 'framerate' },
            { text: '5 FPS', value: 5, checkGroup: 'framerate' },
        ]);

        this._submenus.resolution = this._createSubmenu([
            { text: t('menu.original'), value: 'original', id: 'resOriginal' },
            { type: 'separator' },
            { text: '3840 × 2160 (4K)', value: '3840x2160' },
            { text: '2560 × 1440 (2K)', value: '2560x1440' },
            { text: '1920 × 1080 (FHD)', value: '1920x1080' },
            { text: '1600 × 900', value: '1600x900' },
            { text: '1366 × 768', value: '1366x768' },
            { text: '1280 × 720', value: '1280x720' },
            { text: '1024 × 768', value: '1024x768' },
        ]);

        this._submenus.bitrate = this._createSubmenu([
            { text: '100 MiB', value: 104857600, checkGroup: 'bitrate' },
            { text: '50 MiB', value: 52428800, checkGroup: 'bitrate' },
            { text: '10 MiB', value: 10485760, checkGroup: 'bitrate' },
            { text: '5 MiB', value: 5242880, checkGroup: 'bitrate' },
            { text: '2 MiB', value: 2097152, checkGroup: 'bitrate' },
        ]);

        this._submenus.virtualDisplay = this._createVirtualDisplaySubmenu();

        this._displayHeader = this._createDisplayHeader();
    }

    /** @private */
    _createSubmenu(items) {
        const submenu = document.createElement('div');
        submenu.className = 'floating-submenu';
        submenu.style.display = 'none';

        for (const item of items) {
            if (item.type === 'separator') {
                const sep = document.createElement('div');
                sep.className = 'menu-separator';
                submenu.appendChild(sep);
                continue;
            }

            const el = document.createElement('div');
            el.className = 'menu-item';
            if (item.id) el.id = item.id;
            el.dataset.value = item.value;
            if (item.checkGroup) el.dataset.checkGroup = item.checkGroup;

            const check = this._isChecked(item);
            el.innerHTML = `<span class="menu-text">${item.text}</span>${check ? '<span class="menu-check">✓</span>' : ''}`;

            el.addEventListener('click', (e) => {
                e.stopPropagation();
                this._handleSubmenuClick(item, submenu);
            });

            submenu.appendChild(el);
        }

        submenu.addEventListener('mousedown', (e) => e.preventDefault());

        this.container.appendChild(submenu);
        return submenu;
    }

    /** @private */
    _isChecked(item) {
        if (!item.checkGroup) return false;
        switch (item.checkGroup) {
            case 'boost': return this.settings.framerateBoostMode === item.value;
            case 'framerate': return this.settings.targetFramerate === item.value;
            case 'bitrate': return this.settings.preferredMinBitrate === item.value;
        }
        return false;
    }

    /** @private */
    _updateSubmenuChecks(submenu) {
        submenu.querySelectorAll('.menu-item[data-check-group]').forEach(el => {
            const group = el.dataset.checkGroup;
            let value = el.dataset.value;
            if (group === 'framerate' || group === 'bitrate') value = Number(value);
            const checked = this._isChecked({ checkGroup: group, value });
            const checkEl = el.querySelector('.menu-check');
            if (checked && !checkEl) {
                el.innerHTML += '<span class="menu-check">✓</span>';
            } else if (!checked && checkEl) {
                checkEl.remove();
            }
        });
    }

    /** @private */
    _handleMenuClick(action, triggerEl) {
        this._closeActiveSubmenu();

        if (action.startsWith('submenu-')) {
            const key = action.replace('submenu-', '');
            const submenu = this._submenus[key];
            if (!submenu) return;

            if (key === 'virtualDisplay') {
                this._refreshVirtualDisplaySubmenu();
            } else if (key === 'resolution') {
                const origItem = submenu.querySelector('#resOriginal .menu-text');
                if (origItem) {
                    origItem.textContent = this._originalWidth > 0
                        ? `${t('menu.original')} (${this._originalWidth}×${this._originalHeight})`
                        : t('menu.original');
                }
            }

            this._updateSubmenuChecks(submenu);
            this._showSubmenu(submenu, triggerEl);
            return;
        }

        switch (action) {
            case 'disconnect':
                this.dispatchEvent(new CustomEvent('action', { detail: { action: 'disconnect' } }));
                this._hideMenu();
                break;
            case 'fitWindow':
                this.dispatchEvent(new CustomEvent('action', { detail: { action: 'fitWindow' } }));
                this._hideMenu();
                break;
            case 'toggleStats':
                this.settings.statsVisible = !this.settings.statsVisible;
                this.dispatchEvent(new CustomEvent('settingChange', {
                    detail: { setting: 'stats', value: this.settings.statsVisible }
                }));
                this._hideMenu();
                break;
            case 'toggleAudio':
                this.settings.audioEnabled = !this.settings.audioEnabled;
                const menuItem = this._menuElement.querySelector('#audioMenuItem');
                if (menuItem) {
                    const icon = this.settings.audioEnabled ? '🔊' : '🔇';
                    const label = this.settings.audioEnabled ? t('menu.muteAudio') : t('menu.unmuteAudio');
                    menuItem.innerHTML = `<span class="menu-icon">${icon}</span><span class="menu-text">${label}</span>`;
                }
                this.dispatchEvent(new CustomEvent('settingChange', {
                    detail: { setting: 'audio', value: this.settings.audioEnabled }
                }));
                break;
            case 'screenshot':
                this.dispatchEvent(new CustomEvent('action', { detail: { action: 'screenshot' } }));
                this._hideMenu();
                break;
            case 'toggleLogs':
                this.dispatchEvent(new CustomEvent('action', { detail: { action: 'toggleLogs' } }));
                this._hideMenu();
                break;
            case 'sendAttentionSequence':
            case 'lockWorkstation':
                this.dispatchEvent(new CustomEvent('action', { detail: { action } }));
                this._hideMenu();
                break;
            case 'togglePrivacyScreen':
                this._privacyScreenActive = !this._privacyScreenActive;
                this._updatePrivacyScreenLabel();
                this.dispatchEvent(new CustomEvent('action', { 
                    detail: { action: this._privacyScreenActive ? 'enablePrivacyScreen' : 'disablePrivacyScreen' } 
                }));
                this._hideMenu();
                break;
            case 'uploadFile':
                this.dispatchEvent(new CustomEvent('action', { detail: { action: 'uploadFile' } }));
                this._hideMenu();
                break;
            case 'downloadFile':
                this.dispatchEvent(new CustomEvent('action', { detail: { action: 'downloadFile' } }));
                this._hideMenu();
                break;
            case 'showTransfers':
                this.dispatchEvent(new CustomEvent('action', { detail: { action: 'showTransfers' } }));
                this._hideMenu();
                break;
        }
    }

    /** @private */
    _handleSubmenuClick(item, submenu) {
        switch (item.checkGroup || '') {
            case 'boost':
                this.settings.framerateBoostMode = item.value;
                this.dispatchEvent(new CustomEvent('settingChange', {
                    detail: { setting: 'framerateBoost', value: item.value }
                }));
                break;
            case 'framerate':
                this.settings.targetFramerate = item.value;
                this.dispatchEvent(new CustomEvent('settingChange', {
                    detail: { setting: 'framerate', value: item.value }
                }));
                break;
            case 'bitrate':
                this.settings.preferredMinBitrate = item.value;
                this.dispatchEvent(new CustomEvent('settingChange', {
                    detail: { setting: 'bitrate', value: item.value }
                }));
                break;
            default:
                this.dispatchEvent(new CustomEvent('settingChange', {
                    detail: { setting: 'resolution', value: item.value }
                }));
                break;
        }
        this._hideMenu();
    }

    /** @private */
    _createDisplayHeader() {
        const header = document.createElement('div');
        header.className = 'display-header';
        header.style.display = 'none';
        return header;
    }

    updateDisplayList(displays, activeIndex) {
        this._displayList = displays || [];
        this._activeDisplayIndex = activeIndex;
        this._refreshDisplayHeader();
    }

    /** @private */
    _refreshDisplayHeader() {
        if (!this._displayHeader) return;
        this._displayHeader.innerHTML = '';

        if (this._displayList.length <= 1) {
            this._displayHeader.style.display = 'none';
            return;
        }

        const label = document.createElement('div');
        label.className = 'display-header-label';
        label.textContent = t('menu.displays');
        this._displayHeader.appendChild(label);

        const strip = document.createElement('div');
        strip.className = 'display-header-strip';

        for (let i = 0; i < this._displayList.length; i++) {
            const tile = document.createElement('div');
            tile.className = 'display-tile' + (i === this._activeDisplayIndex ? ' active' : '');
            tile.textContent = String(i + 1);
            tile.title = this._displayList[i].displayName ||
                `${this._displayList[i].width}x${this._displayList[i].height}`;
            tile.addEventListener('click', (e) => {
                e.stopPropagation();
                if (i !== this._activeDisplayIndex) {
                    this.dispatchEvent(new CustomEvent('action', {
                        detail: { action: 'selectDisplay', index: i }
                    }));
                    this._hideMenu();
                }
            });
            strip.appendChild(tile);
        }

        this._displayHeader.appendChild(strip);
        this._displayHeader.style.display = 'block';
    }

    /** @private */
    _createVirtualDisplaySubmenu() {
        const submenu = document.createElement('div');
        submenu.className = 'floating-submenu';
        submenu.style.display = 'none';
        submenu.addEventListener('mousedown', (e) => e.preventDefault());
        this.container.appendChild(submenu);
        return submenu;
    }

    updateVirtualDisplays(displays) {
        this._virtualDisplays = (displays || []).filter(d => d.active);
    }

    /** @private */
    _refreshVirtualDisplaySubmenu() {
        const submenu = this._submenus.virtualDisplay;
        if (!submenu) return;
        submenu.innerHTML = '';

        const presets = [
            { w: 1920, h: 1080, hz: 60 },
            { w: 2560, h: 1440, hz: 60 },
            { w: 3840, h: 2160, hz: 60 },
            { w: 1280, h: 720, hz: 60 },
        ];

        for (const p of presets) {
            const el = document.createElement('div');
            el.className = 'menu-item';
            el.innerHTML = `<span class="menu-text">${t('menu.addDisplay', { resolution: `${p.w}×${p.h}`, hz: p.hz })}</span>`;
            el.addEventListener('click', (e) => {
                e.stopPropagation();
                this.dispatchEvent(new CustomEvent('action', {
                    detail: { action: 'createVirtualDisplay', width: p.w, height: p.h, refreshRate: p.hz }
                }));
                this._hideMenu();
            });
            submenu.appendChild(el);
        }

        if (this._virtualDisplays.length > 0) {
            const sep = document.createElement('div');
            sep.className = 'menu-separator';
            submenu.appendChild(sep);

            for (const vd of this._virtualDisplays) {
                const el = document.createElement('div');
                el.className = 'menu-item';
                el.innerHTML = `<span class="menu-text">${t('menu.removeDisplay', { index: vd.index, resolution: `${vd.width}×${vd.height}` })}</span>`;
                el.addEventListener('click', (e) => {
                    e.stopPropagation();
                    this.dispatchEvent(new CustomEvent('action', {
                        detail: { action: 'removeVirtualDisplay', index: vd.index }
                    }));
                    this._hideMenu();
                });
                submenu.appendChild(el);
            }

            const sep2 = document.createElement('div');
            sep2.className = 'menu-separator';
            submenu.appendChild(sep2);

            const removeAllEl = document.createElement('div');
            removeAllEl.className = 'menu-item';
            removeAllEl.innerHTML = `<span class="menu-text">${t('menu.removeAllDisplays')}</span>`;
            removeAllEl.addEventListener('click', (e) => {
                e.stopPropagation();
                this.dispatchEvent(new CustomEvent('action', {
                    detail: { action: 'removeAllVirtualDisplays' }
                }));
                this._hideMenu();
            });
            submenu.appendChild(removeAllEl);
        }
    }

    // ==================== Submenu positioning ====================

    /** @private */
    _showSubmenu(submenu, triggerEl) {
        this._activeSubmenu = submenu;
        submenu.style.display = 'block';

        const menuRect = this._menuElement.getBoundingClientRect();
        const triggerRect = triggerEl.getBoundingClientRect();
        const containerRect = this.container.getBoundingClientRect();

        let left = menuRect.right + 4 - containerRect.left;
        let top = triggerRect.top - containerRect.top;

        if (left + submenu.offsetWidth > containerRect.width) {
            left = menuRect.left - containerRect.left - submenu.offsetWidth - 4;
        }
        if (left < 0) left = 4;

        if (top + submenu.offsetHeight > containerRect.height) {
            top = containerRect.height - submenu.offsetHeight - 4;
        }
        if (top < 0) top = 4;

        submenu.style.left = `${left}px`;
        submenu.style.top = `${top}px`;
    }

    /** @private */
    _closeActiveSubmenu() {
        if (this._activeSubmenu) {
            this._activeSubmenu.style.display = 'none';
            this._activeSubmenu = null;
        }
    }

    // ==================== Drag ====================

    _startDrag(e) {
        this._isDragging = false;
        this._dragStart = { x: e.clientX, y: e.clientY };
        this._dragOffset = {
            x: e.clientX - this._element.offsetLeft,
            y: e.clientY - this._element.offsetTop,
        };
        this._maybeDragging = true;
    }

    _onDrag(e) {
        if (!this._maybeDragging) return;
        const dx = Math.abs(e.clientX - this._dragStart.x);
        const dy = Math.abs(e.clientY - this._dragStart.y);
        if (dx > 3 || dy > 3) this._isDragging = true;
        if (this._isDragging) {
            const x = e.clientX - this._dragOffset.x;
            const y = e.clientY - this._dragOffset.y;
            this._element.style.left = `${Math.max(0, x)}px`;
            this._element.style.top = `${Math.max(0, y)}px`;
            this._element.style.right = 'auto';
            this._hideMenu();
        }
    }

    _endDrag() {
        this._maybeDragging = false;
        setTimeout(() => { this._isDragging = false; }, 100);
    }

    // ==================== Menu ====================

    _toggleMenu() {
        if (this._menuVisible) {
            this._hideMenu();
        } else {
            this._showMenu();
        }
    }

    _showMenu() {
        this._menuVisible = true;
        this._refreshDisplayHeader();

        if (this._displayHeader) {
            if (this._displayHeader.parentNode !== this._menuElement) {
                this._menuElement.insertBefore(this._displayHeader, this._menuElement.firstChild);
            }
        }

        const btnRect = this._element.getBoundingClientRect();
        const containerRect = this.container.getBoundingClientRect();

        this._menuElement.style.display = 'block';

        const menuW = this._menuElement.offsetWidth;
        const menuH = this._menuElement.offsetHeight;

        let top = btnRect.bottom - containerRect.top + 8;
        let left = btnRect.right - containerRect.left - menuW;

        if (left < 4) left = 4;
        if (top + menuH > containerRect.height) {
            top = btnRect.top - containerRect.top - menuH - 8;
        }
        if (top < 4) top = 4;

        this._menuElement.style.top = `${top}px`;
        this._menuElement.style.left = `${left}px`;
        this._menuElement.style.right = 'auto';
    }

    _hideMenu() {
        this._menuVisible = false;
        this._closeActiveSubmenu();
        this._menuElement.style.display = 'none';
    }

    setActionSupport(supportsSAS, supportsLock, supportsFileTransfer = false, supportsPrivacyScreen = false, supportsVirtualDisplay = false) {
        this._supportsSAS = supportsSAS;
        this._supportsLock = supportsLock;
        this._supportsFileTransfer = supportsFileTransfer;
        this._supportsPrivacyScreen = supportsPrivacyScreen;
        this._supportsVirtualDisplay = supportsVirtualDisplay;

        const sasItem = this._menuElement.querySelector('#sasMenuItem');
        const lockItem = this._menuElement.querySelector('#lockMenuItem');
        const actionSep = this._menuElement.querySelector('#actionSeparator');
        const uploadItem = this._menuElement.querySelector('#uploadMenuItem');
        const uploadSep = this._menuElement.querySelector('#uploadSeparator');
        const privacyItem = this._menuElement.querySelector('#privacyScreenMenuItem');
        const vdItem = this._menuElement.querySelector('#vdMenuItem');
        const vdSep = this._menuElement.querySelector('#vdSeparator');

        if (sasItem) sasItem.style.display = supportsSAS ? '' : 'none';
        if (lockItem) lockItem.style.display = supportsLock ? '' : 'none';
        if (privacyItem) privacyItem.style.display = supportsPrivacyScreen ? '' : 'none';
        if (actionSep) actionSep.style.display = 
            (supportsSAS || supportsLock || supportsPrivacyScreen) ? '' : 'none';
        if (uploadItem) uploadItem.style.display = supportsFileTransfer ? '' : 'none';
        if (uploadSep) uploadSep.style.display = supportsFileTransfer ? '' : 'none';
        if (vdItem) vdItem.style.display = supportsVirtualDisplay ? '' : 'none';
        if (vdSep) vdSep.style.display = supportsVirtualDisplay ? '' : 'none';

        const downloadItem = this._menuElement.querySelector('#downloadMenuItem');
        if (downloadItem) downloadItem.style.display = supportsFileTransfer ? '' : 'none';

        const transfersItem = this._menuElement.querySelector('#transfersMenuItem');
        if (transfersItem) transfersItem.style.display = supportsFileTransfer ? '' : 'none';
    }

    updateTransferCount(count) {
        const transfersItem = this._menuElement.querySelector('#transfersMenuItem');
        if (transfersItem) {
            const textEl = transfersItem.querySelector('.menu-text');
            if (textEl) {
                textEl.textContent = count > 0
                    ? t('menu.transfersCount', { count })
                    : t('menu.transfers');
            }
            transfersItem.style.display = (this._supportsFileTransfer && count > 0) ? '' : 'none';
        }
    }

    refreshI18n() {
        if (this._submenus) {
            Object.values(this._submenus).forEach(s => s.remove());
        }
        if (this._displayHeader) {
            this._displayHeader.remove();
        }
        this._hideMenu();
        this._buildMenu();
        this.setActionSupport(this._supportsSAS, this._supportsLock, this._supportsFileTransfer, this._supportsPrivacyScreen, this._supportsVirtualDisplay);
    }

    /** @private */
    _updatePrivacyScreenLabel() {
        const item = this._menuElement.querySelector('#privacyScreenMenuItem');
        if (!item) return;
        const icon = this._privacyScreenActive ? '👁‍🗨' : '👁';
        const textKey = this._privacyScreenActive ? 'menu.disablePrivacyScreen' : 'menu.privacyScreen';
        item.innerHTML = `<span class="menu-icon">${icon}</span><span class="menu-text">${t(textKey)}</span>`;
    }

    setVisible(visible) {
        this._element.style.display = visible ? 'flex' : 'none';
        if (!visible) this._hideMenu();
    }

    destroy() {
        if (this._element) this._element.remove();
        if (this._menuElement) this._menuElement.remove();
        if (this._submenus) {
            Object.values(this._submenus).forEach(s => s.remove());
        }
        if (this._displayHeader) this._displayHeader.remove();
    }
}
