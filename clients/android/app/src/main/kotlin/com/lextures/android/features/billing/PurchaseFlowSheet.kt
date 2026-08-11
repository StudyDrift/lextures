package com.lextures.android.features.billing

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.Text
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalePreferences
import com.lextures.android.core.lms.BillingLogic
import com.lextures.android.core.lms.CheckoutTaxQuote
import com.lextures.android.core.lms.CouponApiError
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.MarketplaceLogic
import com.lextures.android.core.lms.MarketplaceObservability
import com.lextures.android.core.lms.PendingCheckoutContext
import com.lextures.android.features.home.HomeShellState
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.core.design.textSecondary
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun PurchaseFlowSheet(
    session: AuthSession,
    shell: HomeShellState?,
    localePrefs: LocalePreferences,
    courseId: String,
    courseCode: String,
    title: String,
    priceCents: Int,
    currency: String,
    onDismiss: () -> Unit,
    marketplaceSlug: String? = null,
    couponCode: String? = null,
    listPriceCents: Int? = null,
    discountCents: Int? = null,
    chargedCents: Int? = null,
    onAlreadyOwned: (() -> Unit)? = null,
    onGrantedFree: ((courseCode: String) -> Unit)? = null,
    onCouponRejected: ((reason: String?) -> Unit)? = null,
) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
    val accessToken = session.accessToken.value

    var quote by remember { mutableStateOf<CheckoutTaxQuote?>(null) }
    var loadingQuote by remember { mutableStateOf(false) }
    var purchasing by remember { mutableStateOf(false) }
    var errorMessage by remember { mutableStateOf<String?>(null) }

    val effectiveList = listPriceCents ?: priceCents
    val effectiveDiscount = discountCents ?: 0
    val effectiveCharged = chargedCents ?: priceCents
    val hasCoupon = !couponCode.isNullOrBlank() && effectiveDiscount > 0

    LaunchedEffect(accessToken, shell?.platformFeatures?.ffTaxCollection, courseId, couponCode) {
        val token = accessToken ?: return@LaunchedEffect
        // Coupon path uses server chargedCents in the sheet so Custom Tab amount matches (MKTC.6 AC-2).
        if (!couponCode.isNullOrBlank()) {
            quote = null
            loadingQuote = false
            return@LaunchedEffect
        }
        if (shell?.platformFeatures?.ffTaxCollection != true) return@LaunchedEffect
        loadingQuote = true
        quote = runCatching { LmsApi.fetchCheckoutQuote(courseId, token) }.getOrNull()
        loadingQuote = false
    }

    fun couponReasonMessage(reason: String?): String {
        val key = MarketplaceLogic.couponReasonKey(reason)
        val resId = when (key) {
            "mobile.marketplace.coupon.reason.ok" ->
                com.lextures.android.R.string.mobile_marketplace_coupon_reason_ok
            "mobile.marketplace.coupon.reason.not_found" ->
                com.lextures.android.R.string.mobile_marketplace_coupon_reason_not_found
            "mobile.marketplace.coupon.reason.inactive" ->
                com.lextures.android.R.string.mobile_marketplace_coupon_reason_inactive
            "mobile.marketplace.coupon.reason.not_started" ->
                com.lextures.android.R.string.mobile_marketplace_coupon_reason_not_started
            "mobile.marketplace.coupon.reason.expired" ->
                com.lextures.android.R.string.mobile_marketplace_coupon_reason_expired
            "mobile.marketplace.coupon.reason.exhausted" ->
                com.lextures.android.R.string.mobile_marketplace_coupon_reason_exhausted
            "mobile.marketplace.coupon.reason.already_used" ->
                com.lextures.android.R.string.mobile_marketplace_coupon_reason_already_used
            "mobile.marketplace.coupon.reason.currency_mismatch" ->
                com.lextures.android.R.string.mobile_marketplace_coupon_reason_currency_mismatch
            "mobile.marketplace.coupon.reason.course_free" ->
                com.lextures.android.R.string.mobile_marketplace_coupon_reason_course_free
            "mobile.marketplace.coupon.reason.owned" ->
                com.lextures.android.R.string.mobile_marketplace_coupon_reason_owned
            else -> com.lextures.android.R.string.mobile_marketplace_coupon_reason_not_found
        }
        return L.text(context, localePrefs, resId)
    }

    ModalBottomSheet(onDismissRequest = onDismiss, sheetState = sheetState) {
        Column(
            modifier = Modifier.padding(horizontal = 16.dp, vertical = 8.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp),
        ) {
            Text(
                L.text(context, localePrefs, com.lextures.android.R.string.mobile_billing_purchaseTitle),
                fontWeight = FontWeight.Bold,
                fontSize = 20.sp,
            )
            errorMessage?.let { LmsErrorBanner(message = it) }

            LmsCard {
                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    Text(title, fontWeight = FontWeight.SemiBold)
                    Text(
                        L.text(context, localePrefs, com.lextures.android.R.string.mobile_billing_purchaseHint),
                        fontSize = 12.sp,
                        color = textSecondary(),
                    )
                }
            }

            LmsCard {
                if (loadingQuote) {
                    CircularProgressIndicator(modifier = Modifier.fillMaxWidth())
                } else if (quote != null) {
                    val q = quote!!
                    if (q.lines.isNotEmpty()) {
                        q.lines.forEach { line ->
                            Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                                Text(line.label)
                                Text(BillingLogic.formatMoney(line.amountCents, q.currency), fontWeight = FontWeight.SemiBold)
                            }
                        }
                    } else {
                        Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                            Text(L.text(context, localePrefs, com.lextures.android.R.string.mobile_billing_subtotal))
                            Text(BillingLogic.formatMoney(q.subtotalCents, q.currency), fontWeight = FontWeight.SemiBold)
                        }
                        if (hasCoupon) {
                            val discountMoney = BillingLogic.formatMoney(effectiveDiscount, q.currency)
                            Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                                Text(
                                    L.format(
                                        context,
                                        localePrefs,
                                        com.lextures.android.R.string.mobile_marketplace_coupon_sheetLine,
                                        discountMoney,
                                        couponCode.orEmpty(),
                                    ),
                                )
                                Text("−$discountMoney", fontWeight = FontWeight.SemiBold)
                            }
                        }
                        if (q.taxAmountCents > 0) {
                            Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                                Text(L.text(context, localePrefs, com.lextures.android.R.string.mobile_billing_tax))
                                Text(BillingLogic.formatMoney(q.taxAmountCents, q.currency), fontWeight = FontWeight.SemiBold)
                            }
                        }
                    }
                    Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                        Text(
                            L.text(context, localePrefs, com.lextures.android.R.string.mobile_billing_total),
                            fontWeight = FontWeight.Bold,
                        )
                        Text(
                            BillingLogic.formatMoney(q.totalCents, q.currency),
                            fontWeight = FontWeight.Bold,
                        )
                    }
                } else {
                    // Server-backed subtotal / coupon / total when tax quote is unavailable.
                    if (hasCoupon || listPriceCents != null) {
                        Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                            Text(L.text(context, localePrefs, com.lextures.android.R.string.mobile_billing_subtotal))
                            Text(
                                BillingLogic.formatMoney(effectiveList, currency),
                                fontWeight = FontWeight.SemiBold,
                            )
                        }
                        if (hasCoupon) {
                            val discountMoney = BillingLogic.formatMoney(effectiveDiscount, currency)
                            Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                                Text(
                                    L.format(
                                        context,
                                        localePrefs,
                                        com.lextures.android.R.string.mobile_marketplace_coupon_sheetLine,
                                        discountMoney,
                                        couponCode.orEmpty(),
                                    ),
                                )
                                Text(
                                    "−$discountMoney",
                                    fontWeight = FontWeight.SemiBold,
                                )
                            }
                        }
                        Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                            Text(
                                L.text(context, localePrefs, com.lextures.android.R.string.mobile_billing_total),
                                fontWeight = FontWeight.Bold,
                            )
                            Text(
                                BillingLogic.formatMoney(effectiveCharged, currency),
                                fontWeight = FontWeight.Bold,
                            )
                        }
                    } else {
                        Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                            Text(
                                L.text(context, localePrefs, com.lextures.android.R.string.mobile_billing_total),
                                fontWeight = FontWeight.Bold,
                            )
                            Text(
                                BillingLogic.formatMoney(priceCents, currency),
                                fontWeight = FontWeight.Bold,
                            )
                        }
                    }
                }
            }

            Button(
                onClick = {
                    val token = accessToken ?: return@Button
                    purchasing = true
                    errorMessage = null
                    scope.launch {
                        try {
                            shell?.pendingCheckout = PendingCheckoutContext(
                                courseId = courseId,
                                courseCode = courseCode,
                                title = title,
                            )
                            if (!marketplaceSlug.isNullOrBlank()) {
                                val result = LmsApi.checkoutMarketplaceCourse(
                                    marketplaceSlug,
                                    token,
                                    couponCode,
                                )
                                if (result.alreadyOwned) {
                                    shell?.pendingCheckout = null
                                    if (shell?.pendingCoupon?.slug == marketplaceSlug) {
                                        shell.pendingCoupon = null
                                    }
                                    onDismiss()
                                    onAlreadyOwned?.invoke()
                                    return@launch
                                }
                                if (result.grantedFree) {
                                    shell?.pendingCheckout = null
                                    if (shell?.pendingCoupon?.slug == marketplaceSlug) {
                                        shell.pendingCoupon = null
                                    }
                                    onDismiss()
                                    onGrantedFree?.invoke(result.courseCode ?: courseCode)
                                        ?: onAlreadyOwned?.invoke()
                                    return@launch
                                }
                                val url = result.checkoutUrl
                                if (url.isNullOrBlank()) {
                                    shell?.pendingCheckout = null
                                    errorMessage = L.text(
                                        context,
                                        localePrefs,
                                        com.lextures.android.R.string.mobile_billing_checkoutError,
                                    )
                                    MarketplaceObservability.record(
                                        "marketplace_purchase_failed",
                                        mapOf("reason" to "url"),
                                    )
                                    return@launch
                                }
                                onDismiss()
                                BillingCheckout.openCheckoutUrl(context, url)
                                return@launch
                            }

                            val result = LmsApi.startCheckout(
                                courseId = courseId,
                                successUrl = BillingLogic.checkoutSuccessUrl(courseId),
                                cancelUrl = BillingLogic.checkoutCancelUrl(),
                                usePaymentsAbstraction = shell?.platformFeatures?.ffPaymentsEnabled == true,
                                accessToken = token,
                            )
                            onDismiss()
                            BillingCheckout.openCheckoutUrl(context, result.checkoutUrl)
                        } catch (e: CouponApiError) {
                            shell?.pendingCheckout = null
                            when (e.status) {
                                422 -> {
                                    errorMessage = couponReasonMessage(e.reason)
                                    onCouponRejected?.invoke(e.reason)
                                }
                                429 -> {
                                    errorMessage = L.text(
                                        context,
                                        localePrefs,
                                        com.lextures.android.R.string.mobile_marketplace_coupon_rateLimited,
                                    )
                                }
                                else -> {
                                    errorMessage = L.text(
                                        context,
                                        localePrefs,
                                        com.lextures.android.R.string.mobile_billing_checkoutError,
                                    )
                                }
                            }
                            if (!marketplaceSlug.isNullOrBlank()) {
                                MarketplaceObservability.record(
                                    "marketplace_purchase_failed",
                                    mapOf(
                                        "reason" to (e.reason ?: e.status.toString()),
                                    ),
                                )
                            }
                        } catch (_: Exception) {
                            shell?.pendingCheckout = null
                            errorMessage = L.text(
                                context,
                                localePrefs,
                                com.lextures.android.R.string.mobile_billing_checkoutError,
                            )
                            if (!marketplaceSlug.isNullOrBlank()) {
                                MarketplaceObservability.record(
                                    "marketplace_purchase_failed",
                                    mapOf("reason" to "start"),
                                )
                            }
                        } finally {
                            purchasing = false
                        }
                    }
                },
                enabled = !purchasing && accessToken != null,
                modifier = Modifier.fillMaxWidth(),
            ) {
                Text(
                    if (purchasing) {
                        L.text(context, localePrefs, com.lextures.android.R.string.mobile_billing_startingCheckout)
                    } else {
                        L.text(context, localePrefs, com.lextures.android.R.string.mobile_billing_purchase)
                    },
                )
            }

            Text(
                L.text(context, localePrefs, com.lextures.android.R.string.mobile_billing_storePolicyNote),
                fontSize = 11.sp,
                color = textSecondary(),
            )
        }
    }
}
