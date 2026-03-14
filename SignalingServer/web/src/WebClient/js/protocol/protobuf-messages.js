/**
 * protobuf-messages.js - 手动 Protobuf 编解码器
 * 
 * 实现 Chromium Remoting 使用的 protobuf 消息编解码，
 * 参照 src/remoting/proto/event.proto, control.proto, internal.proto
 * 
 * Wire format:
 *   tag = (field_number << 3) | wire_type
 *   wire_type: 0=varint, 1=64bit, 2=length-delimited, 5=32bit
 */

// ==================== 编码工具 ====================

/** 编码 varint */
function encodeVarint(value) {
    const bytes = [];
    // 处理负数 (int32 的 zigzag 或 two's complement)
    if (value < 0) {
        // protobuf int32 负数用 10 字节 two's complement varint
        const bigVal = BigInt(value) & BigInt('0xFFFFFFFFFFFFFFFF');
        let v = bigVal;
        while (v > 0x7Fn) {
            bytes.push(Number(v & 0x7Fn) | 0x80);
            v >>= 7n;
        }
        bytes.push(Number(v));
        return new Uint8Array(bytes);
    }
    let v = value >>> 0; // ensure unsigned
    while (v > 0x7F) {
        bytes.push((v & 0x7F) | 0x80);
        v >>>= 7;
    }
    bytes.push(v);
    return new Uint8Array(bytes);
}

/** 编码 tag */
function encodeTag(fieldNumber, wireType) {
    return encodeVarint((fieldNumber << 3) | wireType);
}

/** 编码 varint 字段 (int32/uint32/bool/enum) */
function encodeVarintField(fieldNumber, value) {
    if (value === undefined || value === null) return new Uint8Array(0);
    const tag = encodeTag(fieldNumber, 0);
    const val = encodeVarint(typeof value === 'boolean' ? (value ? 1 : 0) : value);
    const result = new Uint8Array(tag.length + val.length);
    result.set(tag, 0);
    result.set(val, tag.length);
    return result;
}

/** 编码 int64 varint 字段 */
function encodeVarint64Field(fieldNumber, value) {
    if (value === undefined || value === null) return new Uint8Array(0);
    const tag = encodeTag(fieldNumber, 0);
    const bytes = [];
    let v = BigInt(value);
    if (v < 0n) {
        v = v & BigInt('0xFFFFFFFFFFFFFFFF');
    }
    while (v > 0x7Fn) {
        bytes.push(Number(v & 0x7Fn) | 0x80);
        v >>= 7n;
    }
    bytes.push(Number(v));
    const val = new Uint8Array(bytes);
    const result = new Uint8Array(tag.length + val.length);
    result.set(tag, 0);
    result.set(val, tag.length);
    return result;
}

/** 编码 float 字段 (wire type 5, fixed32) */
function encodeFloatField(fieldNumber, value) {
    if (value === undefined || value === null) return new Uint8Array(0);
    const tag = encodeTag(fieldNumber, 5);
    const buf = new ArrayBuffer(4);
    new DataView(buf).setFloat32(0, value, true); // little-endian
    const val = new Uint8Array(buf);
    const result = new Uint8Array(tag.length + 4);
    result.set(tag, 0);
    result.set(val, tag.length);
    return result;
}

/** 编码 length-delimited 字段 (bytes/string/embedded message) */
function encodeLengthDelimitedField(fieldNumber, data) {
    if (data === undefined || data === null) return new Uint8Array(0);
    let bytes;
    if (typeof data === 'string') {
        bytes = new TextEncoder().encode(data);
    } else if (data instanceof Uint8Array) {
        bytes = data;
    } else {
        bytes = new Uint8Array(data);
    }
    const tag = encodeTag(fieldNumber, 2);
    const len = encodeVarint(bytes.length);
    const result = new Uint8Array(tag.length + len.length + bytes.length);
    result.set(tag, 0);
    result.set(len, tag.length);
    result.set(bytes, tag.length + len.length);
    return result;
}

/** 合并多个 Uint8Array */
function concatBytes(...arrays) {
    const filtered = arrays.filter(a => a && a.length > 0);
    const totalLength = filtered.reduce((sum, arr) => sum + arr.length, 0);
    const result = new Uint8Array(totalLength);
    let offset = 0;
    for (const arr of filtered) {
        result.set(arr, offset);
        offset += arr.length;
    }
    return result;
}

// ==================== 解码工具 ====================

class ProtobufReader {
    constructor(data) {
        if (data instanceof ArrayBuffer) {
            this.data = new Uint8Array(data);
        } else {
            this.data = new Uint8Array(data);
        }
        this.offset = 0;
    }

    hasMore() {
        return this.offset < this.data.length;
    }

    readVarint() {
        let result = 0;
        let shift = 0;
        while (this.offset < this.data.length) {
            const byte = this.data[this.offset++];
            result |= (byte & 0x7F) << shift;
            if ((byte & 0x80) === 0) {
                return result >>> 0; // unsigned
            }
            shift += 7;
            if (shift >= 35) {
                // Read remaining bytes for larger varints (int64)
                // For now, just consume remaining varint bytes
                while (this.offset < this.data.length && (this.data[this.offset] & 0x80) !== 0) {
                    this.offset++;
                }
                if (this.offset < this.data.length) this.offset++;
                return result >>> 0;
            }
        }
        return result >>> 0;
    }

    readSignedVarint() {
        let result = 0;
        let shift = 0;
        while (this.offset < this.data.length) {
            const byte = this.data[this.offset++];
            result |= (byte & 0x7F) << shift;
            if ((byte & 0x80) === 0) {
                return result | 0; // signed
            }
            shift += 7;
            if (shift >= 35) {
                while (this.offset < this.data.length && (this.data[this.offset] & 0x80) !== 0) {
                    this.offset++;
                }
                if (this.offset < this.data.length) this.offset++;
                return result | 0;
            }
        }
        return result | 0;
    }

    readTag() {
        const varint = this.readVarint();
        return {
            fieldNumber: varint >>> 3,
            wireType: varint & 0x07
        };
    }

    readBytes() {
        const length = this.readVarint();
        const bytes = this.data.slice(this.offset, this.offset + length);
        this.offset += length;
        return bytes;
    }

    readString() {
        return new TextDecoder().decode(this.readBytes());
    }

    readFloat() {
        const buf = this.data.buffer.slice(
            this.data.byteOffset + this.offset,
            this.data.byteOffset + this.offset + 4
        );
        this.offset += 4;
        return new DataView(buf).getFloat32(0, true);
    }

    readFixed64() {
        this.offset += 8;
    }

    skipField(wireType) {
        switch (wireType) {
            case 0: this.readVarint(); break;
            case 1: this.offset += 8; break;
            case 2: {
                const len = this.readVarint();
                this.offset += len;
                break;
            }
            case 5: this.offset += 4; break;
            default: throw new Error(`Unknown wire type: ${wireType}`);
        }
    }
}

// ==================== 消息编码 ====================

/** 编码 MouseEvent (event.proto) */
export function encodeMouseEvent(event) {
    return concatBytes(
        event.x !== undefined ? encodeVarintField(1, event.x) : null,
        event.y !== undefined ? encodeVarintField(2, event.y) : null,
        event.button !== undefined ? encodeVarintField(5, event.button) : null,
        event.buttonDown !== undefined ? encodeVarintField(6, event.buttonDown) : null,
        event.wheelDeltaX !== undefined ? encodeFloatField(7, event.wheelDeltaX) : null,
        event.wheelDeltaY !== undefined ? encodeFloatField(8, event.wheelDeltaY) : null,
        event.wheelTicksX !== undefined ? encodeFloatField(9, event.wheelTicksX) : null,
        event.wheelTicksY !== undefined ? encodeFloatField(10, event.wheelTicksY) : null,
    );
}

/** 编码 KeyEvent (event.proto) */
export function encodeKeyEvent(event) {
    return concatBytes(
        event.pressed !== undefined ? encodeVarintField(2, event.pressed) : null,
        event.usbKeycode !== undefined ? encodeVarintField(3, event.usbKeycode) : null,
        event.lockStates !== undefined ? encodeVarintField(4, event.lockStates) : null,
        event.capsLockState !== undefined ? encodeVarintField(5, event.capsLockState) : null,
        event.numLockState !== undefined ? encodeVarintField(6, event.numLockState) : null,
    );
}

/** 编码 ClipboardEvent (event.proto) */
export function encodeClipboardEvent(event) {
    return concatBytes(
        event.mimeType ? encodeLengthDelimitedField(1, event.mimeType) : null,
        event.data ? encodeLengthDelimitedField(2, event.data) : null,
    );
}

/** 编码 EventMessage 包装器 (internal.proto) */
export function encodeEventMessage(msg) {
    const parts = [];
    if (msg.timestamp !== undefined) {
        parts.push(encodeVarint64Field(1, msg.timestamp));
    }
    if (msg.keyEvent) {
        parts.push(encodeLengthDelimitedField(3, encodeKeyEvent(msg.keyEvent)));
    }
    if (msg.mouseEvent) {
        parts.push(encodeLengthDelimitedField(4, encodeMouseEvent(msg.mouseEvent)));
    }
    if (msg.textEvent) {
        const textBytes = msg.textEvent.text
            ? encodeLengthDelimitedField(1, msg.textEvent.text)
            : new Uint8Array(0);
        parts.push(encodeLengthDelimitedField(5, textBytes));
    }
    return concatBytes(...parts);
}

/** 编码 ClientResolution (control.proto) */
export function encodeClientResolution(res) {
    return concatBytes(
        res.widthPixels !== undefined ? encodeVarintField(1, res.widthPixels) : null,
        res.heightPixels !== undefined ? encodeVarintField(2, res.heightPixels) : null,
        res.xDpi !== undefined ? encodeVarintField(5, res.xDpi) : null,
        res.yDpi !== undefined ? encodeVarintField(6, res.yDpi) : null,
        res.screenId !== undefined ? encodeVarint64Field(7, res.screenId) : null,
    );
}

/** 编码 VideoControl (control.proto) */
export function encodeVideoControl(ctrl) {
    const parts = [];
    if (ctrl.enable !== undefined) {
        parts.push(encodeVarintField(1, ctrl.enable));
    }
    if (ctrl.targetFramerate !== undefined) {
        parts.push(encodeVarintField(5, ctrl.targetFramerate));
    }
    if (ctrl.framerateBoost) {
        const boostBytes = concatBytes(
            ctrl.framerateBoost.enabled !== undefined
                ? encodeVarintField(1, ctrl.framerateBoost.enabled) : null,
            ctrl.framerateBoost.captureIntervalMs !== undefined
                ? encodeVarintField(2, ctrl.framerateBoost.captureIntervalMs) : null,
            ctrl.framerateBoost.boostDurationMs !== undefined
                ? encodeVarintField(3, ctrl.framerateBoost.boostDurationMs) : null,
        );
        parts.push(encodeLengthDelimitedField(4, boostBytes));
    }
    return concatBytes(...parts);
}

/** 编码 AudioControl (control.proto) */
export function encodeAudioControl(ctrl) {
    return encodeVarintField(1, ctrl.enable);
}

/** 编码 Capabilities (control.proto) */
export function encodeCapabilities(caps) {
    return encodeLengthDelimitedField(1, caps.capabilities || '');
}

/** 编码 ExtensionMessage (control.proto) */
export function encodeExtensionMessage(msg) {
    return concatBytes(
        msg.type ? encodeLengthDelimitedField(1, msg.type) : null,
        msg.data ? encodeLengthDelimitedField(2, msg.data) : null,
    );
}

/** 编码 PeerConnectionParameters (control.proto) */
export function encodePeerConnectionParameters(params) {
    return concatBytes(
        params.preferredMinBitrateBps !== undefined
            ? encodeVarintField(1, params.preferredMinBitrateBps) : null,
        params.preferredMaxBitrateBps !== undefined
            ? encodeVarintField(2, params.preferredMaxBitrateBps) : null,
        params.requestIceRestart !== undefined
            ? encodeVarintField(3, params.requestIceRestart) : null,
        params.requestSdpRestart !== undefined
            ? encodeVarintField(4, params.requestSdpRestart) : null,
    );
}

/** 编码 ControlMessage 包装器 (internal.proto) */
export function encodeControlMessage(msg) {
    const parts = [];
    if (msg.clipboardEvent) {
        parts.push(encodeLengthDelimitedField(1, encodeClipboardEvent(msg.clipboardEvent)));
    }
    if (msg.clientResolution) {
        parts.push(encodeLengthDelimitedField(2, encodeClientResolution(msg.clientResolution)));
    }
    if (msg.videoControl) {
        parts.push(encodeLengthDelimitedField(3, encodeVideoControl(msg.videoControl)));
    }
    if (msg.cursorShape) {
        // CursorShapeInfo is field 4, encode inline
        parts.push(encodeLengthDelimitedField(4, encodeCursorShapeInfo(msg.cursorShape)));
    }
    if (msg.audioControl) {
        parts.push(encodeLengthDelimitedField(5, encodeAudioControl(msg.audioControl)));
    }
    if (msg.capabilities) {
        parts.push(encodeLengthDelimitedField(6, encodeCapabilities(msg.capabilities)));
    }
    if (msg.extensionMessage) {
        parts.push(encodeLengthDelimitedField(9, encodeExtensionMessage(msg.extensionMessage)));
    }
    if (msg.peerConnectionParameters) {
        parts.push(encodeLengthDelimitedField(14,
            encodePeerConnectionParameters(msg.peerConnectionParameters)));
    }
    return concatBytes(...parts);
}

/** 编码 CursorShapeInfo (control.proto) */
export function encodeCursorShapeInfo(info) {
    return concatBytes(
        info.width !== undefined ? encodeVarintField(1, info.width) : null,
        info.height !== undefined ? encodeVarintField(2, info.height) : null,
        info.hotspotX !== undefined ? encodeVarintField(3, info.hotspotX) : null,
        info.hotspotY !== undefined ? encodeVarintField(4, info.hotspotY) : null,
        info.data ? encodeLengthDelimitedField(5, info.data) : null,
    );
}

// ==================== 消息解码 ====================

/** 解码 CursorShapeInfo */
export function decodeCursorShapeInfo(data) {
    const reader = new ProtobufReader(data);
    const result = {};
    while (reader.hasMore()) {
        const { fieldNumber, wireType } = reader.readTag();
        switch (fieldNumber) {
            case 1: result.width = reader.readSignedVarint(); break;
            case 2: result.height = reader.readSignedVarint(); break;
            case 3: result.hotspotX = reader.readSignedVarint(); break;
            case 4: result.hotspotY = reader.readSignedVarint(); break;
            case 5: result.data = reader.readBytes(); break;
            default: reader.skipField(wireType);
        }
    }
    return result;
}

/** 解码 ClipboardEvent */
export function decodeClipboardEvent(data) {
    const reader = new ProtobufReader(data);
    const result = {};
    while (reader.hasMore()) {
        const { fieldNumber, wireType } = reader.readTag();
        switch (fieldNumber) {
            case 1: result.mimeType = reader.readString(); break;
            case 2: result.data = reader.readBytes(); break;
            default: reader.skipField(wireType);
        }
    }
    return result;
}

/** 解码 Capabilities */
export function decodeCapabilities(data) {
    const reader = new ProtobufReader(data);
    const result = {};
    while (reader.hasMore()) {
        const { fieldNumber, wireType } = reader.readTag();
        switch (fieldNumber) {
            case 1: result.capabilities = reader.readString(); break;
            default: reader.skipField(wireType);
        }
    }
    return result;
}

/** 解码 VideoControl */
export function decodeVideoControl(data) {
    const reader = new ProtobufReader(data);
    const result = {};
    while (reader.hasMore()) {
        const { fieldNumber, wireType } = reader.readTag();
        switch (fieldNumber) {
            case 1: result.enable = reader.readVarint() !== 0; break;
            case 5: result.targetFramerate = reader.readVarint(); break;
            case 4: {
                const boostData = reader.readBytes();
                const boostReader = new ProtobufReader(boostData);
                result.framerateBoost = {};
                while (boostReader.hasMore()) {
                    const bt = boostReader.readTag();
                    switch (bt.fieldNumber) {
                        case 1: result.framerateBoost.enabled = boostReader.readVarint() !== 0; break;
                        case 2: result.framerateBoost.captureIntervalMs = boostReader.readSignedVarint(); break;
                        case 3: result.framerateBoost.boostDurationMs = boostReader.readSignedVarint(); break;
                        default: boostReader.skipField(bt.wireType);
                    }
                }
                break;
            }
            default: reader.skipField(wireType);
        }
    }
    return result;
}

/** 解码 VideoTrackLayout */
export function decodeVideoTrackLayout(data) {
    const reader = new ProtobufReader(data);
    const result = {};
    while (reader.hasMore()) {
        const { fieldNumber, wireType } = reader.readTag();
        switch (fieldNumber) {
            case 1: result.mediaStreamId = reader.readString(); break;
            case 2: result.positionX = reader.readSignedVarint(); break;
            case 3: result.positionY = reader.readSignedVarint(); break;
            case 4: result.width = reader.readSignedVarint(); break;
            case 5: result.height = reader.readSignedVarint(); break;
            case 6: result.xDpi = reader.readSignedVarint(); break;
            case 7: result.yDpi = reader.readSignedVarint(); break;
            case 8: result.screenId = reader.readVarint(); break;
            case 9: result.displayName = reader.readString(); break;
            default: reader.skipField(wireType);
        }
    }
    return result;
}

/** 解码 VideoLayout */
export function decodeVideoLayout(data) {
    const reader = new ProtobufReader(data);
    const result = { videoTracks: [] };
    while (reader.hasMore()) {
        const { fieldNumber, wireType } = reader.readTag();
        switch (fieldNumber) {
            case 1: result.videoTracks.push(decodeVideoTrackLayout(reader.readBytes())); break;
            case 2: result.supportsFullDesktopCapture = reader.readVarint() !== 0; break;
            case 3: result.primaryScreenId = reader.readVarint(); break;
            default: reader.skipField(wireType);
        }
    }
    return result;
}

/** 解码 ExtensionMessage */
export function decodeExtensionMessage(data) {
    const reader = new ProtobufReader(data);
    const result = {};
    while (reader.hasMore()) {
        const { fieldNumber, wireType } = reader.readTag();
        switch (fieldNumber) {
            case 1: result.type = reader.readString(); break;
            case 2: result.data = reader.readString(); break;
            default: reader.skipField(wireType);
        }
    }
    return result;
}

/** 解码 TransportInfo */
export function decodeTransportInfo(data) {
    const reader = new ProtobufReader(data);
    const result = {};
    while (reader.hasMore()) {
        const { fieldNumber, wireType } = reader.readTag();
        switch (fieldNumber) {
            case 1: result.protocol = reader.readString(); break;
            default: reader.skipField(wireType);
        }
    }
    return result;
}

/** 解码 ControlMessage 包装器 */
export function decodeControlMessage(data) {
    const reader = new ProtobufReader(data);
    const result = {};
    while (reader.hasMore()) {
        const { fieldNumber, wireType } = reader.readTag();
        switch (fieldNumber) {
            case 1: result.clipboardEvent = decodeClipboardEvent(reader.readBytes()); break;
            case 2: {
                // ClientResolution - decode inline
                const resData = reader.readBytes();
                const resReader = new ProtobufReader(resData);
                const res = {};
                while (resReader.hasMore()) {
                    const rt = resReader.readTag();
                    switch (rt.fieldNumber) {
                        case 1: res.widthPixels = resReader.readSignedVarint(); break;
                        case 2: res.heightPixels = resReader.readSignedVarint(); break;
                        case 5: res.xDpi = resReader.readSignedVarint(); break;
                        case 6: res.yDpi = resReader.readSignedVarint(); break;
                        case 7: res.screenId = resReader.readVarint(); break;
                        default: resReader.skipField(rt.wireType);
                    }
                }
                result.clientResolution = res;
                break;
            }
            case 3: result.videoControl = decodeVideoControl(reader.readBytes()); break;
            case 4: result.cursorShape = decodeCursorShapeInfo(reader.readBytes()); break;
            case 5: {
                const audioData = reader.readBytes();
                const audioReader = new ProtobufReader(audioData);
                result.audioControl = {};
                while (audioReader.hasMore()) {
                    const at = audioReader.readTag();
                    switch (at.fieldNumber) {
                        case 1: result.audioControl.enable = audioReader.readVarint() !== 0; break;
                        default: audioReader.skipField(at.wireType);
                    }
                }
                break;
            }
            case 6: result.capabilities = decodeCapabilities(reader.readBytes()); break;
            case 9: result.extensionMessage = decodeExtensionMessage(reader.readBytes()); break;
            case 10: result.videoLayout = decodeVideoLayout(reader.readBytes()); break;
            case 13: result.transportInfo = decodeTransportInfo(reader.readBytes()); break;
            default: reader.skipField(wireType);
        }
    }
    return result;
}

/** 解码 EventMessage 包装器 */
export function decodeEventMessage(data) {
    const reader = new ProtobufReader(data);
    const result = {};
    while (reader.hasMore()) {
        const { fieldNumber, wireType } = reader.readTag();
        switch (fieldNumber) {
            case 1: result.timestamp = reader.readVarint(); break;
            case 3: {
                // KeyEvent
                const keyData = reader.readBytes();
                const keyReader = new ProtobufReader(keyData);
                const key = {};
                while (keyReader.hasMore()) {
                    const kt = keyReader.readTag();
                    switch (kt.fieldNumber) {
                        case 2: key.pressed = keyReader.readVarint() !== 0; break;
                        case 3: key.usbKeycode = keyReader.readVarint(); break;
                        default: keyReader.skipField(kt.wireType);
                    }
                }
                result.keyEvent = key;
                break;
            }
            case 4: {
                // MouseEvent
                const mouseData = reader.readBytes();
                const mouseReader = new ProtobufReader(mouseData);
                const mouse = {};
                while (mouseReader.hasMore()) {
                    const mt = mouseReader.readTag();
                    switch (mt.fieldNumber) {
                        case 1: mouse.x = mouseReader.readSignedVarint(); break;
                        case 2: mouse.y = mouseReader.readSignedVarint(); break;
                        case 5: mouse.button = mouseReader.readVarint(); break;
                        case 6: mouse.buttonDown = mouseReader.readVarint() !== 0; break;
                        case 7: mouse.wheelDeltaX = mouseReader.readFloat(); break;
                        case 8: mouse.wheelDeltaY = mouseReader.readFloat(); break;
                        default: mouseReader.skipField(mt.wireType);
                    }
                }
                result.mouseEvent = mouse;
                break;
            }
            default: reader.skipField(wireType);
        }
    }
    return result;
}

// ==================== MouseButton 枚举 ====================
export const MouseButton = {
    BUTTON_UNDEFINED: 0,
    BUTTON_LEFT: 1,
    BUTTON_MIDDLE: 2,
    BUTTON_RIGHT: 3,
    BUTTON_BACK: 4,
    BUTTON_FORWARD: 5,
};
