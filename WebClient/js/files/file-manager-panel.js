import { t } from '../i18n.js';
import {
    breadcrumbs,
    formatFileSize,
    joinRemotePath,
} from './file-manager-model.js';

export class FileManagerPanel {
    constructor(container, manager) {
        this.container = container;
        this.manager = manager;
        this._initialized = false;
        this._open = false;
        this._build();

        this.manager.addEventListener('listingChanged', (event) => {
            this._renderBreadcrumbs(event.detail.path);
            this._renderEntries(event.detail.entries);
            this._pathInput.value = event.detail.path;
            this._setStatus(t('files.itemCount', { count: event.detail.entries.length }));
        });
        this.manager.addEventListener('busyChanged', (event) => {
            this._setBusy(event.detail.busy);
        });
        this.manager.addEventListener('uploadProgress', (event) => {
            const { filename, written, total } = event.detail;
            const percent = total > 0 ? Math.round((written / total) * 100) : 100;
            this._setStatus(t('files.uploading', { name: filename, percent }));
        });
    }

    async open() {
        this._root.hidden = false;
        this._open = true;
        this._closeButton.focus();
        if (this._initialized) return;
        await this._run(async () => {
            await this.manager.initialize();
        });
        this._initialized = Boolean(this.manager.currentPath);
    }

    close() {
        this._root.hidden = true;
        this._open = false;
        this._hidePrompt();
    }

    refreshI18n() {
        this._title.textContent = t('files.title');
        this._closeButton.textContent = '×';
        this._closeButton.title = t('files.close');
        this._upButton.textContent = t('files.up');
        this._pathInput.placeholder = t('files.pathPlaceholder');
        this._goButton.textContent = t('files.go');
        this._newFolderButton.textContent = t('files.newFolder');
        this._uploadButton.textContent = t('files.upload');
        this._refreshButton.textContent = t('files.refresh');
        this._nameHeader.textContent = t('files.name');
        this._sizeHeader.textContent = t('files.size');
        this._actionsHeader.textContent = t('files.actions');
        this._previewTitle.textContent = t('files.preview');
        this._previewClose.title = t('files.close');
        this._promptCancel.textContent = t('files.cancel');
        this._promptConfirm.textContent = t('files.confirm');
        this._renderBreadcrumbs(this.manager.currentPath);
        this._renderEntries(this.manager.entries);
        if (!this.manager.currentPath) this._setStatus(t('files.loading'));
    }

    _build() {
        const root = document.createElement('div');
        root.className = 'file-manager-root';
        root.hidden = true;
        root.innerHTML = `
            <div class="file-manager-backdrop"></div>
            <section class="file-manager-panel" role="dialog" aria-modal="false">
                <header class="file-manager-header">
                    <div>
                        <div class="file-manager-title"></div>
                        <div class="file-manager-breadcrumbs"></div>
                    </div>
                    <button type="button" class="file-manager-icon-button file-manager-close"></button>
                </header>
                <div class="file-manager-pathbar">
                    <button type="button" class="file-manager-button file-manager-up"></button>
                    <input class="file-manager-path" type="text" spellcheck="false">
                    <button type="button" class="file-manager-button file-manager-go"></button>
                </div>
                <div class="file-manager-toolbar">
                    <button type="button" class="file-manager-button file-manager-new-folder"></button>
                    <button type="button" class="file-manager-button file-manager-upload"></button>
                    <button type="button" class="file-manager-button file-manager-refresh"></button>
                    <span class="file-manager-status"></span>
                </div>
                <div class="file-manager-content">
                    <div class="file-manager-list-wrap">
                        <div class="file-manager-list-header">
                            <span class="file-manager-name-header"></span>
                            <span class="file-manager-size-header"></span>
                            <span class="file-manager-actions-header"></span>
                        </div>
                        <div class="file-manager-list" role="list"></div>
                    </div>
                    <aside class="file-manager-preview" hidden>
                        <div class="file-manager-preview-header">
                            <span class="file-manager-preview-title"></span>
                            <button type="button" class="file-manager-icon-button file-manager-preview-close">&times;</button>
                        </div>
                        <div class="file-manager-preview-path"></div>
                        <pre class="file-manager-preview-text"></pre>
                    </aside>
                </div>
                <div class="file-manager-dialog" hidden>
                    <form class="file-manager-dialog-card">
                        <label class="file-manager-dialog-label"></label>
                        <input class="file-manager-dialog-input" type="text" autocomplete="off">
                        <div class="file-manager-dialog-actions">
                            <button type="button" class="file-manager-button file-manager-dialog-cancel"></button>
                            <button type="submit" class="file-manager-button primary file-manager-dialog-confirm"></button>
                        </div>
                    </form>
                </div>
            </section>
        `;
        this.container.appendChild(root);

        this._root = root;
        this._title = root.querySelector('.file-manager-title');
        this._breadcrumbBar = root.querySelector('.file-manager-breadcrumbs');
        this._closeButton = root.querySelector('.file-manager-close');
        this._upButton = root.querySelector('.file-manager-up');
        this._pathInput = root.querySelector('.file-manager-path');
        this._goButton = root.querySelector('.file-manager-go');
        this._newFolderButton = root.querySelector('.file-manager-new-folder');
        this._uploadButton = root.querySelector('.file-manager-upload');
        this._refreshButton = root.querySelector('.file-manager-refresh');
        this._status = root.querySelector('.file-manager-status');
        this._list = root.querySelector('.file-manager-list');
        this._nameHeader = root.querySelector('.file-manager-name-header');
        this._sizeHeader = root.querySelector('.file-manager-size-header');
        this._actionsHeader = root.querySelector('.file-manager-actions-header');
        this._preview = root.querySelector('.file-manager-preview');
        this._previewTitle = root.querySelector('.file-manager-preview-title');
        this._previewPathEl = root.querySelector('.file-manager-preview-path');
        this._previewText = root.querySelector('.file-manager-preview-text');
        this._previewClose = root.querySelector('.file-manager-preview-close');
        this._dialog = root.querySelector('.file-manager-dialog');
        this._dialogForm = root.querySelector('.file-manager-dialog-card');
        this._dialogLabel = root.querySelector('.file-manager-dialog-label');
        this._dialogInput = root.querySelector('.file-manager-dialog-input');
        this._promptCancel = root.querySelector('.file-manager-dialog-cancel');
        this._promptConfirm = root.querySelector('.file-manager-dialog-confirm');
        this._uploadInput = document.createElement('input');
        this._uploadInput.type = 'file';
        this._uploadInput.multiple = true;
        this._uploadInput.hidden = true;
        root.appendChild(this._uploadInput);

        this._closeButton.addEventListener('click', () => this.close());
        root.querySelector('.file-manager-backdrop').addEventListener('click', () => this.close());
        this._upButton.addEventListener('click', () => this._run(() => this.manager.up()));
        this._goButton.addEventListener('click', () => this._navigateFromInput());
        this._pathInput.addEventListener('keydown', (event) => {
            if (event.key === 'Enter') this._navigateFromInput();
        });
        this._refreshButton.addEventListener('click', () => this._run(() => this.manager.refresh()));
        this._newFolderButton.addEventListener('click', () => this._createDirectory());
        this._uploadButton.addEventListener('click', () => this._uploadInput.click());
        this._uploadInput.addEventListener('change', () => this._uploadSelected());
        this._previewClose.addEventListener('click', () => this._hidePreview());
        this._promptCancel.addEventListener('click', () => this._hidePrompt());
        this._dialogForm.addEventListener('submit', (event) => {
            event.preventDefault();
            const resolve = this._promptResolve;
            const value = this._dialogInput.value;
            this._promptResolve = null;
            this._dialog.hidden = true;
            resolve?.(value);
        });
        root.addEventListener('keydown', (event) => {
            if (event.key === 'Escape') {
                if (!this._dialog.hidden) this._hidePrompt();
                else this.close();
            }
        });

        this.refreshI18n();
    }

    async _navigateFromInput() {
        const path = this._pathInput.value.trim();
        if (!path) return;
        await this._run(() => this.manager.navigate(path));
    }

    async _createDirectory() {
        const name = await this._askName(t('files.newFolderPrompt'));
        if (name === null) return;
        await this._run(
            () => this.manager.createDirectory(name),
            t('files.created', { name }),
        );
    }

    async _rename(entry) {
        const name = await this._askName(t('files.renamePrompt'), entry.name);
        if (name === null || name === entry.name) return;
        await this._run(
            () => this.manager.rename(entry, name),
            t('files.renamed', { name }),
        );
    }

    async _trash(entry) {
        if (!window.confirm(t('files.deleteConfirm', { name: entry.name }))) return;
        await this._run(
            () => this.manager.trash(entry),
            t('files.trashed', { name: entry.name }),
        );
    }

    async _previewFile(entry) {
        await this._run(async () => {
            const preview = await this.manager.readTextPreview(entry);
            this._previewPathEl.textContent = preview.path;
            this._previewText.textContent = preview.truncated
                ? `${preview.text}\n\n${t('files.previewTruncated')}`
                : preview.text;
            this._preview.hidden = false;
            this._previewPathEl.title = preview.path;
            this._previewText.scrollTop = 0;
        });
    }

    async _download(entry) {
        await this._run(async () => {
            const result = await this.manager.download(entry);
            const blob = new Blob([result.bytes], { type: 'application/octet-stream' });
            if (window.showSaveFilePicker) {
                try {
                    const handle = await window.showSaveFilePicker({
                        suggestedName: result.filename,
                    });
                    const writable = await handle.createWritable();
                    await writable.write(blob);
                    await writable.close();
                    this._setStatus(t('files.downloaded', { name: result.filename }));
                    return;
                } catch (error) {
                    if (error.name === 'AbortError') return;
                }
            }
            const url = URL.createObjectURL(blob);
            const link = document.createElement('a');
            link.href = url;
            link.download = result.filename;
            document.body.appendChild(link);
            link.click();
            link.remove();
            URL.revokeObjectURL(url);
            this._setStatus(t('files.downloaded', { name: result.filename }));
        });
    }

    async _uploadSelected() {
        const files = [...this._uploadInput.files];
        this._uploadInput.value = '';
        if (files.length === 0) return;
        await this._run(async () => {
            for (const file of files) {
                await this.manager.upload(file, this.manager.currentPath);
            }
            this._setStatus(t('files.uploadedCount', { count: files.length }));
        });
    }

    _renderBreadcrumbs(path) {
        this._breadcrumbBar.replaceChildren();
        const items = breadcrumbs(path);
        items.forEach((item, index) => {
            const button = document.createElement('button');
            button.type = 'button';
            button.className = 'file-manager-crumb';
            button.textContent = item.label;
            button.title = item.path;
            button.addEventListener('click', () => this._run(() => this.manager.navigate(item.path)));
            this._breadcrumbBar.appendChild(button);
            if (index < items.length - 1) {
                const separator = document.createElement('span');
                separator.className = 'file-manager-crumb-separator';
                separator.textContent = '›';
                this._breadcrumbBar.appendChild(separator);
            }
        });
    }

    _renderEntries(entries) {
        this._list.replaceChildren();
        if (!entries || entries.length === 0) {
            const empty = document.createElement('div');
            empty.className = 'file-manager-empty';
            empty.textContent = t('files.empty');
            this._list.appendChild(empty);
            return;
        }

        for (const entry of entries) {
            const row = document.createElement('div');
            row.className = 'file-manager-entry';
            row.setAttribute('role', 'listitem');

            const name = document.createElement('button');
            name.type = 'button';
            name.className = 'file-manager-entry-name';
            const icon = document.createElement('span');
            icon.className = 'file-manager-entry-icon';
            icon.textContent = entry.isDirectory ? '▸' : '•';
            const text = document.createElement('span');
            text.className = 'file-manager-entry-text';
            text.textContent = entry.name;
            text.title = joinRemotePath(this.manager.currentPath, entry.name);
            name.append(icon, text);
            name.addEventListener('click', () => {
                if (entry.isDirectory) {
                    this._run(() => this.manager.navigate(joinRemotePath(
                        this.manager.currentPath,
                        entry.name,
                    )));
                } else {
                    this._previewFile(entry);
                }
            });

            const size = document.createElement('span');
            size.className = 'file-manager-entry-size';
            size.textContent = entry.isDirectory ? '' : formatFileSize(entry.size);

            const actions = document.createElement('span');
            actions.className = 'file-manager-entry-actions';
            actions.append(
                this._actionButton('✎', t('files.rename'), () => this._rename(entry)),
            );
            if (!entry.isDirectory) {
                actions.append(
                    this._actionButton('↓', t('files.download'), () => this._download(entry)),
                );
            }
            actions.append(
                this._actionButton('×', t('files.trash'), () => this._trash(entry), true),
            );

            row.append(name, size, actions);
            this._list.appendChild(row);
        }
    }

    _actionButton(label, title, handler, destructive = false) {
        const button = document.createElement('button');
        button.type = 'button';
        button.className = `file-manager-action${destructive ? ' destructive' : ''}`;
        button.textContent = label;
        button.title = title;
        button.setAttribute('aria-label', title);
        button.addEventListener('click', handler);
        return button;
    }

    _askName(title, initialValue = '') {
        this._dialogLabel.textContent = title;
        this._dialogInput.value = initialValue;
        this._dialog.hidden = false;
        this._dialogInput.focus();
        this._dialogInput.select();
        return new Promise((resolve) => {
            this._promptResolve = resolve;
        });
    }

    _hidePrompt() {
        this._dialog.hidden = true;
        const resolve = this._promptResolve;
        this._promptResolve = null;
        if (resolve) resolve(null);
    }

    _hidePreview() {
        this._preview.hidden = true;
        this._previewPathEl.textContent = '';
        this._previewText.textContent = '';
    }

    async _run(task, successMessage = '') {
        try {
            await task();
            if (successMessage) this._setStatus(successMessage);
        } catch (error) {
            this._setStatus(error.message || String(error), true);
        }
    }

    _setBusy(busy) {
        this._root.classList.toggle('busy', busy);
        [this._upButton, this._goButton, this._newFolderButton,
            this._uploadButton, this._refreshButton].forEach(button => {
            button.disabled = busy;
        });
        if (busy) this._setStatus(t('files.loading'));
    }

    _setStatus(message, error = false) {
        this._status.textContent = message;
        this._status.classList.toggle('error', error);
    }
}
