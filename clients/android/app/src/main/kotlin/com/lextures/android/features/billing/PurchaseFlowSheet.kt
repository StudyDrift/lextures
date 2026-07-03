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
import com.lextures.android.core.lms.LmsApi
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
) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
    val accessToken = session.accessToken.value

    var quote by remember { mutableStateOf<CheckoutTaxQuote?>(null) }
    var loadingQuote by remember { mutableStateOf(false) }
    var purchasing by remember { mutableStateOf(false) }
    var errorMessage by remember { mutableStateOf<String?>(null) }

    LaunchedEffect(accessToken, shell?.platformFeatures?.ffTaxCollection, courseId) {
        val token = accessToken ?: return@LaunchedEffect
        if (shell?.platformFeatures?.ffTaxCollection != true) return@LaunchedEffect
        loadingQuote = true
        quote = runCatching { LmsApi.fetchCheckoutQuote(courseId, token) }.getOrNull()
        loadingQuote = false
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
                            val result = LmsApi.startCheckout(
                                courseId = courseId,
                                successUrl = BillingLogic.checkoutSuccessUrl(courseId),
                                cancelUrl = BillingLogic.checkoutCancelUrl(),
                                usePaymentsAbstraction = shell?.platformFeatures?.ffPaymentsEnabled == true,
                                accessToken = token,
                            )
                            onDismiss()
                            BillingCheckout.openCheckoutUrl(context, result.checkoutUrl)
                        } catch (_: Exception) {
                            shell?.pendingCheckout = null
                            errorMessage = L.text(
                                context,
                                localePrefs,
                                com.lextures.android.R.string.mobile_billing_checkoutError,
                            )
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