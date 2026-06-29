package com.lextures.android.core.network

import com.lextures.android.core.auth.AuthSession

/** Weak link from the networking layer back to the active auth session for 401 refresh/retry. */
object NetworkAuthContext {
    @Volatile
    var session: AuthSession? = null
}
