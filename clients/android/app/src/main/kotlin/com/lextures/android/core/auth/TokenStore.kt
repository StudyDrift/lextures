package com.lextures.android.core.auth

import android.content.Context
import androidx.security.crypto.EncryptedSharedPreferences
import androidx.security.crypto.MasterKey

/** Encrypted credential storage (parity with iOS Keychain). */
class TokenStore(context: Context) {
    private val prefs = EncryptedSharedPreferences.create(
        context,
        PREFS_NAME,
        MasterKey.Builder(context)
            .setKeyScheme(MasterKey.KeyScheme.AES256_GCM)
            .build(),
        EncryptedSharedPreferences.PrefKeyEncryptionScheme.AES256_SIV,
        EncryptedSharedPreferences.PrefValueEncryptionScheme.AES256_GCM,
    )

    fun readAccessToken(): String? = prefs.getString(Keys.ACCESS_TOKEN, null)

    fun readRefreshToken(): String? = prefs.getString(Keys.REFRESH_TOKEN, null)

    fun saveTokens(accessToken: String, refreshToken: String?) {
        prefs.edit()
            .putString(Keys.ACCESS_TOKEN, accessToken)
            .apply {
                if (refreshToken != null) {
                    putString(Keys.REFRESH_TOKEN, refreshToken)
                } else {
                    remove(Keys.REFRESH_TOKEN)
                }
            }
            .apply()
    }

    fun clearAll() {
        prefs.edit().clear().apply()
    }

    private object Keys {
        const val ACCESS_TOKEN = "access_token"
        const val REFRESH_TOKEN = "refresh_token"
    }

    companion object {
        private const val PREFS_NAME = "com.lextures.android.auth"
    }
}
