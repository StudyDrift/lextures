package com.lextures.android.features.introcourse

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.AutoAwesome
import androidx.compose.material.icons.filled.Book
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.Circle
import androidx.compose.material.icons.filled.RadioButtonChecked
import androidx.compose.material3.Button
import androidx.compose.material3.Icon
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalLocalePreferences
import com.lextures.android.core.lms.IntroCourseCardState
import com.lextures.android.core.lms.IntroCourseLogic
import com.lextures.android.core.lms.IntroCourseModuleProgress
import com.lextures.android.core.lms.IntroCourseObservability
import com.lextures.android.core.lms.IntroCourseProgress
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.offline.OfflineCacheKey
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.core.routing.DeepLinkRouter
import com.lextures.android.features.home.HomeShellState
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsSkeletonList
import kotlinx.serialization.serializer

@Composable
fun IntroCourseEntryCard(
    session: AuthSession,
    shell: HomeShellState,
    modifier: Modifier = Modifier,
) {
    if (!IntroCourseLogic.introCourseEnabled(shell.platformFeatures)) return

    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    val accessToken by session.accessToken.collectAsState()
    val offline = remember { OfflineService.get(context) }
    var progress by remember { mutableStateOf<IntroCourseProgress?>(null) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf(false) }
    var recordedView by remember { mutableStateOf(false) }

    val state = IntroCourseLogic.cardState(progress, loading, error)

    LaunchedEffect(accessToken) {
        val token = accessToken ?: return@LaunchedEffect
        loading = true
        error = false
        try {
            val (value, _) = offline.cachedFetch(
                key = OfflineCacheKey.introCourseProgress(),
                accessToken = token,
                serializer = serializer<IntroCourseProgress>(),
            ) { LmsApi.fetchIntroCourseProgress(token) }
            progress = value
        } catch (_: Exception) {
            error = true
        } finally {
            loading = false
        }
    }

    LaunchedEffect(state) {
        if (!recordedView && state != IntroCourseCardState.Hidden && state != IntroCourseCardState.Loading) {
            recordedView = true
            IntroCourseObservability.recordCardView(context)
        }
    }

    when (state) {
        IntroCourseCardState.Hidden -> Unit
        IntroCourseCardState.Loading -> LmsSkeletonList(count = 1, modifier = modifier)
        IntroCourseCardState.Error -> IntroCourseFallbackCard(shell, modifier)
        IntroCourseCardState.NotStarted, IntroCourseCardState.InProgress -> progress?.let {
            IntroCourseActiveCard(
                progress = it,
                notStarted = state == IntroCourseCardState.NotStarted,
                onCta = {
                    IntroCourseObservability.recordCtaClick(context)
                    shell.openDeepLink(DeepLinkRouter.resolve(IntroCourseLogic.ctaRoute(it)))
                },
                modifier = modifier,
            )
        }
        IntroCourseCardState.Completed -> progress?.let {
            IntroCourseCompletedCard(
                onRevisit = {
                    IntroCourseObservability.recordCtaClick(context)
                    shell.openDeepLink(
                        DeepLinkRouter.resolve(
                            IntroCourseLogic.fallbackRoute(
                                it.courseCode ?: com.lextures.android.core.lms.IntroCourseConstants.courseCode,
                            ),
                        ),
                    )
                },
                modifier = modifier,
            )
        }
    }
}

@Composable
fun IntroCourseProgressRail(
    courseCode: String,
    session: AuthSession,
    shell: HomeShellState,
    modifier: Modifier = Modifier,
) {
    if (!IntroCourseLogic.introCourseEnabled(shell.platformFeatures) || !IntroCourseLogic.isIntroCourse(courseCode)) {
        return
    }

    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    val accessToken by session.accessToken.collectAsState()
    val offline = remember { OfflineService.get(context) }
    var progress by remember { mutableStateOf<IntroCourseProgress?>(null) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf(false) }

    LaunchedEffect(accessToken) {
        val token = accessToken ?: return@LaunchedEffect
        loading = true
        error = false
        try {
            val (value, _) = offline.cachedFetch(
                key = OfflineCacheKey.introCourseProgress(),
                accessToken = token,
                serializer = serializer<IntroCourseProgress>(),
            ) { LmsApi.fetchIntroCourseProgress(token) }
            progress = value
        } catch (_: Exception) {
            error = true
        } finally {
            loading = false
        }
    }

    val current = progress
    if (loading || error || current?.enrolled != true) return

    LmsCard(
        modifier = modifier.semantics {
            contentDescription = L.text(context, localePrefs, R.string.mobile_introCourse_rail_ariaLabel)
        },
    ) {
        Text(
            L.text(context, localePrefs, R.string.mobile_introCourse_rail_title),
            style = MaterialTheme.typography.titleSmall,
            fontWeight = FontWeight.SemiBold,
            color = textPrimary(),
        )
        IntroCourseProgressBar(
            percent = current.percent,
            modulesComplete = current.modulesComplete,
            modulesTotal = current.modulesTotal,
            modifier = Modifier.padding(top = 8.dp),
        )
        current.nextItem?.title?.takeIf { current.completedAt == null }?.let { title ->
            Text(
                L.format(context, localePrefs, R.string.mobile_introCourse_rail_nextUp, title),
                style = MaterialTheme.typography.bodySmall,
                color = textSecondary(),
                modifier = Modifier.padding(top = 8.dp),
            )
        }
        current.modules?.takeIf { it.isNotEmpty() }?.forEach { module ->
            IntroCourseModuleRow(module)
        }
    }
}

@Composable
private fun IntroCourseModuleRow(module: IntroCourseModuleProgress) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(top = 6.dp),
        horizontalArrangement = Arrangement.spacedBy(8.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(
            imageVector = when (module.status) {
                "done" -> Icons.Default.CheckCircle
                "current" -> Icons.Default.RadioButtonChecked
                else -> Icons.Default.Circle
            },
            contentDescription = null,
        )
        Text(
            module.title,
            style = MaterialTheme.typography.bodySmall,
            fontWeight = if (module.status == "current") FontWeight.SemiBold else FontWeight.Normal,
            color = textPrimary(),
        )
    }
}

@Composable
fun IntroCourseProgressBar(
    percent: Int,
    modulesComplete: Int,
    modulesTotal: Int,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    Column(modifier = modifier) {
        Text(
            L.format(
                context,
                localePrefs,
                R.string.mobile_introCourse_progress_modules,
                maxOf(modulesComplete, 1).toString(),
                modulesTotal.toString(),
            ),
            style = MaterialTheme.typography.labelSmall,
            color = textSecondary(),
        )
        LinearProgressIndicator(
            progress = { (percent.coerceIn(0, 100) / 100f) },
            modifier = Modifier
                .fillMaxWidth()
                .padding(top = 6.dp)
                .height(8.dp)
                .clip(RoundedCornerShape(999.dp)),
        )
    }
}

@Composable
private fun IntroCourseActiveCard(
    progress: IntroCourseProgress,
    notStarted: Boolean,
    onCta: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    LmsCard(
        modifier = modifier.semantics {
            contentDescription = L.text(context, localePrefs, R.string.mobile_introCourse_card_ariaLabel)
        },
    ) {
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp), verticalAlignment = Alignment.CenterVertically) {
            Icon(Icons.Default.AutoAwesome, contentDescription = null)
            Text(
                L.text(
                    context,
                    localePrefs,
                    if (notStarted) R.string.mobile_introCourse_card_startHere else R.string.mobile_introCourse_card_continueOnboarding,
                ),
                style = MaterialTheme.typography.labelMedium,
                fontWeight = FontWeight.SemiBold,
            )
        }
        Text(
            L.text(context, localePrefs, R.string.mobile_introCourse_card_title),
            style = MaterialTheme.typography.titleMedium,
            fontWeight = FontWeight.Bold,
            modifier = Modifier.padding(top = 8.dp),
        )
        Text(
            if (!notStarted && progress.nextItem?.title != null) {
                L.format(context, localePrefs, R.string.mobile_introCourse_card_nextUp, progress.nextItem!!.title)
            } else {
                L.text(context, localePrefs, R.string.mobile_introCourse_card_subtitle)
            },
            style = MaterialTheme.typography.bodySmall,
            color = textSecondary(),
            modifier = Modifier.padding(top = 4.dp),
        )
        IntroCourseProgressBar(
            percent = progress.percent,
            modulesComplete = progress.modulesComplete,
            modulesTotal = progress.modulesTotal,
            modifier = Modifier.padding(top = 8.dp),
        )
        Button(onClick = onCta, modifier = Modifier.padding(top = 12.dp)) {
            Text(
                L.text(
                    context,
                    localePrefs,
                    if (notStarted) R.string.mobile_introCourse_card_ctaStart else R.string.mobile_introCourse_card_ctaContinue,
                ),
            )
        }
    }
}

@Composable
private fun IntroCourseCompletedCard(
    onRevisit: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    LmsCard(modifier = modifier) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp), verticalAlignment = Alignment.CenterVertically) {
                Icon(Icons.Default.CheckCircle, contentDescription = null)
                Text(L.text(context, localePrefs, R.string.mobile_introCourse_card_completedLabel))
            }
            TextButton(onClick = onRevisit) {
                Text(L.text(context, localePrefs, R.string.mobile_introCourse_card_revisit))
            }
        }
    }
}

@Composable
private fun IntroCourseFallbackCard(shell: HomeShellState, modifier: Modifier = Modifier) {
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    LmsCard(modifier = modifier) {
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp), verticalAlignment = Alignment.CenterVertically) {
            Icon(Icons.Default.Book, contentDescription = null)
            Text(L.text(context, localePrefs, R.string.mobile_introCourse_card_fallbackLabel))
        }
        TextButton(
            onClick = {
                IntroCourseObservability.recordCtaClick(context)
                shell.openDeepLink(DeepLinkRouter.resolve(IntroCourseLogic.fallbackRoute()))
            },
        ) {
            Text(L.text(context, localePrefs, R.string.mobile_introCourse_card_fallbackLink))
        }
    }
}