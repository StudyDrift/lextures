package com.lextures.android.features.auth

import android.content.Context
import android.net.Uri
import androidx.browser.customtabs.CustomTabsIntent
import com.lextures.android.core.auth.AuthConstants
import com.lextures.android.core.auth.SsoProvider
import com.lextures.android.core.config.AppConfiguration
import java.net.URLEncoder

object SsoAuth {
    fun start(context: Context, provider: SsoProvider) {
        val url = buildStartUrl(provider)
        CustomTabsIntent.Builder()
            .setShowTitle(true)
            .build()
            .launchUrl(context, Uri.parse(url))
    }

    fun buildStartUrl(provider: SsoProvider): String {
        val next = URLEncoder.encode(AuthConstants.MOBILE_CALLBACK_PATH, Charsets.UTF_8.name())
        val path = when (provider) {
            is SsoProvider.Saml -> {
                val idpId = URLEncoder.encode(provider.idpId, Charsets.UTF_8.name())
                "/auth/saml/login?idpId=$idpId&RelayState=$next"
            }
            is SsoProvider.Oidc -> {
                val separator = if (provider.path.contains("?")) "&" else "?"
                "${provider.path}${separator}next=$next"
            }
        }
        return AppConfiguration.apiUrl(path).toString()
    }
}
