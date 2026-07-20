package com.quickcoder.quickdesk_android

import android.security.keystore.KeyGenParameterSpec
import android.security.keystore.KeyProperties
import android.util.Base64
import java.math.BigInteger
import java.security.KeyPairGenerator
import java.security.KeyStore
import java.security.SecureRandom
import java.security.cert.X509Certificate
import java.util.Date
import javax.security.auth.x500.X500Principal

/**
 * 被控端 SPAKE2 Host 身份证书。
 *
 * Chromium Client 要求 Host 的第一条 SPAKE2 消息携带 DER X.509 证书（base64）。
 * 密钥对生成并保存在 Android Keystore，证书随密钥自动生成（自签名）。
 */
object HostIdentity {

    private const val ANDROID_KEY_STORE = "AndroidKeyStore"
    private const val HOST_KEY_ALIAS = "quickdesk_chromoting_host_identity"
    private const val HOST_CERT_VALIDITY_MS = 10L * 365L * 24L * 60L * 60L * 1000L

    /** 返回 base64(DER) 证书；密钥不存在或证书为空时重建。 */
    @Synchronized
    fun getOrCreateCertificate(): String {
        val keyStore = KeyStore.getInstance(ANDROID_KEY_STORE).apply { load(null) }
        var certificate = keyStore.getCertificate(HOST_KEY_ALIAS) as? X509Certificate

        if (certificate == null || certificate.encoded.isEmpty()) {
            if (keyStore.containsAlias(HOST_KEY_ALIAS)) {
                keyStore.deleteEntry(HOST_KEY_ALIAS)
            }

            val now = System.currentTimeMillis()
            val serialNumber = BigInteger(64, SecureRandom()).let {
                if (it == BigInteger.ZERO) BigInteger.ONE else it
            }
            val keyPairGenerator = KeyPairGenerator.getInstance(
                KeyProperties.KEY_ALGORITHM_RSA,
                ANDROID_KEY_STORE,
            )
            val spec = KeyGenParameterSpec.Builder(
                HOST_KEY_ALIAS,
                KeyProperties.PURPOSE_SIGN or KeyProperties.PURPOSE_VERIFY,
            )
                .setKeySize(2048)
                .setDigests(KeyProperties.DIGEST_SHA256, KeyProperties.DIGEST_SHA512)
                .setSignaturePaddings(KeyProperties.SIGNATURE_PADDING_RSA_PKCS1)
                .setCertificateSubject(X500Principal("CN=chromoting"))
                .setCertificateSerialNumber(serialNumber)
                .setCertificateNotBefore(Date(now - 60_000L))
                .setCertificateNotAfter(Date(now + HOST_CERT_VALIDITY_MS))
                .build()

            keyPairGenerator.initialize(spec)
            keyPairGenerator.generateKeyPair()
            certificate = keyStore.getCertificate(HOST_KEY_ALIAS) as? X509Certificate
        }

        val der = certificate?.encoded
            ?: throw IllegalStateException("Android Keystore did not return a host certificate")
        if (der.isEmpty()) {
            throw IllegalStateException("Android Keystore returned an empty host certificate")
        }
        return Base64.encodeToString(der, Base64.NO_WRAP)
    }
}
