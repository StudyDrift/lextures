package com.lextures.android.features.billing

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CreditCard
import androidx.compose.material.icons.filled.List
import androidx.compose.material.icons.filled.ShoppingCart
import androidx.compose.material3.Button
import androidx.compose.material3.Divider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalePreferences
import com.lextures.android.core.lms.BillingEntitlement
import com.lextures.android.core.lms.BillingLogic
import com.lextures.android.core.lms.BillingTransaction
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.features.home.HomeShellState
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsSectionHeader
import com.lextures.android.features.home.LmsSkeletonList
import com.lextures.android.core.design.textSecondary
import kotlinx.coroutines.launch

@Composable
fun BillingScreen(
    session: AuthSession,
    shell: HomeShellState?,
    localePrefs: LocalePreferences,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val accessToken = session.accessToken.value

    var entitlements by remember { mutableStateOf<List<BillingEntitlement>>(emptyList()) }
    var transactions by remember { mutableStateOf<List<BillingTransaction>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var portalLoading by remember { mutableStateOf(false) }
    var errorMessage by remember { mutableStateOf<String?>(null) }

    suspend fun load() {
        val token = accessToken ?: return
        loading = true
        errorMessage = null
        try {
            entitlements = LmsApi.fetchMyEntitlements(token)
            transactions = if (shell?.platformFeatures?.ffPaymentsEnabled == true) {
                LmsApi.fetchMyTransactions(token)
            } else {
                emptyList()
            }
        } catch (_: Exception) {
            errorMessage = L.text(context, localePrefs, com.lextures.android.R.string.mobile_billing_loadError)
        } finally {
            loading = false
        }
    }

    LaunchedEffect(accessToken, shell?.platformFeatures) {
        load()
    }

    Column(
        modifier = modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        if (loading) {
            LmsSkeletonList(count = 3)
            return@Column
        }

        errorMessage?.let { LmsErrorBanner(message = it) }

        LmsSectionHeader(
            title = L.text(context, localePrefs, com.lextures.android.R.string.mobile_billing_subscription),
            icon = Icons.Default.CreditCard,
        )
        LmsCard {
            Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
                val active = BillingLogic.activeSubscription(entitlements)
                if (active != null) {
                    Text(
                        context.getString(
                            com.lextures.android.R.string.mobile_billing_subscriptionActive,
                            BillingLogic.entitlementLabelRes(active.entitlementType)?.let {
                                L.text(context, localePrefs, it)
                            } ?: active.entitlementType,
                        ),
                        fontWeight = FontWeight.SemiBold,
                        color = MaterialTheme.colorScheme.primary,
                    )
                } else {
                    Text(
                        L.text(context, localePrefs, com.lextures.android.R.string.mobile_billing_noSubscription),
                        color = textSecondary(),
                    )
                }
                Button(
                    onClick = {
                        val token = accessToken ?: return@Button
                        portalLoading = true
                        errorMessage = null
                        scope.launch {
                            try {
                                val url = LmsApi.openBillingPortal(BillingLogic.billingReturnUrl(), token)
                                BillingCheckout.openPortalUrl(context, url)
                            } catch (_: Exception) {
                                errorMessage = L.text(
                                    context,
                                    localePrefs,
                                    com.lextures.android.R.string.mobile_billing_portalError,
                                )
                            } finally {
                                portalLoading = false
                            }
                        }
                    },
                    enabled = !portalLoading && accessToken != null,
                ) {
                    Text(
                        if (portalLoading) {
                            L.text(context, localePrefs, com.lextures.android.R.string.mobile_billing_openingPortal)
                        } else {
                            L.text(context, localePrefs, com.lextures.android.R.string.mobile_billing_manageSubscription)
                        },
                    )
                }
            }
        }

        LmsSectionHeader(
            title = L.text(context, localePrefs, com.lextures.android.R.string.mobile_billing_purchaseHistory),
            icon = Icons.Default.List,
        )

        when {
            transactions.isEmpty() && entitlements.isEmpty() -> {
                LmsEmptyState(
                    icon = Icons.Default.ShoppingCart,
                    title = L.text(context, localePrefs, com.lextures.android.R.string.mobile_billing_noPurchasesTitle),
                    message = L.text(context, localePrefs, com.lextures.android.R.string.mobile_billing_noPurchasesMessage),
                )
            }
            transactions.isNotEmpty() -> {
                LmsCard {
                    Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
                        transactions.forEachIndexed { index, tx ->
                            Row(
                                modifier = Modifier.fillMaxWidth(),
                                horizontalArrangement = Arrangement.SpaceBetween,
                            ) {
                                Column {
                                    Text(tx.provider.replaceFirstChar { it.uppercase() }, fontWeight = FontWeight.SemiBold)
                                    Text(tx.createdAt.take(10), fontSize = 12.sp, color = textSecondary())
                                }
                                Column(horizontalAlignment = Alignment.End) {
                                    Text(
                                        BillingLogic.formatMoney(tx.amountCents, tx.currency),
                                        fontWeight = FontWeight.SemiBold,
                                    )
                                    Text(tx.status.replaceFirstChar { it.uppercase() }, fontSize = 12.sp, color = textSecondary())
                                }
                            }
                            if (index < transactions.lastIndex) Divider()
                        }
                    }
                }
            }
            else -> {
                LmsCard {
                    Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
                        entitlements.forEachIndexed { index, item ->
                            Row(
                                modifier = Modifier.fillMaxWidth(),
                                horizontalArrangement = Arrangement.SpaceBetween,
                            ) {
                                Column {
                                    Text(
                                        BillingLogic.entitlementLabelRes(item.entitlementType)?.let {
                                            L.text(context, localePrefs, it)
                                        } ?: item.entitlementType,
                                        fontWeight = FontWeight.SemiBold,
                                    )
                                    Text(item.validFrom.take(10), fontSize = 12.sp, color = textSecondary())
                                }
                                Column(horizontalAlignment = Alignment.End) {
                                    Text(
                                        BillingLogic.formatMoney(item.amountPaidCents, item.currency),
                                        fontWeight = FontWeight.SemiBold,
                                    )
                                    item.taxAmountCents?.takeIf { it > 0 }?.let { tax ->
                                        Text(
                                            context.getString(
                                                com.lextures.android.R.string.mobile_billing_taxLine,
                                                BillingLogic.formatMoney(tax, item.currency),
                                            ),
                                            fontSize = 11.sp,
                                            color = textSecondary(),
                                        )
                                    }
                                }
                            }
                            if (index < entitlements.lastIndex) Divider()
                        }
                    }
                }
            }
        }

        shell?.profile?.email?.let { email ->
            Text(
                context.getString(com.lextures.android.R.string.mobile_billing_signedInAs, email),
                fontSize = 12.sp,
                color = textSecondary(),
            )
        }
    }
}