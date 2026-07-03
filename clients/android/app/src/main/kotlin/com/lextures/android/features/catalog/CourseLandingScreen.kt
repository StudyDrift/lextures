package com.lextures.android.features.catalog

import android.content.Intent
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
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
import com.lextures.android.core.network.ApiError
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Warning
import com.lextures.android.core.lms.BillingLogic
import com.lextures.android.core.lms.CatalogLogic
import com.lextures.android.core.lms.CourseReview
import com.lextures.android.core.lms.CourseReviewLogic
import com.lextures.android.core.lms.CourseReviewsListResponse
import com.lextures.android.core.lms.ReviewEligibility
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.PublicCatalogCourse
import com.lextures.android.core.navigation.CourseWorkspaceSection
import com.lextures.android.core.navigation.RootDestination
import com.lextures.android.features.courses.CourseDetailScreen
import com.lextures.android.features.home.CourseHeroImage
import com.lextures.android.features.home.HomeShellState
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.billing.PurchaseFlowSheet
import com.lextures.android.features.home.LmsSkeletonList
import kotlinx.coroutines.launch

@Composable
fun CourseLandingScreen(
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

    var course by remember { mutableStateOf<PublicCatalogCourse?>(null) }
    var reviews by remember { mutableStateOf<CourseReviewsListResponse?>(null) }
    var enrolledCourses by remember { mutableStateOf<List<CourseSummary>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var enrolling by remember { mutableStateOf(false) }
    var enrollError by remember { mutableStateOf<String?>(null) }
    var openCourse by remember { mutableStateOf<CourseSummary?>(null) }
    var showPurchase by remember { mutableStateOf(false) }
    var showReviewComposer by remember { mutableStateOf(false) }
    var reviewEligibility by remember { mutableStateOf<ReviewEligibility?>(null) }

    val isEnrolled = course?.let { CatalogLogic.isEnrolled(it.courseCode, enrolledCourses) } == true

    LaunchedEffect(accessToken, slug) {
        loading = true
        errorMessage = null
        try {
            val token = accessToken
            if (token != null) {
                enrolledCourses = LmsApi.fetchCourses(token)
            }
            val detail = LmsApi.fetchPublicCatalogCourseDetail(slug, token)
            if (detail == null) {
                errorMessage = L.text(context, localePrefs, R.string.mobile_catalog_landingNotFound)
            } else {
                course = detail
                reviews = runCatching { LmsApi.fetchPublicCatalogCourseReviews(slug, accessToken = token) }.getOrNull()
                reviewEligibility = if (token != null && shell.platformFeatures.ffCourseReviews &&
                    CatalogLogic.isEnrolled(detail.courseCode, enrolledCourses)
                ) {
                    runCatching { LmsApi.fetchReviewEligibility(detail.courseCode, token) }.getOrNull()
                } else {
                    null
                }
            }
        } catch (e: Exception) {
            errorMessage = L.text(context, localePrefs, R.string.mobile_catalog_landingError)
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
        errorMessage != null && course == null -> LmsEmptyState(
            icon = Icons.Default.Warning,
            title = L.text(context, localePrefs, R.string.mobile_catalog_landingErrorTitle),
            message = errorMessage!!,
            modifier = modifier.fillMaxSize(),
        )
        course != null -> {
            val current = course!!
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
                            url = current.heroImageUrl,
                            fallbackKey = current.courseCode,
                            accessToken = accessToken,
                            height = 160.dp,
                        )
                        Text(current.title, fontSize = 20.sp, fontWeight = FontWeight.Bold, color = textPrimary())
                        current.instructorName?.takeIf { it.isNotBlank() }?.let {
                            Text(
                                context.getString(R.string.mobile_catalog_instructor, it),
                                fontSize = 12.sp,
                                color = textSecondary(),
                            )
                        }
                        Text(
                            context.getString(R.string.mobile_catalog_enrolledCount, current.enrollmentCount),
                            fontSize = 12.sp,
                            color = textSecondary(),
                        )
                    }
                }

                Text(
                    L.text(context, localePrefs, R.string.mobile_catalog_aboutTitle),
                    fontSize = 16.sp,
                    fontWeight = FontWeight.SemiBold,
                    color = textPrimary(),
                )
                LmsCard {
                    val paragraphs = CatalogLogic.previewParagraphs(current.description)
                    if (paragraphs.isEmpty()) {
                        Text(
                            L.text(context, localePrefs, R.string.mobile_catalog_noDescription),
                            fontSize = 14.sp,
                            color = textSecondary(),
                        )
                    } else {
                        paragraphs.forEach { paragraph ->
                            Text(paragraph, fontSize = 14.sp, color = textSecondary())
                        }
                    }
                }

                if (shell.platformFeatures.ffCourseReviews && reviews != null) {
                    Text(
                        L.text(context, localePrefs, R.string.mobile_catalog_reviewsTitle),
                        fontSize = 16.sp,
                        fontWeight = FontWeight.SemiBold,
                        color = textPrimary(),
                    )
                    LmsCard {
                        reviewsSection(
                            context = context,
                            localePrefs = localePrefs,
                            reviews = reviews!!,
                            reviewEligibility = reviewEligibility,
                            onWriteReview = { showReviewComposer = true },
                        )
                    }
                }

                enrollError?.let { LmsErrorBanner(message = it) }

                LmsCard {
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween,
                    ) {
                        Text(
                            CatalogLogic.formatPrice(current.priceCents),
                            fontSize = 20.sp,
                            fontWeight = FontWeight.Bold,
                            color = textPrimary(),
                        )
                        when {
                            isEnrolled -> Button(onClick = {
                                val summary = CatalogLogic.enrolledCourse(current.courseCode, enrolledCourses)
                                if (summary != null) {
                                    shell.select(RootDestination.Courses)
                                    shell.activeCourse = summary
                                    shell.activeCourseSection = CourseWorkspaceSection.Modules
                                    openCourse = summary
                                }
                            }) {
                                Text(L.text(context, localePrefs, R.string.mobile_catalog_continue))
                            }
                            CatalogLogic.isPaid(current.priceCents) -> {
                                if (BillingLogic.billingEnabled(shell.platformFeatures)) {
                                    Button(onClick = { showPurchase = true }) {
                                        Text(L.text(context, localePrefs, R.string.mobile_billing_purchase))
                                    }
                                } else {
                                    Button(onClick = {
                                        val url = AppConfiguration.webUrl(CatalogLogic.catalogWebPath(slug))
                                        context.startActivity(Intent(Intent.ACTION_VIEW, url.toUri()))
                                    }) {
                                        Text(L.text(context, localePrefs, R.string.mobile_catalog_openOnWeb))
                                    }
                                }
                            }
                            else -> Button(
                                onClick = {
                                    val token = accessToken ?: return@Button
                                    enrolling = true
                                    enrollError = null
                                    scope.launch {
                                        try {
                                            LmsApi.selfEnrollInCourse(current.courseCode, token)
                                            val refreshed = LmsApi.fetchCourse(current.courseCode, token)
                                            enrolledCourses = enrolledCourses + refreshed
                                            shell.select(RootDestination.Courses)
                                            shell.activeCourse = refreshed
                                            shell.activeCourseSection = CourseWorkspaceSection.Modules
                                            openCourse = refreshed
                                        } catch (e: ApiError.HttpStatus) {
                                            enrollError = when (e.code) {
                                                402 -> L.text(context, localePrefs, R.string.mobile_catalog_paidRequired)
                                                403 -> L.text(context, localePrefs, R.string.mobile_catalog_enrollForbidden)
                                                else -> L.text(context, localePrefs, R.string.mobile_catalog_enrollError)
                                            }
                                        } catch (e: Exception) {
                                            enrollError = L.text(context, localePrefs, R.string.mobile_catalog_enrollError)
                                        } finally {
                                            enrolling = false
                                        }
                                    }
                                },
                                enabled = !enrolling && accessToken != null,
                            ) {
                                Text(
                                    if (enrolling) {
                                        L.text(context, localePrefs, R.string.mobile_catalog_enrolling)
                                    } else {
                                        L.text(context, localePrefs, R.string.mobile_catalog_enrollFree)
                                    },
                                )
                            }
                        }
                    }
                }
            }
        }
    }

    if (showReviewComposer) {
        course?.let { current ->
            ReviewComposerSheet(
                session = session,
                localePrefs = localePrefs,
                courseCode = current.courseCode,
                courseTitle = current.title,
                hasReview = reviewEligibility?.hasReview == true,
                canEdit = reviewEligibility?.canEdit == true,
                onDismiss = { showReviewComposer = false },
                onSubmitted = {
                    showReviewComposer = false
                    scope.launch {
                        reviews = runCatching {
                            LmsApi.fetchPublicCatalogCourseReviews(slug, accessToken = accessToken)
                        }.getOrNull()
                    }
                },
            )
        }
    }

    if (showPurchase) {
        course?.let { current ->
            PurchaseFlowSheet(
                session = session,
                shell = shell,
                localePrefs = localePrefs,
                courseId = current.id,
                courseCode = current.courseCode,
                title = current.title,
                priceCents = current.priceCents,
                currency = "USD",
                onDismiss = { showPurchase = false },
            )
        }
    }
}

@Composable
private fun reviewsSection(
    context: android.content.Context,
    localePrefs: com.lextures.android.core.i18n.LocalePreferences,
    reviews: CourseReviewsListResponse,
    reviewEligibility: ReviewEligibility?,
    onWriteReview: () -> Unit,
) {
    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
        reviewEligibility?.let { eligibility ->
            if (CourseReviewLogic.shouldShowComposer(eligibility)) {
                Button(onClick = onWriteReview) {
                    Text(L.text(context, localePrefs, R.string.mobile_reviews_writeCta))
                }
            } else if (!eligibility.eligible) {
                Text(
                    context.getString(R.string.mobile_reviews_progressHint, eligibility.progressPercent),
                    fontSize = 12.sp,
                    color = textSecondary(),
                )
            }
        }
        reviews.reviews.forEach { review ->
            reviewRow(context, localePrefs, review)
        }
    }
}

@Composable
private fun reviewRow(
    context: android.content.Context,
    localePrefs: com.lextures.android.core.i18n.LocalePreferences,
    review: CourseReview,
) {
    Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
        Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
            Text(review.reviewerDisplayName, fontSize = 12.sp, fontWeight = FontWeight.SemiBold, color = textPrimary())
            Text(
                context.getString(R.string.mobile_catalog_reviewStars, review.rating),
                fontSize = 11.sp,
                color = textSecondary(),
            )
        }
        review.reviewText?.takeIf { it.isNotBlank() }?.let {
            Text(it, fontSize = 12.sp, color = textSecondary())
        }
    }
}