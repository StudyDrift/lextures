package com.lextures.android.features.courses.settings

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
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
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalLocalePreferences
import com.lextures.android.core.lms.CourseCatalogListing
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.MarketplaceLogic
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsSkeletonList
import kotlinx.coroutines.launch

@Composable
fun CourseMarketplaceSettingsScreen(
    session: AuthSession,
    course: CourseSummary,
) {
    val accessToken by session.accessToken.collectAsState()
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    val scope = rememberCoroutineScope()

    var listing by remember { mutableStateOf<CourseCatalogListing?>(null) }
    var marketplaceListed by remember { mutableStateOf(false) }
    var amount by remember { mutableStateOf("") }
    var currency by remember { mutableStateOf("usd") }
    var loading by remember { mutableStateOf(true) }
    var saving by remember { mutableStateOf(false) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var amountError by remember { mutableStateOf<String?>(null) }
    var currencyMenu by remember { mutableStateOf(false) }
    var savedMessage by remember { mutableStateOf<String?>(null) }

    suspend fun reload() {
        val token = accessToken ?: return
        loading = true
        errorMessage = null
        try {
            val data = LmsApi.fetchCourseCatalogListing(course.courseCode, token)
            listing = data
            marketplaceListed = data.marketplaceListed
            amount = MarketplaceLogic.priceCentsToMajorUnits(data.priceCents)
            currency = data.priceCurrency.ifBlank { "usd" }
            amountError = null
        } catch (_: Exception) {
            errorMessage = L.text(context, localePrefs, R.string.mobile_courseSettings_marketplace_loadError)
            listing = null
        } finally {
            loading = false
        }
    }

    LaunchedEffect(accessToken, course.courseCode) {
        reload()
    }

    when {
        loading && listing == null -> LmsSkeletonList(count = 3)
        listing == null -> {
            errorMessage?.let { LmsErrorBanner(message = it) }
        }
        else -> {
            val current = listing!!
            val isDraft = current.publishState == "draft"
            Column(
                modifier = Modifier
                    .fillMaxSize()
                    .verticalScroll(rememberScrollState())
                    .padding(16.dp),
                verticalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                Text(
                    L.text(context, localePrefs, R.string.mobile_courseSettings_marketplace_title),
                    fontSize = 18.sp,
                    fontWeight = FontWeight.SemiBold,
                    color = textPrimary(),
                )
                Text(
                    L.text(context, localePrefs, R.string.mobile_courseSettings_marketplace_description),
                    fontSize = 13.sp,
                    color = textSecondary(),
                )

                errorMessage?.let { LmsErrorBanner(message = it) }
                savedMessage?.let {
                    Text(it, fontSize = 13.sp, color = textSecondary())
                }

                LmsCard {
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween,
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Column(modifier = Modifier.weight(1f)) {
                            Text(
                                L.text(context, localePrefs, R.string.mobile_courseSettings_marketplace_listToggle),
                                fontWeight = FontWeight.SemiBold,
                                color = textPrimary(),
                            )
                            Text(
                                if (isDraft) {
                                    L.text(context, localePrefs, R.string.mobile_courseSettings_marketplace_publishFirst)
                                } else {
                                    L.text(context, localePrefs, R.string.mobile_courseSettings_marketplace_listHelp)
                                },
                                fontSize = 12.sp,
                                color = textSecondary(),
                            )
                        }
                        Switch(
                            checked = marketplaceListed,
                            onCheckedChange = { marketplaceListed = it },
                            enabled = !saving && !isDraft,
                        )
                    }
                }

                LmsCard {
                    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                        Text(
                            L.text(context, localePrefs, R.string.mobile_courseSettings_marketplace_fee),
                            fontWeight = FontWeight.SemiBold,
                            color = textPrimary(),
                        )
                        OutlinedTextField(
                            value = amount,
                            onValueChange = {
                                amount = it
                                amountError = null
                            },
                            modifier = Modifier.fillMaxWidth(),
                            placeholder = {
                                Text(L.text(context, localePrefs, R.string.mobile_courseSettings_marketplace_feePlaceholder))
                            },
                            singleLine = true,
                            enabled = !saving,
                            isError = amountError != null,
                        )
                        amountError?.let {
                            Text(it, fontSize = 12.sp, color = textSecondary())
                        }
                        TextButton(onClick = { currencyMenu = true }, enabled = !saving) {
                            Text(currency.uppercase())
                        }
                        DropdownMenu(expanded = currencyMenu, onDismissRequest = { currencyMenu = false }) {
                            MarketplaceLogic.currencies.forEach { code ->
                                DropdownMenuItem(
                                    text = { Text(code.uppercase()) },
                                    onClick = {
                                        currency = code
                                        currencyMenu = false
                                    },
                                )
                            }
                        }
                        val previewCents = if (amount.trim().isEmpty()) {
                            0
                        } else {
                            MarketplaceLogic.majorUnitsToPriceCents(amount) ?: current.priceCents
                        }
                        Text(
                            MarketplaceLogic.formatPrice(
                                previewCents,
                                currency,
                                L.text(context, localePrefs, R.string.mobile_marketplace_free),
                            ),
                            fontSize = 13.sp,
                            color = textSecondary(),
                        )
                    }
                }

                Button(
                    onClick = {
                        val token = accessToken ?: return@Button
                        val validation = MarketplaceLogic.validateAmount(amount)
                        if (validation != null) {
                            amountError = when (validation) {
                                "min" -> L.text(context, localePrefs, R.string.mobile_courseSettings_marketplace_amountMin)
                                "max" -> L.text(context, localePrefs, R.string.mobile_courseSettings_marketplace_amountMax)
                                else -> L.text(context, localePrefs, R.string.mobile_courseSettings_marketplace_amountInvalid)
                            }
                            return@Button
                        }
                        val nextCents = if (amount.trim().isEmpty()) {
                            0
                        } else {
                            MarketplaceLogic.majorUnitsToPriceCents(amount) ?: current.priceCents
                        }
                        saving = true
                        savedMessage = null
                        errorMessage = null
                        scope.launch {
                            try {
                                val updated = LmsApi.putCourseCatalogListing(
                                    course.courseCode,
                                    MarketplaceLogic.buildListingPutBody(
                                        listing = current,
                                        marketplaceListed = marketplaceListed,
                                        priceCents = nextCents,
                                        priceCurrency = currency,
                                    ),
                                    token,
                                )
                                listing = updated
                                marketplaceListed = updated.marketplaceListed
                                amount = MarketplaceLogic.priceCentsToMajorUnits(updated.priceCents)
                                currency = updated.priceCurrency.ifBlank { "usd" }
                                savedMessage = L.text(context, localePrefs, R.string.mobile_courseSettings_marketplace_saved)
                            } catch (_: Exception) {
                                errorMessage = L.text(context, localePrefs, R.string.mobile_courseSettings_marketplace_saveError)
                                reload()
                            } finally {
                                saving = false
                            }
                        }
                    },
                    enabled = !saving,
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    Text(
                        if (saving) {
                            L.text(context, localePrefs, R.string.mobile_courseSettings_marketplace_saving)
                        } else {
                            L.text(context, localePrefs, R.string.mobile_courseSettings_marketplace_save)
                        },
                    )
                }
            }
        }
    }
}
