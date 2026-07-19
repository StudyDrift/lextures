package com.lextures.android.features.marketplace

import android.content.Intent
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
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
import androidx.core.net.toUri
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.config.AppConfiguration
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalLocalePreferences
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.MarketplaceCourseDetail
import com.lextures.android.core.lms.MarketplaceLogic
import com.lextures.android.core.lms.MarketplaceObservability
import com.lextures.android.core.navigation.CourseWorkspaceSection
import com.lextures.android.core.navigation.RootDestination
import com.lextures.android.core.network.ApiError
import com.lextures.android.features.billing.PurchaseFlowSheet
import com.lextures.android.features.courses.CourseDetailScreen
import com.lextures.android.features.home.CourseHeroImage
import com.lextures.android.features.home.HomeShellState
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsSkeletonList
import kotlinx.coroutines.launch

@Composable
fun MarketplaceDetailScreen(
    session: AuthSession,
    shell: HomeShellState,
    slug: String,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    val scope = rememberCoroutineScope()

    var detail by remember { mutableStateOf<MarketplaceCourseDetail?>(null) }
    var loading by remember { mutableStateOf(true) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var claiming by remember { mutableStateOf(false) }
    var claimError by remember { mutableStateOf<String?>(null) }
    var openCourse by remember { mutableStateOf<CourseSummary?>(null) }
    var showPurchase by remember { mutableStateOf(false) }

    val purchaseEnabled = MarketplaceLogic.purchaseEnabled(shell.platformFeatures)

    LaunchedEffect(accessToken, slug) {
        MarketplaceObservability.record("marketplace_viewed")
        loading = true
        errorMessage = null
        try {
            val token = accessToken
            if (token == null) {
                errorMessage = L.text(context, localePrefs, R.string.mobile_marketplace_signInRequired)
            } else {
                val loaded = LmsApi.fetchMarketplaceCourseDetail(slug, token)
                if (loaded == null) {
                    errorMessage = L.text(context, localePrefs, R.string.mobile_marketplace_landingNotFound)
                } else {
                    detail = loaded
                }
            }
        } catch (_: Exception) {
            errorMessage = L.text(context, localePrefs, R.string.mobile_marketplace_landingError)
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
        errorMessage != null && detail == null -> LmsEmptyState(
            icon = Icons.Default.Warning,
            title = L.text(context, localePrefs, R.string.mobile_marketplace_landingErrorTitle),
            message = errorMessage!!,
            modifier = modifier.fillMaxSize(),
        )
        detail != null -> {
            val current = detail!!
            val course = current.course
            val freeLabel = L.text(context, localePrefs, R.string.mobile_marketplace_free)
            val priceLabel = MarketplaceLogic.formatPrice(current.priceCents, current.priceCurrency, freeLabel)
            Column(
                modifier = modifier
                    .fillMaxSize()
                    .verticalScroll(rememberScrollState())
                    .padding(16.dp),
                verticalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                LmsCard {
                    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                        CourseHeroImage(
                            url = course.heroImageUrl,
                            fallbackKey = course.courseCode,
                            accessToken = accessToken,
                            height = 160.dp,
                        )
                        Text(course.title, fontSize = 20.sp, fontWeight = FontWeight.Bold, color = textPrimary())
                        course.instructorName?.takeIf { it.isNotBlank() }?.let {
                            Text(
                                context.getString(R.string.mobile_marketplace_instructor, it),
                                fontSize = 12.sp,
                                color = textSecondary(),
                            )
                        }
                        Text(
                            context.getString(R.string.mobile_marketplace_enrolledCount, course.enrollmentCount),
                            fontSize = 12.sp,
                            color = textSecondary(),
                        )
                        if (course.owned) {
                            Text(
                                L.text(context, localePrefs, R.string.mobile_marketplace_owned),
                                fontSize = 12.sp,
                                fontWeight = FontWeight.SemiBold,
                                color = textSecondary(),
                            )
                        }
                    }
                }

                Text(
                    L.text(context, localePrefs, R.string.mobile_marketplace_aboutTitle),
                    fontSize = 16.sp,
                    fontWeight = FontWeight.SemiBold,
                    color = textPrimary(),
                )
                LmsCard {
                    val paragraphs = MarketplaceLogic.previewParagraphs(course.description)
                    if (paragraphs.isEmpty()) {
                        Text(
                            L.text(context, localePrefs, R.string.mobile_marketplace_noDescription),
                            fontSize = 14.sp,
                            color = textSecondary(),
                        )
                    } else {
                        paragraphs.forEach { paragraph ->
                            Text(paragraph, fontSize = 14.sp, color = textSecondary())
                        }
                    }
                }

                Text(
                    L.text(context, localePrefs, R.string.mobile_marketplace_whatsIncluded),
                    fontSize = 16.sp,
                    fontWeight = FontWeight.SemiBold,
                    color = textPrimary(),
                )
                LmsCard {
                    Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
                        Text(
                            context.getString(
                                R.string.mobile_marketplace_modulesCount,
                                current.whatsIncluded.moduleCount,
                            ),
                            fontSize = 14.sp,
                            color = textSecondary(),
                        )
                        Text(
                            context.getString(
                                R.string.mobile_marketplace_itemsCount,
                                current.whatsIncluded.itemCount,
                            ),
                            fontSize = 14.sp,
                            color = textSecondary(),
                        )
                    }
                }

                claimError?.let { LmsErrorBanner(message = it) }

                LmsCard {
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween,
                    ) {
                        Text(priceLabel, fontSize = 20.sp, fontWeight = FontWeight.Bold, color = textPrimary())
                        when {
                            course.owned -> Button(onClick = {
                                val token = accessToken ?: return@Button
                                scope.launch {
                                    runCatching {
                                        val summary = LmsApi.fetchCourse(course.courseCode, token)
                                        shell.select(RootDestination.Courses)
                                        shell.activeCourse = summary
                                        shell.activeCourseSection = CourseWorkspaceSection.Modules
                                        openCourse = summary
                                    }.onFailure {
                                        claimError = L.text(context, localePrefs, R.string.mobile_marketplace_openCourseError)
                                    }
                                }
                            }) {
                                Text(L.text(context, localePrefs, R.string.mobile_marketplace_goToCourse))
                            }
                            MarketplaceLogic.isPaid(current.priceCents) -> {
                                if (purchaseEnabled) {
                                    Button(
                                        onClick = { showPurchase = true },
                                        enabled = accessToken != null,
                                    ) {
                                        Text(L.text(context, localePrefs, R.string.mobile_marketplace_buy))
                                    }
                                } else {
                                    Button(onClick = {
                                        val url = AppConfiguration.webUrl(MarketplaceLogic.marketplaceWebPath(slug))
                                        context.startActivity(Intent(Intent.ACTION_VIEW, url.toUri()))
                                    }) {
                                        Text(L.text(context, localePrefs, R.string.mobile_marketplace_buyOnWeb))
                                    }
                                }
                            }
                            else -> Button(
                                onClick = {
                                    val token = accessToken ?: return@Button
                                    claiming = true
                                    claimError = null
                                    scope.launch {
                                        try {
                                            val result = LmsApi.claimMarketplaceCourse(slug, token)
                                            val summary = LmsApi.fetchCourse(
                                                result.courseCode.ifBlank { course.courseCode },
                                                token,
                                            )
                                            shell.select(RootDestination.Courses)
                                            shell.activeCourse = summary
                                            shell.activeCourseSection = CourseWorkspaceSection.Modules
                                            openCourse = summary
                                        } catch (e: ApiError.HttpStatus) {
                                            claimError = if (e.code == 402) {
                                                L.text(context, localePrefs, R.string.mobile_marketplace_claimPaidError)
                                            } else {
                                                L.text(context, localePrefs, R.string.mobile_marketplace_claimError)
                                            }
                                        } catch (_: Exception) {
                                            claimError = L.text(context, localePrefs, R.string.mobile_marketplace_claimError)
                                        } finally {
                                            claiming = false
                                        }
                                    }
                                },
                                enabled = !claiming,
                            ) {
                                Text(
                                    if (claiming) {
                                        L.text(context, localePrefs, R.string.mobile_marketplace_claiming)
                                    } else {
                                        L.text(context, localePrefs, R.string.mobile_marketplace_enrollFree)
                                    },
                                )
                            }
                        }
                    }
                }

                if (MarketplaceLogic.isPaid(current.priceCents) && !course.owned) {
                    Text(
                        L.text(
                            context,
                            localePrefs,
                            if (purchaseEnabled) {
                                R.string.mobile_marketplace_paidCheckoutHint
                            } else {
                                R.string.mobile_marketplace_paidWebHint
                            },
                        ),
                        fontSize = 12.sp,
                        color = textSecondary(),
                    )
                }
            }

            if (showPurchase) {
                PurchaseFlowSheet(
                    session = session,
                    shell = shell,
                    localePrefs = localePrefs,
                    courseId = current.course.id,
                    courseCode = current.course.courseCode,
                    title = current.course.title,
                    priceCents = current.priceCents,
                    currency = current.priceCurrency,
                    marketplaceSlug = slug,
                    onDismiss = { showPurchase = false },
                    onAlreadyOwned = {
                        val token = accessToken ?: return@PurchaseFlowSheet
                        scope.launch {
                            runCatching {
                                val summary = LmsApi.fetchCourse(current.course.courseCode, token)
                                detail = current.copy(course = current.course.copy(owned = true), owned = true)
                                shell.select(RootDestination.Courses)
                                shell.activeCourse = summary
                                shell.activeCourseSection = CourseWorkspaceSection.Modules
                                openCourse = summary
                            }.onFailure {
                                claimError = L.text(context, localePrefs, R.string.mobile_marketplace_openCourseError)
                            }
                        }
                    },
                )
            }
        }
    }
}
