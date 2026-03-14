/**
 * floating-toolbar.js - 浮动工具栏
 * 
 * 参照 FloatingToolButton.qml 实现一致的菜单结构
 * 可拖拽浮动按钮 + 下拉菜单（含子菜单）
 */

export class FloatingToolbar extends EventTarget {
    /**
     * @param {HTMLElement} container - 容器元素
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
            preferredMinBitrate: 10485760, // 10 MiB (10*1024*1024) bps
            audioEnabled: true,
            statsVisible: false,
        };

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

    /**
     * @private
     */
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

    /**
     * @private
     */
    _buildMenu() {
        this._menuElement.innerHTML = '';

        const items = [
            { text: 'Smart Boost', icon: '⚡', hasSubmenu: true, action: 'submenu-boost' },
            { text: 'Target Framerate', icon: '🎯', hasSubmenu: true, action: 'submenu-framerate' },
            { text: 'Resolution', icon: '🖥️', hasSubmenu: true, action: 'submenu-resolution' },
            { text: 'Bitrate', icon: '📶', hasSubmenu: true, action: 'submenu-bitrate' },
            { type: 'separator' },
            { text: 'Fit Window', icon: '⛶', action: 'fitWindow' },
            { text: 'Video Stats', icon: '📊', action: 'toggleStats' },
            { text: '', icon: '🔊', action: 'toggleAudio', id: 'audioMenuItem' },
            { text: 'Screenshot', icon: '📷', action: 'screenshot' },
            { text: 'Logs', icon: '📋', action: 'toggleLogs' },
            { type: 'separator' },
            { text: 'Disconnect', icon: '✕', action: 'disconnect', destructive: true },
        ];

        for (const item of items) {
            if (item.type === 'separator') {
                const sep = document.createElement('div');
                sep.className = 'menu-separator';
                this._menuElement.appendChild(sep);
                continue;
            }

            const el = document.createElement('div');
            el.className = 'menu-item' + (item.destructive ? ' destructive' : '');
            if (item.id) el.id = item.id;

            let label = item.text;
            if (item.action === 'toggleAudio') {
                label = this.settings.audioEnabled ? 'Mute Audio' : 'Unmute Audio';
                el.innerHTML = `<span class="menu-icon">${this.settings.audioEnabled ? '🔊' : '🔇'}</span><span class="menu-text">${label}</span>`;
            } else {
                el.innerHTML = `<span class="menu-icon">${item.icon}</span><span class="menu-text">${label}</span>`;
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

    /**
     * @private
     */
    _createSubmenus() {
        this._submenus = {};

        // Smart Boost
        this._submenus.boost = this._createSubmenu([
            { text: 'Off', value: 'off', checkGroup: 'boost' },
            { text: 'Office', value: 'office', checkGroup: 'boost' },
            { text: 'Gaming', value: 'gaming', checkGroup: 'boost' },
        ]);

        // Target Framerate
        this._submenus.framerate = this._createSubmenu([
            { text: '60 FPS', value: 60, checkGroup: 'framerate' },
            { text: '30 FPS', value: 30, checkGroup: 'framerate' },
            { text: '15 FPS', value: 15, checkGroup: 'framerate' },
            { text: '5 FPS', value: 5, checkGroup: 'framerate' },
        ]);

        // Resolution
        this._submenus.resolution = this._createSubmenu([
            { text: 'Original', value: 'original', id: 'resOriginal' },
            { type: 'separator' },
            { text: '3840 × 2160 (4K)', value: '3840x2160' },
            { text: '2560 × 1440 (2K)', value: '2560x1440' },
            { text: '1920 × 1080 (FHD)', value: '1920x1080' },
            { text: '1600 × 900', value: '1600x900' },
            { text: '1366 × 768', value: '1366x768' },
            { text: '1280 × 720', value: '1280x720' },
            { text: '1024 × 768', value: '1024x768' },
        ]);

        // Bitrate
        this._submenus.bitrate = this._createSubmenu([
            { text: '100 MiB', value: 104857600, checkGroup: 'bitrate' },
            { text: '50 MiB', value: 52428800, checkGroup: 'bitrate' },
            { text: '10 MiB', value: 10485760, checkGroup: 'bitrate' },
            { text: '5 MiB', value: 5242880, checkGroup: 'bitrate' },
            { text: '2 MiB', value: 2097152, checkGroup: 'bitrate' },
        ]);
    }

    /**
     * @private
     */
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

    /**
     * @private
     */
    _isChecked(item) {
        if (!item.checkGroup) return false;
        switch (item.checkGroup) {
            case 'boost': return this.settings.framerateBoostMode === item.value;
            case 'framerate': return this.settings.targetFramerate === item.value;
            case 'bitrate': return this.settings.preferredMinBitrate === item.value;
        }
        return false;
    }

    /**
     * @private
     */
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

    /**
     * @private
     */
    _handleMenuClick(action, triggerEl) {
        this._closeActiveSubmenu();

        if (action.startsWith('submenu-')) {
            const key = action.replace('submenu-', '');
            const submenu = this._submenus[key];
            if (!submenu) return;

            if (key === 'resolution') {
                const origItem = submenu.querySelector('#resOriginal .menu-text');
                if (origItem) {
                    origItem.textContent = this._originalWidth > 0
                        ? `Original (${this._originalWidth}×${this._originalHeight})`
                        : 'Original';
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
                    const text = this.settings.audioEnabled ? 'Mute Audio' : 'Unmute Audio';
                    menuItem.innerHTML = `<span class="menu-icon">${icon}</span><span class="menu-text">${text}</span>`;
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
        }
    }

    /**
     * @private
     */
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
                // Resolution items (no checkGroup)
                this.dispatchEvent(new CustomEvent('settingChange', {
                    detail: { setting: 'resolution', value: item.value }
                }));
                break;
        }
        this._hideMenu();
    }

    // ==================== Submenu positioning ====================

    /**
     * @private
     */
    _showSubmenu(submenu, triggerEl) {
        this._activeSubmenu = submenu;
        submenu.style.display = 'block';

        const menuRect = this._menuElement.getBoundingClientRect();
        const triggerRect = triggerEl.getBoundingClientRect();
        const containerRect = this.container.getBoundingClientRect();

        let left = menuRect.right + 4 - containerRect.left;
        let top = triggerRect.top - containerRect.top;

        // If goes off right edge, show on left side
        if (left + submenu.offsetWidth > containerRect.width) {
            left = menuRect.left - containerRect.left - submenu.offsetWidth - 4;
        }
        if (left < 0) left = 4;

        // If goes off bottom
        if (top + submenu.offsetHeight > containerRect.height) {
            top = containerRect.height - submenu.offsetHeight - 4;
        }
        if (top < 0) top = 4;

        submenu.style.left = `${left}px`;
        submenu.style.top = `${top}px`;
    }

    /**
     * @private
     */
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
    }
}
