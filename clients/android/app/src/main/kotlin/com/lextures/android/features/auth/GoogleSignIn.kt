package com.lextures.android.features.auth

import android.content.Context
import androidx.credentials.CredentialManager
import androidx.credentials.CustomCredential
import androidx.credentials.GetCredentialRequest
import androidx.credentials.exceptions.GetCredentialCancellationException
import androidx.credentials.exceptions.GetCredentialException
import com.google.android.libraries.identity.googleid.GetSignInWithGoogleOption
import com.google.android.libraries.identity.googleid.GoogleIdTokenCredential
import com.google.android.libraries.identity.googleid.GoogleIdTokenParsingException
import com.lextures.android.BuildConfig
import java.security.MessageDigest
import java.security.SecureRandom

/** Result of a native Credential Manager Google sign-in (MOB.9). */
data class GoogleSignInResult(
    val idToken: String,
    val rawNonce: String?,
)

sealed class GoogleSignInError : Exception() {
    data object Cancelled : GoogleSignInError()
    data object NotConfigured : GoogleSignInError()
    data object Failed : GoogleSignInError()
}

/**
 * Native "Continue with Google" via Credential Manager + GetSignInWithGoogleOption.
 * Requires BuildConfig.GOOGLE_SERVER_CLIENT_ID (web/server OAuth client ID).
 */
object GoogleSignIn {
    fun isConfigured(): Boolean = BuildConfig.GOOGLE_SERVER_CLIENT_ID.isNotBlank()

    fun randomNonce(length: Int = 32): String {
        val charset = "0123456789ABCDEFGHIJKLMNOPQRSTUVXYZabcdefghijklmnopqrstuvwxyz-._"
        val random = SecureRandom()
        return buildString(length) {
            repeat(length) { append(charset[random.nextInt(charset.length)]) }
        }
    }

    fun sha256Hex(input: String): String {
        val digest = MessageDigest.getInstance("SHA-256").digest(input.toByteArray(Charsets.UTF_8))
        return digest.joinToString("") { "%02x".format(it) }
    }

    /**
     * Launches the system account picker and returns a Google ID token for server verification.
     * Throws [GoogleSignInError.Cancelled] on user cancel (callers should return silently).
     */
    suspend fun signIn(context: Context): GoogleSignInResult {
        val serverClientId = BuildConfig.GOOGLE_SERVER_CLIENT_ID.trim()
        if (serverClientId.isEmpty()) {
            throw GoogleSignInError.NotConfigured
        }
        val rawNonce = randomNonce()
        val hashedNonce = sha256Hex(rawNonce)
        val option = GetSignInWithGoogleOption.Builder(serverClientId)
            .setNonce(hashedNonce)
            .build()
        val request = GetCredentialRequest.Builder()
            .addCredentialOption(option)
            .build()
        val manager = CredentialManager.create(context)
        return try {
            val response = manager.getCredential(context, request)
            val credential = response.credential
            if (credential is CustomCredential &&
                credential.type == GoogleIdTokenCredential.TYPE_GOOGLE_ID_TOKEN_CREDENTIAL
            ) {
                val googleId = GoogleIdTokenCredential.createFrom(credential.data)
                val token = googleId.idToken
                if (token.isBlank()) throw GoogleSignInError.Failed
                GoogleSignInResult(idToken = token, rawNonce = rawNonce)
            } else {
                throw GoogleSignInError.Failed
            }
        } catch (_: GetCredentialCancellationException) {
            throw GoogleSignInError.Cancelled
        } catch (_: GoogleIdTokenParsingException) {
            throw GoogleSignInError.Failed
        } catch (e: GetCredentialException) {
            if (e is GetCredentialCancellationException) throw GoogleSignInError.Cancelled
            throw GoogleSignInError.Failed
        }
    }
}
