package com.lextures.android.features.marketplace

import com.lextures.android.core.routing.LinkOpener
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
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.OutlinedTextField
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
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.liveRegion
import androidx.compose.ui.semantics.LiveRegionMode
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextDecoration
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.config.AppConfiguration
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalLocalePreferences
import com.lextures.android.core.lms.CouponApiError
import com.lextures.android.core.lms.CouponPreview
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

private enum class CouponFieldStatus {
    Idle,
    Checking,
    Applied,
    Rejected,
    RateLimited,
}

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

    var couponInput by remember { mutableStateOf("") }
    var couponStatus by remember { mutableStateOf(CouponFieldStatus.Idle) }
    var couponPreview by remember { mutableStateOf<CouponPreview?>(null) }
    var couponError by remember { mutableStateOf<String?>(null) }
    var couponAnnouncement by remember { mutableStateOf("") }
    var autoApplyDone by remember(slug) { mutableStateOf(false) }

    val purchaseEnabled = MarketplaceLogic.purchaseEnabled(shell.platformFeatures)
    val couponsOn = MarketplaceLogic.couponsEnabled(shell.platformFeatures)

    fun couponReasonMessage(reason: String?): String {
        val key = MarketplaceLogic.couponReasonKey(reason)
        val resId = couponReasonStringRes(key)
        return if (resId != 0) L.text(context, localePrefs, resId) else reason.orEmpty()
    }

    fun clearCoupon(clearShellPending: Boolean = true) {
        couponPreview = null
        couponStatus = CouponFieldStatus.Idle
        couponError = null
        couponInput = ""
        couponAnnouncement = ""
        if (clearShellPending && shell.pendingCoupon?.slug == slug) {
            shell.pendingCoupon = null
        }
    }

    fun applyPreviewResult(preview: CouponPreview, fromDeeplink: Boolean) {
        val code = MarketplaceLogic.normalizeCouponCode(preview.code.ifBlank { couponInput })
        couponInput = code
        if (preview.applied) {
            couponPreview = preview
            couponStatus = CouponFieldStatus.Applied
            couponError = null
            val freeLabel = L.text(context, localePrefs, R.string.mobile_marketplace_free)
            val priceText = MarketplaceLogic.formatPrice(preview.chargedCents, preview.currency, freeLabel)
            couponAnnouncement = L.format(
                context,
                localePrefs,
                R.string.mobile_marketplace_coupon_applied,
                code,
                priceText,
            )
            if (fromDeeplink) {
                MarketplaceObservability.record("coupon_from_deeplink", mapOf("result" to "ok"))
            }
        } else {
            couponPreview = null
            couponStatus = CouponFieldStatus.Rejected
            couponError = couponReasonMessage(preview.reason)
            if (fromDeeplink) {
                MarketplaceObservability.record(
                    "coupon_from_deeplink",
                    mapOf("result" to preview.reason.ifBlank { "not_found" }),
                )
            }
        }
    }

    fun runCouponPreview(codeRaw: String, fromDeeplink: Boolean) {
        val token = accessToken ?: return
        val code = MarketplaceLogic.normalizeCouponCode(codeRaw)
        if (code.isEmpty() || !couponsOn) return
        couponStatus = CouponFieldStatus.Checking
        couponError = null
        scope.launch {
            try {
                val preview = LmsApi.previewMarketplaceCoupon(slug, code, token)
                applyPreviewResult(preview, fromDeeplink)
            } catch (e: CouponApiError) {
                if (e.status == 429) {
                    couponStatus = CouponFieldStatus.RateLimited
                    couponPreview = null
                    couponError = L.text(context, localePrefs, R.string.mobile_marketplace_coupon_rateLimited)
                } else {
                    couponStatus = CouponFieldStatus.Rejected
                    couponPreview = null
                    couponError = couponReasonMessage(e.reason).ifBlank {
                        L.text(context, localePrefs, R.string.mobile_marketplace_coupon_reason_not_found)
                    }
                }
            } catch (_: Exception) {
                couponStatus = CouponFieldStatus.Rejected
                couponPreview = null
                couponError = L.text(context, localePrefs, R.string.mobile_marketplace_coupon_reason_not_found)
            }
        }
    }

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
                    if (loaded.owned || loaded.course.owned) {
                        if (shell.pendingCoupon?.slug == slug) shell.pendingCoupon = null
                        clearCoupon(clearShellPending = true)
                    }
                }
            }
        } catch (_: Exception) {
            errorMessage = L.text(context, localePrefs, R.string.mobile_marketplace_landingError)
        } finally {
            loading = false
        }
    }

    // Auto-apply pending / deep-link coupon once detail is loaded (flag off ⇒ ignore FR-11 / AC-11).
    LaunchedEffect(detail, couponsOn, accessToken, slug, autoApplyDone) {
        if (autoApplyDone || detail == null || accessToken == null) return@LaunchedEffect
        if (!couponsOn) {
            if (shell.pendingCoupon?.slug == slug) shell.pendingCoupon = null
            autoApplyDone = true
            return@LaunchedEffect
        }
        val course = detail ?: return@LaunchedEffect
        if (course.owned || course.course.owned || !MarketplaceLogic.isPaid(course.priceCents)) {
            if (shell.pendingCoupon?.slug == slug) shell.pendingCoupon = null
            autoApplyDone = true
            return@LaunchedEffect
        }
        val pending = shell.pendingCoupon
        val code = when {
            pending != null && pending.slug == slug && MarketplaceLogic.isValidCouponParam(pending.code) ->
                MarketplaceLogic.normalizeCouponCode(pending.code)
            else -> null
        }
        autoApplyDone = true
        if (code.isNullOrEmpty()) return@LaunchedEffect
        couponInput = code
        runCouponPreview(code, fromDeeplink = true)
    }

    if (openCourse != null) {
        CourseDetailScreen(
            session = session,
            course = openCourse!!,
            onBack = {
                openCourse = null
                onBack()
            },
            shell = shell,
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
            val couponApplied = couponStatus == CouponFieldStatus.Applied && couponPreview?.applied == true
            val displayPriceCents = if (couponApplied) couponPreview!!.chargedCents else current.priceCents
            val displayCurrency = if (couponApplied) couponPreview!!.currency else current.priceCurrency
            val listPriceForStrike = if (couponApplied) {
                couponPreview!!.listPriceCents
            } else {
                current.listPriceCents ?: current.priceCents
            }
            val priceLabel = MarketplaceLogic.formatPrice(displayPriceCents, displayCurrency, freeLabel)
            val wasLabel = if (
                couponApplied &&
                listPriceForStrike > displayPriceCents
            ) {
                L.format(
                    context,
                    localePrefs,
                    R.string.mobile_marketplace_coupon_was,
                    MarketplaceLogic.formatPrice(listPriceForStrike, displayCurrency, freeLabel),
                )
            } else {
                null
            }
            val savingsLabel = if (
                couponApplied &&
                (couponPreview?.discountCents ?: 0) > 0
            ) {
                L.format(
                    context,
                    localePrefs,
                    R.string.mobile_marketplace_coupon_savings,
                    MarketplaceLogic.formatPrice(couponPreview!!.discountCents, displayCurrency, freeLabel),
                )
            } else {
                null
            }
            val priceA11y = if (wasLabel != null) {
                L.format(
                    context,
                    localePrefs,
                    R.string.mobile_marketplace_coupon_priceA11y,
                    priceLabel,
                    MarketplaceLogic.formatPrice(listPriceForStrike, displayCurrency, freeLabel),
                )
            } else {
                priceLabel
            }
            val freeAfterCoupon = couponApplied && (
                couponPreview!!.freeAfterDiscount || couponPreview!!.chargedCents <= 0
            )
            val activeCouponCode = if (couponApplied) couponPreview!!.code else null
            val showCouponUi = couponsOn &&
                !course.owned &&
                MarketplaceLogic.isPaid(current.priceCents)

            fun openOwnedCourse(courseCode: String) {
                val token = accessToken ?: return
                scope.launch {
                    runCatching {
                        val summary = LmsApi.fetchCourse(courseCode, token)
                        detail = current.copy(course = current.course.copy(owned = true), owned = true)
                        shell.pendingCoupon = null
                        clearCoupon(clearShellPending = false)
                        shell.select(RootDestination.Courses)
                        shell.activeCourse = summary
                        shell.activeCourseSection = CourseWorkspaceSection.Modules
                        openCourse = summary
                    }.onFailure {
                        claimError = L.text(context, localePrefs, R.string.mobile_marketplace_openCourseError)
                    }
                }
            }

            fun enrollFreeWithCoupon(code: String) {
                val token = accessToken ?: return
                claiming = true
                claimError = null
                scope.launch {
                    try {
                        MarketplaceObservability.record(
                            "coupon_checkout_started",
                            mapOf("discounted" to "1"),
                        )
                        val result = try {
                            LmsApi.claimMarketplaceCourse(slug, token, code)
                        } catch (_: Exception) {
                            val checkout = LmsApi.checkoutMarketplaceCourse(slug, token, code)
                            if (checkout.grantedFree || checkout.alreadyOwned || checkout.enrolled) {
                                com.lextures.android.core.lms.MarketplaceClaimResult(
                                    enrolled = checkout.enrolled || checkout.grantedFree || checkout.alreadyOwned,
                                    alreadyOwned = checkout.alreadyOwned,
                                    courseCode = checkout.courseCode.orEmpty(),
                                    grantedFree = checkout.grantedFree,
                                    firstItemId = checkout.firstItemId,
                                    entitlementId = checkout.entitlementId.orEmpty(),
                                )
                            } else {
                                throw ApiError.HttpStatus(500, "Free grant failed")
                            }
                        }
                        val codeForOpen = result.courseCode.ifBlank { course.courseCode }
                        shell.pendingCoupon = null
                        clearCoupon(clearShellPending = false)
                        openOwnedCourse(codeForOpen)
                    } catch (e: CouponApiError) {
                        if (e.status == 422) {
                            clearCoupon()
                            couponStatus = CouponFieldStatus.Rejected
                            couponError = couponReasonMessage(e.reason)
                            claimError = couponError
                        } else if (e.status == 429) {
                            couponStatus = CouponFieldStatus.RateLimited
                            couponError = L.text(context, localePrefs, R.string.mobile_marketplace_coupon_rateLimited)
                            claimError = couponError
                        } else {
                            claimError = L.text(context, localePrefs, R.string.mobile_marketplace_claimError)
                        }
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
            }

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

                if (showCouponUi) {
                    LmsCard {
                        Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                            Text(
                                L.text(context, localePrefs, R.string.mobile_marketplace_coupon_disclosure),
                                fontSize = 14.sp,
                                fontWeight = FontWeight.SemiBold,
                                color = textPrimary(),
                            )
                            when (couponStatus) {
                                CouponFieldStatus.Applied -> {
                                    Text(
                                        L.format(
                                            context,
                                            localePrefs,
                                            R.string.mobile_marketplace_coupon_applied,
                                            couponPreview?.code.orEmpty(),
                                            priceLabel,
                                        ),
                                        fontSize = 13.sp,
                                        color = textPrimary(),
                                    )
                                    savingsLabel?.let {
                                        Text(it, fontSize = 12.sp, color = textSecondary())
                                    }
                                    TextButton(onClick = { clearCoupon() }) {
                                        Text(L.text(context, localePrefs, R.string.mobile_marketplace_coupon_remove))
                                    }
                                }
                                else -> {
                                    Row(
                                        modifier = Modifier.fillMaxWidth(),
                                        horizontalArrangement = Arrangement.spacedBy(8.dp),
                                    ) {
                                        OutlinedTextField(
                                            value = couponInput,
                                            onValueChange = {
                                                couponInput = MarketplaceLogic.normalizeCouponCode(it)
                                                if (couponStatus == CouponFieldStatus.Rejected ||
                                                    couponStatus == CouponFieldStatus.RateLimited
                                                ) {
                                                    couponStatus = CouponFieldStatus.Idle
                                                    couponError = null
                                                }
                                            },
                                            label = {
                                                Text(L.text(context, localePrefs, R.string.mobile_marketplace_coupon_label))
                                            },
                                            placeholder = {
                                                Text(L.text(context, localePrefs, R.string.mobile_marketplace_coupon_placeholder))
                                            },
                                            singleLine = true,
                                            enabled = couponStatus != CouponFieldStatus.Checking,
                                            isError = couponError != null,
                                            supportingText = couponError?.let { { Text(it) } },
                                            modifier = Modifier.weight(1f),
                                        )
                                        Button(
                                            onClick = { runCouponPreview(couponInput, fromDeeplink = false) },
                                            enabled = couponInput.isNotBlank() &&
                                                couponStatus != CouponFieldStatus.Checking &&
                                                accessToken != null,
                                        ) {
                                            if (couponStatus == CouponFieldStatus.Checking) {
                                                CircularProgressIndicator(
                                                    modifier = Modifier.padding(4.dp),
                                                    strokeWidth = 2.dp,
                                                )
                                            } else {
                                                Text(
                                                    L.text(
                                                        context,
                                                        localePrefs,
                                                        if (couponStatus == CouponFieldStatus.Checking) {
                                                            R.string.mobile_marketplace_coupon_applying
                                                        } else {
                                                            R.string.mobile_marketplace_coupon_apply
                                                        },
                                                    ),
                                                )
                                            }
                                        }
                                    }
                                }
                            }
                            if (couponAnnouncement.isNotBlank()) {
                                Text(
                                    couponAnnouncement,
                                    modifier = Modifier.semantics {
                                        liveRegion = LiveRegionMode.Polite
                                        contentDescription = couponAnnouncement
                                    },
                                    fontSize = 1.sp,
                                    color = textSecondary().copy(alpha = 0f),
                                )
                            }
                        }
                    }
                }

                LmsCard {
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .semantics {
                                contentDescription = priceA11y
                                liveRegion = LiveRegionMode.Polite
                            },
                        horizontalArrangement = Arrangement.SpaceBetween,
                    ) {
                        Column {
                            Text(priceLabel, fontSize = 20.sp, fontWeight = FontWeight.Bold, color = textPrimary())
                            wasLabel?.let {
                                Text(
                                    it,
                                    fontSize = 12.sp,
                                    color = textSecondary(),
                                    textDecoration = TextDecoration.LineThrough,
                                )
                            }
                            savingsLabel?.let {
                                Text(it, fontSize = 12.sp, color = textSecondary())
                            }
                        }
                        when {
                            course.owned -> Button(onClick = {
                                openOwnedCourse(course.courseCode)
                            }) {
                                Text(L.text(context, localePrefs, R.string.mobile_marketplace_goToCourse))
                            }
                            MarketplaceLogic.isPaid(current.priceCents) -> {
                                if (purchaseEnabled) {
                                    if (freeAfterCoupon && activeCouponCode != null) {
                                        Button(
                                            onClick = { enrollFreeWithCoupon(activeCouponCode) },
                                            enabled = accessToken != null && !claiming,
                                        ) {
                                            Text(
                                                if (claiming) {
                                                    L.text(context, localePrefs, R.string.mobile_marketplace_claiming)
                                                } else {
                                                    L.text(context, localePrefs, R.string.mobile_marketplace_enrollFree)
                                                },
                                            )
                                        }
                                    } else {
                                        val buyLabel = if (couponApplied) {
                                            L.format(
                                                context,
                                                localePrefs,
                                                R.string.mobile_marketplace_buyWithPrice,
                                                priceLabel,
                                            )
                                        } else {
                                            L.text(context, localePrefs, R.string.mobile_marketplace_buy)
                                        }
                                        Button(
                                            onClick = { showPurchase = true },
                                            enabled = accessToken != null,
                                        ) {
                                            Text(buyLabel)
                                        }
                                    }
                                } else {
                                    Button(onClick = {
                                        val url = AppConfiguration.webUrl(MarketplaceLogic.marketplaceWebPath(slug))
                                        LinkOpener.open(context, url.toString(), null, "legacy")
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
                                            openOwnedCourse(result.courseCode.ifBlank { course.courseCode })
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
                val sheetPrice = if (couponApplied) couponPreview!!.chargedCents else current.priceCents
                val sheetCurrency = if (couponApplied) couponPreview!!.currency else current.priceCurrency
                PurchaseFlowSheet(
                    session = session,
                    shell = shell,
                    localePrefs = localePrefs,
                    courseId = current.course.id,
                    courseCode = current.course.courseCode,
                    title = current.course.title,
                    priceCents = sheetPrice,
                    currency = sheetCurrency,
                    marketplaceSlug = slug,
                    couponCode = activeCouponCode,
                    listPriceCents = if (couponApplied) couponPreview!!.listPriceCents else null,
                    discountCents = if (couponApplied) couponPreview!!.discountCents else null,
                    chargedCents = if (couponApplied) couponPreview!!.chargedCents else null,
                    onDismiss = { showPurchase = false },
                    onCouponRejected = { reason ->
                        showPurchase = false
                        clearCoupon()
                        couponStatus = CouponFieldStatus.Rejected
                        couponError = couponReasonMessage(reason)
                        claimError = couponError
                    },
                    onAlreadyOwned = {
                        openOwnedCourse(current.course.courseCode)
                    },
                    onGrantedFree = { courseCode ->
                        shell.pendingCoupon = null
                        clearCoupon(clearShellPending = false)
                        openOwnedCourse(courseCode.ifBlank { current.course.courseCode })
                    },
                )
            }
        }
    }
}

private fun couponReasonStringRes(key: String): Int = when (key) {
    "mobile.marketplace.coupon.reason.ok" -> R.string.mobile_marketplace_coupon_reason_ok
    "mobile.marketplace.coupon.reason.not_found" -> R.string.mobile_marketplace_coupon_reason_not_found
    "mobile.marketplace.coupon.reason.inactive" -> R.string.mobile_marketplace_coupon_reason_inactive
    "mobile.marketplace.coupon.reason.not_started" -> R.string.mobile_marketplace_coupon_reason_not_started
    "mobile.marketplace.coupon.reason.expired" -> R.string.mobile_marketplace_coupon_reason_expired
    "mobile.marketplace.coupon.reason.exhausted" -> R.string.mobile_marketplace_coupon_reason_exhausted
    "mobile.marketplace.coupon.reason.already_used" -> R.string.mobile_marketplace_coupon_reason_already_used
    "mobile.marketplace.coupon.reason.currency_mismatch" -> R.string.mobile_marketplace_coupon_reason_currency_mismatch
    "mobile.marketplace.coupon.reason.course_free" -> R.string.mobile_marketplace_coupon_reason_course_free
    "mobile.marketplace.coupon.reason.owned" -> R.string.mobile_marketplace_coupon_reason_owned
    else -> R.string.mobile_marketplace_coupon_reason_not_found
}
