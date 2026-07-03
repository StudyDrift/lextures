package com.lextures.android.core.lms

import com.lextures.android.core.config.AppConfiguration
import com.lextures.android.core.navigation.MobilePlatformFeatures
import java.net.URLEncoder

/** Checkout and billing helpers (M9.2). v1 uses compliant web-checkout handoff (Stripe in browser). */
object BillingLogic {
    const val ENTITLEMENT_POLL_ATTEMPTS = 10
    const val ENTITLEMENT_POLL_INTERVAL_MS = 1_000L

    fun billingEnabled(features: MobilePlatformFeatures): Boolean =
        features.ffStripeBilling || features.ffPaymentsEnabled

    fun checkoutSuccessPath(courseId: String): String {
        val encoded = URLEncoder.encode(courseId, Charsets.UTF_8.name())
        return "/checkout/success?course_id=$encoded"
    }

    fun checkoutCancelPath(): String = "/checkout/cancel"

    fun checkoutSuccessUrl(courseId: String): String =
        AppConfiguration.webUrl(checkoutSuccessPath(courseId))

    fun checkoutCancelUrl(): String = AppConfiguration.webUrl(checkoutCancelPath())

    fun billingReturnUrl(): String = AppConfiguration.webUrl("/me/billing")

    fun formatMoney(cents: Int, currency: String = "USD"): String =
        PathsLogic.formatPrice(cents, currency)

    fun entitlementLabelRes(type: String): Int? = when (type) {
        "course_purchase" -> com.lextures.android.R.string.mobile_billing_entitlement_coursePurchase
        "subscription_monthly" -> com.lextures.android.R.string.mobile_billing_entitlement_subscriptionMonthly
        "subscription_annual" -> com.lextures.android.R.string.mobile_billing_entitlement_subscriptionAnnual
        else -> null
    }

    fun isSubscriptionEntitlement(type: String): Boolean = type.startsWith("subscription")

    fun activeSubscription(entitlements: List<BillingEntitlement>): BillingEntitlement? =
        entitlements.firstOrNull { isSubscriptionEntitlement(it.entitlementType) && it.status == "active" }

    fun checkoutEndpoint(usePaymentsAbstraction: Boolean): String =
        if (usePaymentsAbstraction) "/api/v1/checkout" else "/api/v1/billing/checkout"
}