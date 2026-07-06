package com.lextures.android.core.network

import com.lextures.android.BuildConfig
import com.lextures.android.core.auth.SignOutReason
import com.lextures.android.core.config.AppConfiguration
import com.lextures.android.core.i18n.MobileLocale
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.MultipartBody
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.asRequestBody
import okhttp3.RequestBody.Companion.toRequestBody
import java.io.File

class ApiClient(
    private val http: OkHttpClient = OkHttpClient(),
    private val json: Json = Json { ignoreUnknownKeys = true },
) {
    suspend fun request(
        path: String,
        method: String = "GET",
        body: String? = null,
        accessToken: String? = null,
        idempotencyKey: String? = null,
    ): Pair<String, Int> = requestRaw(path, method, body, accessToken, idempotencyKey)

    suspend fun requestRaw(
        path: String,
        method: String = "GET",
        body: String? = null,
        accessToken: String? = null,
        idempotencyKey: String? = null,
        isRetryAfterRefresh: Boolean = false,
    ): Pair<String, Int> {
        val builder = Request.Builder()
            .url(AppConfiguration.apiUrl(path))
            .header("Accept", "application/json")
            .header("X-Platform", "android")
            .header("X-App-Version", BuildConfig.VERSION_NAME)
            .header("Accept-Language", MobileLocale.acceptLanguage)

        if (body != null) {
            builder
                .method(method, body.toRequestBody("application/json".toMediaType()))
                .header("Content-Type", "application/json")
        } else {
            builder.method(method, null)
        }

        if (!accessToken.isNullOrBlank()) {
            builder.header("Authorization", "Bearer $accessToken")
        }

        if (!idempotencyKey.isNullOrBlank()) {
            builder.header("X-Idempotency-Key", idempotencyKey)
        }

        val response = try {
            http.newCall(builder.build()).execute()
        } catch (e: Exception) {
            throw ApiError.Transport(e)
        }

        response.use {
            val responseBody = it.body?.string().orEmpty()
            val code = it.code
            if (code !in 200..299) {
                if (code == 401 && !accessToken.isNullOrBlank()) {
                    if (isRetryAfterRefresh) {
                        NetworkAuthContext.session?.signOut(SignOutReason.SessionRevoked)
                    } else if (path != "/api/v1/auth/refresh") {
                        val session = NetworkAuthContext.session
                        if (session != null) {
                            session.refreshIfNeeded(force = true)
                            val newToken = session.accessToken.value
                            if (!newToken.isNullOrBlank()) {
                                return requestRaw(
                                    path = path,
                                    method = method,
                                    body = body,
                                    accessToken = newToken,
                                    idempotencyKey = idempotencyKey,
                                    isRetryAfterRefresh = true,
                                )
                            }
                        }
                    }
                }
                throw ApiError.HttpStatus(code, parseApiErrorMessage(responseBody))
            }
            return responseBody to code
        }
    }

    fun <T> encodeBody(value: T, serializer: kotlinx.serialization.KSerializer<T>): String =
        json.encodeToString(serializer, value)

    suspend fun uploadMultipart(
        path: String,
        fieldName: String,
        fileName: String,
        mimeType: String,
        fileBytes: ByteArray,
        accessToken: String,
        extraFields: Map<String, String> = emptyMap(),
    ): String {
        val temp = File.createTempFile("lextures-upload-", "-$fileName")
        try {
            temp.writeBytes(fileBytes)
            val multipartBuilder = MultipartBody.Builder()
                .setType(MultipartBody.FORM)
            for ((key, value) in extraFields) {
                multipartBuilder.addFormDataPart(key, value)
            }
            val body = multipartBuilder
                .addFormDataPart(
                    fieldName,
                    fileName,
                    temp.asRequestBody(mimeType.toMediaType()),
                )
                .build()
            val request = Request.Builder()
                .url(AppConfiguration.apiUrl(path))
                .post(body)
                .header("Accept", "application/json")
                .header("Authorization", "Bearer $accessToken")
                .header("X-Platform", "android")
                .header("X-App-Version", BuildConfig.VERSION_NAME)
                .header("Accept-Language", MobileLocale.acceptLanguage)
                .build()
            val response = try {
                http.newCall(request).execute()
            } catch (e: Exception) {
                throw ApiError.Transport(e)
            }
            response.use {
                val responseBody = it.body?.string().orEmpty()
                if (it.code !in 200..299) {
                    throw ApiError.HttpStatus(it.code, parseApiErrorMessage(responseBody))
                }
                return responseBody
            }
        } finally {
            temp.delete()
        }
    }
}
