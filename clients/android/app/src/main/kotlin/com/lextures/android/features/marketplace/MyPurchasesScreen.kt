package com.lextures.android.features.marketplace

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.ShoppingCart
import androidx.compose.material.icons.filled.Warning
import androidx.compose.material3.Button
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
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
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalLocalePreferences
import com.lextures.android.core.lms.CoursePurchase
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.MarketplaceLogic
import com.lextures.android.core.navigation.CourseWorkspaceSection
import com.lextures.android.core.navigation.RootDestination
import com.lextures.android.features.courses.CourseDetailScreen
import com.lextures.android.features.home.HomeShellState
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsSkeletonList
import kotlinx.coroutines.launch

@Composable
fun MyPurchasesScreen(
    session: AuthSession,
    shell: HomeShellState,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    val scope = rememberCoroutineScope()

    var purchases by remember { mutableStateOf<List<CoursePurchase>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var openError by remember { mutableStateOf<String?>(null) }
    var openCourse by remember { mutableStateOf<CourseSummary?>(null) }

    LaunchedEffect(accessToken, shell.platformFeatures.ffCourseMarketplace) {
        loading = true
        errorMessage = null
        try {
            val token = accessToken
            if (token == null) {
                errorMessage = L.text(context, localePrefs, R.string.mobile_marketplace_signInRequired)
            } else if (!shell.platformFeatures.ffCourseMarketplace) {
                purchases = emptyList()
            } else {
                purchases = LmsApi.fetchMyPurchases(token)
            }
        } catch (_: Exception) {
            errorMessage = L.text(context, localePrefs, R.string.mobile_marketplace_purchases_error)
            purchases = emptyList()
        } finally {
            loading = false
        }
    }

    if (openCourse != null) {
        CourseDetailScreen(
            session = session,
            course = openCourse!!,
            onBack = {
                openCourse = null
                onBack()
            },
            modifier = modifier.fillMaxSize(),
        )
        return
    }

    when {
        loading -> LmsSkeletonList(count = 3, modifier = modifier.fillMaxSize())
        !shell.platformFeatures.ffCourseMarketplace -> LmsEmptyState(
            icon = Icons.Default.ShoppingCart,
            title = L.text(context, localePrefs, R.string.mobile_marketplace_unavailable),
            message = L.text(context, localePrefs, R.string.mobile_marketplace_purchases_disabledBody),
            modifier = modifier.fillMaxSize(),
        )
        errorMessage != null && purchases.isEmpty() -> LmsEmptyState(
            icon = Icons.Default.Warning,
            title = L.text(context, localePrefs, R.string.mobile_marketplace_purchases_errorTitle),
            message = errorMessage!!,
            modifier = modifier.fillMaxSize(),
        )
        purchases.isEmpty() -> LmsEmptyState(
            icon = Icons.Default.ShoppingCart,
            title = L.text(context, localePrefs, R.string.mobile_marketplace_purchases_emptyTitle),
            message = L.text(context, localePrefs, R.string.mobile_marketplace_purchases_emptyMessage),
            modifier = modifier.fillMaxSize(),
        )
        else -> {
            val freeLabel = L.text(context, localePrefs, R.string.mobile_marketplace_free)
            Column(
                modifier = modifier
                    .fillMaxSize()
                    .verticalScroll(rememberScrollState())
                    .padding(16.dp),
                verticalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                openError?.let { LmsErrorBanner(message = it) }
                Text(
                    L.text(context, localePrefs, R.string.mobile_marketplace_purchases_subtitle),
                    fontSize = 14.sp,
                    color = textSecondary(),
                )
                purchases.forEach { purchase ->
                    val priceLabel = MarketplaceLogic.formatPrice(
                        purchase.priceCents,
                        purchase.currency,
                        freeLabel,
                    )
                    val sourceKey = MarketplaceLogic.purchaseSourceLabelKey(purchase.source)
                    val sourceLabel = when (sourceKey) {
                        "mobile.marketplace.purchases.source.free" ->
                            L.text(context, localePrefs, R.string.mobile_marketplace_purchases_source_free)
                        "mobile.marketplace.purchases.source.stripe" ->
                            L.text(context, localePrefs, R.string.mobile_marketplace_purchases_source_stripe)
                        "mobile.marketplace.purchases.source.comp" ->
                            L.text(context, localePrefs, R.string.mobile_marketplace_purchases_source_comp)
                        else ->
                            L.text(context, localePrefs, R.string.mobile_marketplace_purchases_source_other)
                    }
                    LmsCard {
                        Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                            Text(purchase.title, fontWeight = FontWeight.Bold, color = textPrimary())
                            Row(
                                modifier = Modifier.fillMaxWidth(),
                                horizontalArrangement = Arrangement.SpaceBetween,
                            ) {
                                Text("$priceLabel · $sourceLabel", fontSize = 12.sp, color = textSecondary())
                                Text(
                                    MarketplaceLogic.formatAcquiredAt(purchase.acquiredAt),
                                    fontSize = 12.sp,
                                    color = textSecondary(),
                                )
                            }
                            Button(onClick = {
                                val token = accessToken ?: return@Button
                                openError = null
                                scope.launch {
                                    runCatching {
                                        val summary = LmsApi.fetchCourse(purchase.courseCode, token)
                                        shell.select(RootDestination.Courses)
                                        shell.activeCourse = summary
                                        shell.activeCourseSection = CourseWorkspaceSection.Modules
                                        openCourse = summary
                                    }.onFailure {
                                        openError = L.text(
                                            context,
                                            localePrefs,
                                            R.string.mobile_marketplace_openCourseError,
                                        )
                                    }
                                }
                            }) {
                                Text(L.text(context, localePrefs, R.string.mobile_marketplace_goToCourse))
                            }
                        }
                    }
                }
            }
        }
    }
}
