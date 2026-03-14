/**
 * spake2.js - SPAKE2 Curve25519 认证协议
 * 
 * 严格参照 BoringSSL spake25519.cc 和 Chromium spake2_authenticator.cc
 * 使用 @noble/curves Ed25519 实现
 * 
 * 协议流程:
 * 1. 双方各生成 SPAKE2 消息 (generate_msg)
 * 2. 交换消息后各自计算共享密钥 (process_msg)
 * 3. 使用 verification-hash 验证双方得到相同密钥
 */

import { ed25519 } from '@noble/curves/ed25519';
import { sha256, sha512, hmacSha256, concatBytes, uint64ToLittleEndian, base64Encode, base64Decode } from './auth-util.js';

// ==================== 常量 ====================

// BoringSSL 的 SPAKE2 M/N 点 (与 RFC 9382 不同!)
// 来自 BoringSSL spake25519.cc 的注释和预计算表
// M: 由 SHA-256 迭代哈希 'edwards25519 point generation seed (M)' 生成
//   encoded: 5ada7e4bf6ddd9adb6626d32131c6b5c51a1e347a3478f53cfcf441b88eed12e
// N: 由 SHA-256 迭代哈希 'edwards25519 point generation seed (N)' 生成
//   encoded: 10e3df0ae37d8e7a99b5fe74b44672103dbddcbd06af680d71329a11693bc778
const M_HEX = '5ada7e4bf6ddd9adb6626d32131c6b5c51a1e347a3478f53cfcf441b88eed12e';
const N_HEX = '10e3df0ae37d8e7a99b5fe74b44672103dbddcbd06af680d71329a11693bc778';

// Ed25519 群的阶 (prime order l)
const CURVE_ORDER = ed25519.CURVE.n;

// 角色定义 (与 BoringSSL spake2_role_alice/bob 对应)
export const SPAKE2_ROLE_ALICE = 0; // Client
export const SPAKE2_ROLE_BOB = 1;   // Host

// ==================== 工具函数 ====================

/** 字节数组(LE)转 BigInt */
function bytesToNumberLE(bytes) {
    let result = 0n;
    for (let i = bytes.length - 1; i >= 0; i--) {
        result = (result << 8n) | BigInt(bytes[i]);
    }
    return result;
}

/** BigInt 转字节数组(LE, 固定长度) */
function numberToBytesLE(num, length) {
    const bytes = new Uint8Array(length);
    let n = num;
    for (let i = 0; i < length; i++) {
        bytes[i] = Number(n & 0xFFn);
        n >>= 8n;
    }
    return bytes;
}

/**
 * sc_reduce: 将 64 字节 (512-bit LE integer) 归约为 mod l
 * 参照 BoringSSL x25519_sc_reduce
 */
function scReduce(bytes64) {
    const value = bytesToNumberLE(bytes64);
    return value % CURVE_ORDER;
}

/**
 * 点乘以余因子 8: 计算 8 * (scalar * point) = (8*scalar) * point
 * 
 * noble-curves 的 multiply() 要求 scalar < CURVE_ORDER，
 * 但 BoringSSL 的私钥是 sc_reduce(random) << 3，可能超出 CURVE_ORDER。
 * 解决方案: 先用 scalar * point，然后 3 次点倍增 (等价于 ×8)。
 * 
 * 这与 BoringSSL 的 left_shift_3 + ge_scalarmult 数学上完全等价。
 */
function multiplyWithCofactor(point, reducedScalar) {
    let result = point.multiply(reducedScalar);
    result = result.add(result); // 2x
    result = result.add(result); // 4x
    result = result.add(result); // 8x
    return result;
}

/**
 * 将密码字节映射为标量
 * 参照 BoringSSL SPAKE2_generate_msg:
 *   SHA512(password) → sc_reduce → password_scalar
 *   (注: 密码标量 hack 不影响协议正确性，可以省略)
 * 
 * @param {Uint8Array} passwordBytes - 密码字节
 * @returns {Promise<{scalar: bigint, hash: Uint8Array}>}
 */
async function passwordToScalar(passwordBytes) {
    // Step 1: SHA-512 密码 (与 BoringSSL 一致)
    const passwordHash = await sha512(passwordBytes);
    
    // Step 2: sc_reduce (512-bit → mod l)
    const scalar = scReduce(passwordHash);
    
    // 注: BoringSSL 的 "password scalar hack" (添加 l 的倍数清除底 3 位)
    // 数学上证明: 因为私钥是 8 的倍数，hack 在共享密钥计算中被消除
    // 因此可以安全省略 hack
    
    return {
        scalar: scalar === 0n ? 1n : scalar,
        hash: passwordHash,  // 64 字节，需要保存用于转录
    };
}

/**
 * 生成随机私钥 (归约后的标量，不含余因子)
 * 参照 BoringSSL SPAKE2_generate_msg:
 *   random(64 bytes) → sc_reduce
 * 
 * 注意: 余因子 ×8 通过 multiplyWithCofactor() 在点运算时应用
 */
function generateReducedPrivateKey() {
    const randomData = new Uint8Array(64);
    crypto.getRandomValues(randomData);
    const reduced = scReduce(randomData);
    return reduced === 0n ? 1n : reduced;
}

/**
 * PrefixWithLength - 在字符串前加4字节大端长度
 * 参照 spake2_authenticator.cc 的 PrefixWithLength
 * 注意: 使用 U32ToBigEndian (4字节大端)
 */
function prefixWithLength(str) {
    const encoder = new TextEncoder();
    const strBytes = encoder.encode(str);
    const lenBuf = new ArrayBuffer(4);
    new DataView(lenBuf).setUint32(0, strBytes.length, false); // big-endian
    return concatBytes(new Uint8Array(lenBuf), strBytes);
}

// ==================== SPAKE2 上下文 ====================

export class Spake2Context {
    /**
     * @param {number} role - SPAKE2_ROLE_ALICE (client) 或 SPAKE2_ROLE_BOB (host)
     * @param {string} myName - 本端标识
     * @param {string} theirName - 对端标识
     */
    constructor(role, myName, theirName) {
        this.role = role;
        this.myName = myName;
        this.theirName = theirName;
        this.privateKey = null;
        this.myMessage = null;
        this.authKey = null;
        this.passwordHash = null;  // SHA-512(password), 64字节, 用于转录
        
        // 解码 M 和 N 点
        try {
            this.pointM = ed25519.ExtendedPoint.fromHex(M_HEX);
            this.pointN = ed25519.ExtendedPoint.fromHex(N_HEX);
        } catch (e) {
            console.error('Failed to decode SPAKE2 M/N points:', e);
            throw new Error('SPAKE2 initialization failed: invalid M/N points');
        }
    }

    /**
     * 生成 SPAKE2 消息
     * 
     * 参照 BoringSSL SPAKE2_generate_msg:
     *   private_key = sc_reduce(random) * 8
     *   password_scalar = sc_reduce(SHA512(password))
     *   my_msg = private_key * G + password_scalar * (M or N)
     * 
     * @param {Uint8Array} password - 密码字节 (shared_secret_hash, 32 bytes)
     * @returns {Promise<Uint8Array>} SPAKE2 消息 (32 字节压缩点)
     */
    async generateMessage(password) {
        // 1. 计算密码标量和密码哈希
        const { scalar: passwordScalar, hash: passwordHash } = await passwordToScalar(password);
        this.passwordScalar = passwordScalar;
        this.passwordHash = passwordHash;

        // 2. 生成随机私钥 (sc_reduce后的标量，不含×8)
        this.reducedPrivateKey = generateReducedPrivateKey();

        // 3. 选择盲化点: Alice用M, Bob用N
        const blindingPoint = this.role === SPAKE2_ROLE_ALICE ? this.pointM : this.pointN;

        // 4. 计算: my_msg = (8 * reducedPrivateKey) * G + passwordScalar * blindingPoint
        //    等价于: 8 * (reducedPrivateKey * G) + passwordScalar * blindingPoint
        const G = ed25519.ExtendedPoint.BASE;
        const pubPoint = multiplyWithCofactor(G, this.reducedPrivateKey);
        const blindedPoint = blindingPoint.multiply(passwordScalar);
        const myMsgPoint = pubPoint.add(blindedPoint);

        // 5. 编码为压缩字节
        this.myMessage = myMsgPoint.toRawBytes();
        return this.myMessage;
    }

    /**
     * 处理对方的 SPAKE2 消息并派生密钥
     * 
     * 参照 BoringSSL SPAKE2_process_msg:
     *   Q = their_msg - password_scalar * (N or M)
     *   K = private_key * Q
     *   key = SHA512(transcript)
     * 
     * @param {Uint8Array} theirMessage - 对方的 SPAKE2 消息 (32 字节)
     * @returns {Promise<Uint8Array>} 派生的认证密钥 (64 字节)
     */
    async processMessage(theirMessage) {
        if (!this.reducedPrivateKey || !this.myMessage) {
            throw new Error('Must call generateMessage() first');
        }

        // 1. 解码对方消息为点
        let theirPoint;
        try {
            theirPoint = ed25519.ExtendedPoint.fromHex(theirMessage);
        } catch (e) {
            throw new Error('Invalid SPAKE2 message: failed to decode point');
        }

        // 2. 选择解盲点: Alice用N, Bob用M
        const unblindingPoint = this.role === SPAKE2_ROLE_ALICE ? this.pointN : this.pointM;

        // 3. 计算: Q = their_point - password_scalar * unblindingPoint
        const blindedPoint = unblindingPoint.multiply(this.passwordScalar);
        const Q = theirPoint.add(blindedPoint.negate());

        // 4. K = (8 * reducedPrivateKey) * Q = 8 * (reducedPrivateKey * Q)
        const sharedPoint = multiplyWithCofactor(Q, this.reducedPrivateKey);
        const dhShared = sharedPoint.toRawBytes();

        // 5. 构建转录并派生密钥 (参照 BoringSSL SPAKE2_process_msg)
        // SHA512(
        //   update_with_length_prefix(alice_name) ||
        //   update_with_length_prefix(bob_name) ||
        //   update_with_length_prefix(T*) ||     // Alice's msg
        //   update_with_length_prefix(S*) ||     // Bob's msg
        //   update_with_length_prefix(K) ||      // DH shared
        //   update_with_length_prefix(pw_hash)   // SHA512(password)
        // )
        const encoder = new TextEncoder();
        
        let aliceName, bobName, tMsg, sMsg;
        if (this.role === SPAKE2_ROLE_ALICE) {
            aliceName = this.myName;
            bobName = this.theirName;
            tMsg = this.myMessage;      // Alice's message (T*)
            sMsg = theirMessage;        // Bob's message (S*)
        } else {
            aliceName = this.theirName;
            bobName = this.myName;
            tMsg = theirMessage;        // Alice's message (T*)
            sMsg = this.myMessage;      // Bob's message (S*)
        }

        const aliceNameBytes = encoder.encode(aliceName);
        const bobNameBytes = encoder.encode(bobName);

        // 每个字段都带 8字节LE 长度前缀 (参照 BoringSSL update_with_length_prefix)
        const transcript = concatBytes(
            uint64ToLittleEndian(aliceNameBytes.length),
            aliceNameBytes,
            uint64ToLittleEndian(bobNameBytes.length),
            bobNameBytes,
            uint64ToLittleEndian(tMsg.length),
            new Uint8Array(tMsg),
            uint64ToLittleEndian(sMsg.length),
            new Uint8Array(sMsg),
            uint64ToLittleEndian(dhShared.length),
            dhShared,
            uint64ToLittleEndian(this.passwordHash.length),
            this.passwordHash
        );

        // 密钥 = SHA-512(transcript) → 64 字节
        this.authKey = await sha512(transcript);
        return this.authKey;
    }

    /**
     * 计算验证哈希
     * 
     * 参照 spake2_authenticator.cc CalculateVerificationHash:
     *   HMAC_SHA256(auth_key, ("host"|"client") + PrefixWithLength(local_id) + PrefixWithLength(remote_id))
     * 
     * 注: PrefixWithLength 使用 4字节大端长度前缀
     * 注: auth_key 是 64 字节 (SHA-512 输出)
     * 
     * @param {boolean} fromHost - 是否从 host 角度计算
     * @param {string} localId - local_id 参数
     * @param {string} remoteId - remote_id 参数
     * @returns {Promise<Uint8Array>} 验证哈希 (32 字节)
     */
    async calculateVerificationHash(fromHost, localId, remoteId) {
        if (!this.authKey) {
            throw new Error('Must call processMessage() first');
        }

        const encoder = new TextEncoder();
        const roleStr = fromHost ? 'host' : 'client';
        const message = concatBytes(
            encoder.encode(roleStr),
            prefixWithLength(localId),
            prefixWithLength(remoteId)
        );

        return await hmacSha256(this.authKey, message);
    }

    /**
     * 获取本端的验证哈希 (outgoing)
     * @returns {Promise<Uint8Array>}
     */
    async getOutgoingVerificationHash() {
        const isHost = this.role === SPAKE2_ROLE_BOB;
        return this.calculateVerificationHash(isHost, this.myName, this.theirName);
    }

    /**
     * 获取期望的对端验证哈希 (expected)
     * @returns {Promise<Uint8Array>}
     */
    async getExpectedVerificationHash() {
        const isHost = this.role === SPAKE2_ROLE_BOB;
        return this.calculateVerificationHash(!isHost, this.theirName, this.myName);
    }

    /**
     * 验证对方的验证哈希
     * @param {Uint8Array} theirHash 
     * @returns {Promise<boolean>}
     */
    async verifyHash(theirHash) {
        const expected = await this.getExpectedVerificationHash();
        if (theirHash.length !== expected.length) return false;
        // 安全比较
        let diff = 0;
        for (let i = 0; i < theirHash.length; i++) {
            diff |= theirHash[i] ^ expected[i];
        }
        return diff === 0;
    }
}

// ==================== SPAKE2 认证器 ====================

/**
 * Spake2Authenticator - 管理 SPAKE2 认证状态
 * 
 * 参照 NegotiatingClientAuthenticator + Spake2Authenticator
 */
export const AuthState = {
    MESSAGE_READY: 'MESSAGE_READY',
    WAITING_MESSAGE: 'WAITING_MESSAGE',
    ACCEPTED: 'ACCEPTED',
    REJECTED: 'REJECTED',
};

export class Spake2Authenticator {
    /**
     * @param {string} localId - 本端 JID
     * @param {string} remoteId - 对端 JID
     * @param {Uint8Array} sharedSecretHash - 共享密钥哈希 (32 bytes)
     */
    constructor(localId, remoteId, sharedSecretHash) {
        this.localId = localId;
        this.remoteId = remoteId;
        this.sharedSecretHash = sharedSecretHash;
        
        this.spakeCtx = new Spake2Context(
            SPAKE2_ROLE_ALICE,
            localId,
            remoteId
        );

        this.localSpakeMessage = null;  // 异步生成
        this.spakeMessageSent = false;
        this.outgoingVerificationHash = null;
        this.expectedVerificationHash = null;
        this.remoteCert = null;
        this.state = AuthState.MESSAGE_READY;
        this.rejectionReason = null;

        // NegotiatingClient 状态
        this.methodSelected = false;
        this.firstMessageSent = false;
        this._initialized = false;
    }

    /**
     * 初始化 (异步生成 SPAKE2 消息)
     * 必须在 getFirstNegotiationMessage() 之前调用
     */
    async initialize() {
        this.localSpakeMessage = await this.spakeCtx.generateMessage(this.sharedSecretHash);
        this._initialized = true;
    }

    /**
     * 获取第一条认证消息 (NegotiatingClientAuthenticator 的首条消息)
     * 仅包含 supported-methods，不含 SPAKE2 数据
     * @returns {object} { supportedMethods: string }
     */
    getFirstNegotiationMessage() {
        if (!this._initialized) {
            throw new Error('Must call initialize() first');
        }
        this.firstMessageSent = true;
        this.state = AuthState.WAITING_MESSAGE;
        return {
            supportedMethods: 'spake2_curve25519',
        };
    }

    /**
     * 处理 Host 回复的认证消息
     * @param {object} message - 解析后的认证消息
     *   { method, certificate, spakeMessage, verificationHash }
     * @returns {Promise<void>}
     */
    async processMessage(message) {
        // 首次收到 Host 的回复，检查 method
        if (!this.methodSelected && message.method) {
            if (message.method !== 'spake2_curve25519') {
                this.state = AuthState.REJECTED;
                this.rejectionReason = `Unsupported method: ${message.method}`;
                return;
            }
            this.methodSelected = true;
        }

        // 解析证书
        if (message.certificate) {
            this.remoteCert = base64Decode(message.certificate);
        }

        // 解析 SPAKE2 消息
        if (message.spakeMessage) {
            const spakeBytes = base64Decode(message.spakeMessage);
            
            try {
                await this.spakeCtx.processMessage(spakeBytes);
                this.outgoingVerificationHash = await this.spakeCtx.getOutgoingVerificationHash();
                this.expectedVerificationHash = await this.spakeCtx.getExpectedVerificationHash();
            } catch (e) {
                console.error('SPAKE2 process_msg failed:', e);
                this.state = AuthState.REJECTED;
                this.rejectionReason = 'Failed to process SPAKE2 message';
                return;
            }
        }

        // 验证对方的验证哈希
        if (message.verificationHash) {
            const hashBytes = base64Decode(message.verificationHash);
            const valid = await this.spakeCtx.verifyHash(hashBytes);
            if (!valid) {
                this.state = AuthState.REJECTED;
                this.rejectionReason = 'Verification hash mismatch';
                return;
            }
            this.state = AuthState.ACCEPTED;
            return;
        }

        // 需要发送消息
        this.state = AuthState.MESSAGE_READY;
    }

    /**
     * 获取下一条认证消息
     * @returns {object} 认证消息数据
     *   { method, spakeMessage, verificationHash }
     */
    getNextMessage() {
        const message = {};

        if (this.methodSelected) {
            message.method = 'spake2_curve25519';
        }

        // 发送 SPAKE2 消息 (如果还没发)
        if (!this.spakeMessageSent) {
            message.spakeMessage = base64Encode(this.localSpakeMessage);
            this.spakeMessageSent = true;
        }

        // 发送验证哈希 (如果已计算)
        if (this.outgoingVerificationHash) {
            message.verificationHash = base64Encode(this.outgoingVerificationHash);
            this.outgoingVerificationHash = null;
        }

        // 更新状态
        if (this.state !== AuthState.ACCEPTED) {
            this.state = AuthState.WAITING_MESSAGE;
        }

        return message;
    }

    /**
     * 获取 SPAKE2 协商产生的 auth key (64 字节)
     * 参照 C++ Spake2Authenticator::GetAuthKey()
     * @returns {Uint8Array|null}
     */
    getAuthKey() {
        return this.spakeCtx.authKey;
    }

    /**
     * 获取当前状态
     * @returns {string}
     */
    getState() {
        if (this.state === AuthState.ACCEPTED && this.outgoingVerificationHash) {
            return AuthState.MESSAGE_READY;
        }
        return this.state;
    }
}
