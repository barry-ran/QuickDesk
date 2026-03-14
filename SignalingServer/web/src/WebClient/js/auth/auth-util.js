/**
 * auth-util.js - 认证工具函数
 * 
 * 参照 src/remoting/protocol/auth_util.cc
 * 优先使用 Web Crypto API (secure context)，
 * 非 secure context (手机 HTTP 访问) 回退到 @noble/hashes。
 */

import { sha256 as _nobleSha256 } from '@noble/hashes/sha256';
import { sha512 as _nobleSha512 } from '@noble/hashes/sha512';
import { hmac } from '@noble/hashes/hmac';

const hasSubtle = typeof crypto !== 'undefined' && !!crypto.subtle;

/**
 * @param {string} tag - Host ID (device_id)
 * @param {string} sharedSecret - device_id + access_code
 * @returns {Promise<Uint8Array>} 32 字节 HMAC-SHA256 哈希
 */
export async function getSharedSecretHash(tag, sharedSecret) {
    const encoder = new TextEncoder();
    return hmacSha256(encoder.encode(tag), encoder.encode(sharedSecret));
}

/** @returns {Promise<Uint8Array>} */
export async function hmacSha256(key, data) {
    if (hasSubtle) {
        const cryptoKey = await crypto.subtle.importKey(
            'raw', key, { name: 'HMAC', hash: 'SHA-256' }, false, ['sign']);
        return new Uint8Array(await crypto.subtle.sign('HMAC', cryptoKey, data));
    }
    return hmac(_nobleSha256, key, data);
}

/** @returns {Promise<Uint8Array>} */
export async function sha256(data) {
    if (hasSubtle) {
        return new Uint8Array(await crypto.subtle.digest('SHA-256', data));
    }
    return _nobleSha256(data);
}

/** @returns {Promise<Uint8Array>} */
export async function sha512(data) {
    if (hasSubtle) {
        return new Uint8Array(await crypto.subtle.digest('SHA-512', data));
    }
    return _nobleSha512(data);
}

/**
 * Base64 编码
 * @param {Uint8Array} data 
 * @returns {string}
 */
export function base64Encode(data) {
    let binary = '';
    for (let i = 0; i < data.length; i++) {
        binary += String.fromCharCode(data[i]);
    }
    return btoa(binary);
}

/**
 * Base64 解码
 * @param {string} str 
 * @returns {Uint8Array}
 */
export function base64Decode(str) {
    const binary = atob(str);
    const bytes = new Uint8Array(binary.length);
    for (let i = 0; i < binary.length; i++) {
        bytes[i] = binary.charCodeAt(i);
    }
    return bytes;
}

/**
 * 字节数组转十六进制字符串
 * @param {Uint8Array} bytes 
 * @returns {string}
 */
export function bytesToHex(bytes) {
    return Array.from(bytes).map(b => b.toString(16).padStart(2, '0')).join('');
}

/**
 * 十六进制字符串转字节数组
 * @param {string} hex 
 * @returns {Uint8Array}
 */
export function hexToBytes(hex) {
    const bytes = new Uint8Array(hex.length / 2);
    for (let i = 0; i < bytes.length; i++) {
        bytes[i] = parseInt(hex.substr(i * 2, 2), 16);
    }
    return bytes;
}

/**
 * 生成随机字节
 * @param {number} length 
 * @returns {Uint8Array}
 */
export function randomBytes(length) {
    const bytes = new Uint8Array(length);
    crypto.getRandomValues(bytes);
    return bytes;
}

/**
 * 生成 UUID v4
 * @returns {string}
 */
export function generateUUID() {
    if (typeof crypto.randomUUID === 'function') {
        return crypto.randomUUID();
    }
    const b = randomBytes(16);
    b[6] = (b[6] & 0x0f) | 0x40;
    b[8] = (b[8] & 0x3f) | 0x80;
    const h = bytesToHex(b);
    return `${h.slice(0,8)}-${h.slice(8,12)}-${h.slice(12,16)}-${h.slice(16,20)}-${h.slice(20)}`;
}

/**
 * 合并字节数组
 * @param  {...Uint8Array} arrays 
 * @returns {Uint8Array}
 */
export function concatBytes(...arrays) {
    const totalLength = arrays.reduce((sum, arr) => sum + arr.length, 0);
    const result = new Uint8Array(totalLength);
    let offset = 0;
    for (const arr of arrays) {
        result.set(arr, offset);
        offset += arr.length;
    }
    return result;
}

/**
 * 将 uint32 转为 4 字节大端字节数组
 * @param {number} value 
 * @returns {Uint8Array}
 */
export function uint32ToBigEndian(value) {
    const buf = new ArrayBuffer(4);
    new DataView(buf).setUint32(0, value, false);
    return new Uint8Array(buf);
}

/**
 * 将 uint64 转为 8 字节小端字节数组
 * @param {number} value 
 * @returns {Uint8Array}
 */
export function uint64ToLittleEndian(value) {
    const buf = new ArrayBuffer(8);
    const view = new DataView(buf);
    // JS number precision: ok for lengths up to 2^53
    view.setUint32(0, value & 0xFFFFFFFF, true);
    view.setUint32(4, Math.floor(value / 0x100000000), true);
    return new Uint8Array(buf);
}
