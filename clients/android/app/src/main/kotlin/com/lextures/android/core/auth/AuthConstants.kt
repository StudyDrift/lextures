package com.lextures.android.core.auth

object AuthConstants {
    /** RelayState / OIDC `next` path that tells the web saml-callback page to return tokens to the app. */
    const val MOBILE_CALLBACK_PATH = "/__mobile_callback__"

    const val CALLBACK_SCHEME = "lextures"
    const val CALLBACK_HOST = "auth"
    const val CALLBACK_PATH = "/callback"
}
