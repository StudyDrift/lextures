package com.lextures.android.core.lms

import kotlinx.serialization.Serializable

@Serializable
data class BillingEntitlement(
    val id: String,
    val entitlementType: String,
    val courseId: String? = null,
    val amountPaidCents: Int = 0,
    val subtotalCents: Int? = null,
    val taxAmountCents: Int? = null,
    val taxType: String? = null,
    val taxJurisdiction: String? = null,
    val reverseCharge: Boolean? = null,
    val invoiceId: String? = null,
    val currency: String = "USD",
    val validFrom: String,
    val validUntil: String? = null,
    val status: String,
)

@Serializable
data class BillingEntitlementsResponse(
    val entitlements: List<BillingEntitlement>? = null,
)

@Serializable
data class BillingTransaction(
    val id: String,
    val courseId: String? = null,
    val provider: String,
    val providerTxnId: String,
    val amountCents: Int,
    val currency: String,
    val status: String,
    val subscriptionId: String? = null,
    val createdAt: String,
)

@Serializable
data class BillingTransactionsResponse(
    val transactions: List<BillingTransaction>? = null,
)

@Serializable
data class CheckoutSessionRequest(
    val courseId: String? = null,
    val successUrl: String,
    val cancelUrl: String,
)

@Serializable
data class CheckoutSessionResponse(
    val sessionId: String? = null,
    val checkoutUrl: String,
    val provider: String? = null,
)

@Serializable
data class CheckoutTaxQuoteRequest(
    val courseId: String,
)

@Serializable
data class CheckoutTaxQuoteLine(
    val label: String,
    val amountCents: Int,
)

@Serializable
data class CheckoutTaxQuote(
    val subtotalCents: Int,
    val taxAmountCents: Int,
    val totalCents: Int,
    val currency: String,
    val taxType: String? = null,
    val taxJurisdiction: String? = null,
    val reverseCharge: Boolean? = null,
    val lines: List<CheckoutTaxQuoteLine> = emptyList(),
)

@Serializable
data class BillingPortalResponse(
    val portalUrl: String,
)

@Serializable
data class EntitlementCheckResponse(
    val entitled: Boolean? = null,
)

data class PendingCheckoutContext(
    val courseId: String,
    val courseCode: String,
    val title: String,
)

sealed class CheckoutReturnPhase {
    data class Success(val courseId: String?) : CheckoutReturnPhase()
    data object Cancel : CheckoutReturnPhase()
}