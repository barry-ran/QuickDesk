/**
 * Browser client for the host-side skill bridge.
 *
 * The WebClient calls this through the qd-skill-bridge extension message. The
 * host forwards each call to quickdesk-skill-host and returns the response with
 * the same request id.
 */
export class SkillBridgeClient extends EventTarget {
    constructor(send, { timeoutMs = 30000, idPrefix = 'web-skill' } = {}) {
        super();
        if (typeof send !== 'function') {
            throw new TypeError('send must be a function');
        }
        this._send = send;
        this._timeoutMs = timeoutMs;
        this._idPrefix = idPrefix;
        this._nextId = 1;
        this._pending = new Map();
        this._tools = new Set();
        this._random = Math.random().toString(36).slice(2, 10);
    }

    get pendingCount() {
        return this._pending.size;
    }

    get tools() {
        return [...this._tools];
    }

    supportsTool(name) {
        return this._tools.has(name);
    }

    async request(tool, args = {}) {
        const id = `${this._idPrefix}-${Date.now().toString(36)}-${this._random}-${this._nextId++}`;
        const payload = {
            id,
            type: 'toolCall',
            tool,
            args,
        };

        const promise = new Promise((resolve, reject) => {
            const timer = setTimeout(() => {
                this._pending.delete(id);
                reject(new Error(`skill request '${tool}' timed out`));
            }, this._timeoutMs);
            this._pending.set(id, { resolve, reject, timer, tool });
        });

        try {
            this._send(JSON.stringify(payload));
        } catch (error) {
            const pending = this._pending.get(id);
            if (pending) {
                clearTimeout(pending.timer);
                this._pending.delete(id);
                pending.reject(new Error(`failed to send skill request: ${error.message || error}`));
            }
        }
        return promise;
    }

    handleMessage(rawMessage) {
        let message = rawMessage;
        if (typeof rawMessage === 'string') {
            try {
                message = JSON.parse(rawMessage);
            } catch {
                return;
            }
        }
        if (!message || typeof message !== 'object') return;

        const type = message.type || '';
        if (type === 'capabilitiesReady') {
            this._tools = new Set(this._toolNames(message.tools));
            this._emitCapabilitiesChanged();
            return;
        }
        if (type === 'capabilitiesChanged') {
            for (const name of this._toolNames(message.removed)) {
                this._tools.delete(name);
            }
            for (const name of this._toolNames(message.added)) {
                this._tools.add(name);
            }
            this._emitCapabilitiesChanged();
            return;
        }

        const id = message.id;
        if (!id) return;
        const pending = this._pending.get(id);
        if (!pending) return;

        clearTimeout(pending.timer);
        this._pending.delete(id);

        if (type === 'toolError' || type === 'error') {
            pending.reject(new Error(message.error || `${pending.tool} failed`));
            return;
        }
        if (type !== 'toolResult') {
            pending.reject(new Error(`unexpected skill response: ${type || '(empty)'}`));
            return;
        }
        pending.resolve(this._parseResult(message.result));
    }

    close(reason = 'skill bridge closed') {
        for (const pending of this._pending.values()) {
            clearTimeout(pending.timer);
            pending.reject(new Error(reason));
        }
        this._pending.clear();
    }

    _toolNames(tools) {
        if (!Array.isArray(tools)) return [];
        return tools
            .map(tool => {
                if (typeof tool === 'string') return tool;
                if (!tool || typeof tool !== 'object') return '';
                return tool.name || tool.tool || '';
            })
            .filter(Boolean);
    }

    _parseResult(result) {
        if (typeof result !== 'string') return result;
        try {
            return JSON.parse(result);
        } catch {
            return result;
        }
    }

    _emitCapabilitiesChanged() {
        this.dispatchEvent(new CustomEvent('capabilitiesChanged', {
            detail: { tools: this.tools },
        }));
    }
}
