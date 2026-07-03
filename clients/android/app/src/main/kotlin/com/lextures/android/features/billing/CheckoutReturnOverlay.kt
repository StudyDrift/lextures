package com.lextures.android.features.billing

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalePreferences
import com.lextures.android.core.lms.BillingLogic
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.CheckoutReturnPhase
import com.lextures.android.core.navigation.CourseWorkspaceSection
import com.lextures.android.core.navigation.RootDestination
import com.lextures.android.features.home.HomeShellState
import com.lextures.android.features.home.LmsCard
import com.lextures.android.core.design.textSecondary
import kotlinx.coroutines.delay

private enum class CheckoutStatus {
    Verifying,
    Ready,
    Timeout,
    Cancelled,
}

@Composable
fun CheckoutReturnOverlay(
    session: AuthSession,
    shell: HomeShellState,
    localePrefs: LocalePreferences,
    phase: CheckoutReturnPhase,
    onDismiss: () -> Unit,
) {
    val context = LocalContext.current
    val accessToken = session.accessToken.value
    var status by remember(phase) { mutableStateOf(CheckoutStatus.Verifying) }
    var message by remember { mutableStateOf<String?>(null) }

    LaunchedEffect(phase, accessToken) {
        when (phase) {
            CheckoutReturnPhase.Cancel -> status = CheckoutStatus.Cancelled
            is CheckoutReturnPhase.Success -> {
                val token = accessToken ?: run {
                    status = CheckoutStatus.Timeout
                    return@LaunchedEffect
                }
                val userId = shell.profile?.id ?: run {
                    status = CheckoutStatus.Timeout
                    return@LaunchedEffect
                }
                val targetCourseId = phase.courseId ?: shell.pendingCheckout?.courseId
                val courseCode = shell.pendingCheckout?.courseCode
                if (targetCourseId == null && courseCode == null) {
                    status = CheckoutStatus.Ready
                    shell.pendingCheckout = null
                    return@LaunchedEffect
                }
                repeat(BillingLogic.ENTITLEMENT_POLL_ATTEMPTS) { attempt ->
                    val entitled = targetCourseId?.let { courseId ->
                        runCatching { LmsApi.checkEntitlement(userId, courseId, token) }.getOrDefault(false)
                    } == true
                    if (entitled && courseCode != null) {
                        runCatching { LmsApi.fetchCourse(courseCode, token) }.onSuccess { course ->
                            shell.select(RootDestination.Courses)
                            shell.activeCourse = course
                            shell.activeCourseSection = CourseWorkspaceSection.Modules
                            message = course.title
                        }
                        shell.pendingCheckout = null
                        status = CheckoutStatus.Ready
                        return@LaunchedEffect
                    }
                    if (attempt < BillingLogic.ENTITLEMENT_POLL_ATTEMPTS - 1) {
                        delay(BillingLogic.ENTITLEMENT_POLL_INTERVAL_MS)
                    }
                }
                status = CheckoutStatus.Timeout
            }
        }
    }

    Box(
        modifier = Modifier
            .fillMaxSize()
            .background(Color.Black.copy(alpha = 0.35f)),
        contentAlignment = Alignment.Center,
    ) {
        LmsCard(modifier = Modifier.padding(24.dp)) {
            Column(
                horizontalAlignment = Alignment.CenterHorizontally,
                verticalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                when (status) {
                    CheckoutStatus.Verifying -> {
                        CircularProgressIndicator()
                        Text(
                            L.text(context, localePrefs, com.lextures.android.R.string.mobile_billing_verifyingPayment),
                            fontWeight = FontWeight.SemiBold,
                            textAlign = TextAlign.Center,
                        )
                        Text(
                            L.text(context, localePrefs, com.lextures.android.R.string.mobile_billing_verifyingHint),
                            fontSize = 12.sp,
                            color = textSecondary(),
                            textAlign = TextAlign.Center,
                        )
                    }
                    CheckoutStatus.Ready -> {
                        Text(
                            L.text(context, localePrefs, com.lextures.android.R.string.mobile_billing_paymentConfirmed),
                            fontWeight = FontWeight.SemiBold,
                            textAlign = TextAlign.Center,
                        )
                        message?.let {
                            Text(it, fontSize = 12.sp, color = textSecondary(), textAlign = TextAlign.Center)
                        }
                    }
                    CheckoutStatus.Timeout -> {
                        Text(
                            L.text(context, localePrefs, com.lextures.android.R.string.mobile_billing_paymentProcessing),
                            fontWeight = FontWeight.SemiBold,
                            textAlign = TextAlign.Center,
                        )
                        Text(
                            L.text(context, localePrefs, com.lextures.android.R.string.mobile_billing_paymentProcessingHint),
                            fontSize = 12.sp,
                            color = textSecondary(),
                            textAlign = TextAlign.Center,
                        )
                    }
                    CheckoutStatus.Cancelled -> {
                        Text(
                            L.text(context, localePrefs, com.lextures.android.R.string.mobile_billing_checkoutCancelled),
                            fontWeight = FontWeight.SemiBold,
                            textAlign = TextAlign.Center,
                        )
                    }
                }
                if (status != CheckoutStatus.Verifying) {
                    Button(onClick = onDismiss) {
                        Text(L.text(context, localePrefs, com.lextures.android.R.string.mobile_common_close))
                    }
                }
            }
        }
    }
}