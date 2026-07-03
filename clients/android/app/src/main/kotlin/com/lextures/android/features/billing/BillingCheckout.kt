package com.lextures.android.features.billing

import android.content.Context
import android.net.Uri
import androidx.browser.customtabs.CustomTabsIntent
import com.lextures.android.core.lms.BillingLogic

object BillingCheckout {
    fun openCheckoutUrl(context: Context, checkoutUrl: String) {
        CustomTabsIntent.Builder()
            .setShowTitle(true)
            .build()
            .launchUrl(context, Uri.parse(checkoutUrl))
    }

    fun openPortalUrl(context: Context, portalUrl: String) {
        openCheckoutUrl(context, portalUrl)
    }
}