package com.lextures.android.features.advising

import android.content.Intent
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Person
import androidx.compose.material.icons.filled.Warning
import androidx.compose.material3.Button
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.core.net.toUri
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.StalenessChip
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalePreferences
import com.lextures.android.core.lms.AdvisingLogic
import com.lextures.android.core.lms.AdvisingNote
import com.lextures.android.core.lms.AdvisingRequirementGroup
import com.lextures.android.core.lms.DegreeProgress
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.MyAdvisingConfig
import com.lextures.android.core.offline.OfflineCacheKey
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsSkeletonList
import kotlinx.serialization.builtins.ListSerializer

@Composable
fun AdvisingScreen(
    session: AuthSession,
    localePrefs: LocalePreferences,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val offline = remember { OfflineService.get(context) }
    val isOnline by offline.networkMonitor.isOnline.collectAsState()
    val accessToken = session.accessToken.value

    var notes by remember { mutableStateOf<List<AdvisingNote>>(emptyList()) }
    var progress by remember { mutableStateOf<DegreeProgress?>(null) }
    var config by remember { mutableStateOf<MyAdvisingConfig?>(null) }
    var loading by remember { mutableStateOf(true) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var cacheLabel by remember { mutableStateOf<String?>(null) }

    val sortedNotes = remember(notes) { AdvisingLogic.sortedNotes(notes) }
    val advisor = remember(notes) { AdvisingLogic.advisorFromNotes(notes) }
    val appointmentUrl = remember(progress, config) {
        AdvisingLogic.appointmentUrl(progress, config)
    }
    val advisorFallback = L.text(context, localePrefs, R.string.mobile_advising_advisorFallback)

    LaunchedEffect(accessToken) {
        val token = accessToken ?: return@LaunchedEffect
        loading = notes.isEmpty() && progress == null
        errorMessage = null
        try {
            val notesResult = offline.cachedFetch(
                key = OfflineCacheKey.advisingNotes(),
                accessToken = token,
                serializer = ListSerializer(AdvisingNote.serializer()),
            ) {
                LmsApi.fetchAdvisingNotes(token)
            }
            notes = notesResult.first
            val notesCached = notesResult.second
            cacheLabel = if (notesCached != null && notesCached.isStale(isOnline)) {
                notesCached.lastUpdatedLabel()
            } else {
                null
            }

            val progressResult = offline.cachedFetch(
                key = OfflineCacheKey.degreeProgress(),
                accessToken = token,
                serializer = DegreeProgress.serializer(),
            ) {
                LmsApi.fetchDegreeProgress(token)
            }
            progress = progressResult.first

            if (isOnline) {
                config = runCatching { LmsApi.fetchMyAdvisingConfig(token) }.getOrNull()
            }
        } catch (_: Exception) {
            if (notes.isEmpty() && progress == null) {
                errorMessage = L.text(context, localePrefs, R.string.mobile_advising_loadError)
            }
        } finally {
            loading = false
        }
    }

    Column(
        modifier = modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(16.dp),
    ) {
        when {
            loading -> LmsSkeletonList(count = 3)
            errorMessage != null && sortedNotes.isEmpty() && progress == null -> LmsEmptyState(
                icon = Icons.Default.Person,
                title = L.text(context, localePrefs, R.string.mobile_advising_errorTitle),
                message = errorMessage!!,
            )
            else -> {
                cacheLabel?.let { StalenessChip(label = it) }

                Text(
                    L.text(context, localePrefs, R.string.mobile_advising_subtitle),
                    fontSize = 14.sp,
                    color = textSecondary(),
                    modifier = Modifier.padding(bottom = 12.dp),
                )

                if (progress?.atRisk == true) {
                    AtRiskBanner(localePrefs = localePrefs, modifier = Modifier.padding(bottom = 12.dp))
                }

                advisor?.let { info ->
                    AdvisorCard(
                        displayName = info.displayName,
                        email = info.email,
                        localePrefs = localePrefs,
                        modifier = Modifier.padding(bottom = 12.dp),
                    )
                }

                progress?.let { degreeProgress ->
                    DegreeProgressCard(
                        progress = degreeProgress,
                        localePrefs = localePrefs,
                        modifier = Modifier.padding(bottom = 12.dp),
                    )
                }

                Text(
                    L.text(context, localePrefs, R.string.mobile_advising_notesTitle),
                    fontWeight = FontWeight.SemiBold,
                    fontSize = 12.sp,
                    color = textSecondary(),
                    modifier = Modifier.padding(bottom = 8.dp),
                )

                if (sortedNotes.isEmpty()) {
                    LmsEmptyState(
                        icon = Icons.Default.Person,
                        title = L.text(context, localePrefs, R.string.mobile_advising_emptyTitle),
                        message = L.text(context, localePrefs, R.string.mobile_advising_emptyMessage),
                    )
                } else {
                    sortedNotes.forEach { note ->
                        NoteRow(
                            note = note,
                            advisorFallback = advisorFallback,
                            localePrefs = localePrefs,
                            modifier = Modifier.padding(bottom = 12.dp),
                        )
                    }
                }

                appointmentUrl?.let { url ->
                    val canBook = AdvisingLogic.canBookAppointment(isOnline, url)
                    Button(
                        onClick = {
                            context.startActivity(Intent(Intent.ACTION_VIEW, url.toUri()))
                        },
                        enabled = canBook,
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(top = 8.dp),
                    ) {
                        Text(L.text(context, localePrefs, R.string.mobile_advising_bookAppointment))
                    }
                    if (!isOnline) {
                        Text(
                            L.text(context, localePrefs, R.string.mobile_advising_bookOffline),
                            fontSize = 12.sp,
                            color = textSecondary(),
                            modifier = Modifier.padding(top = 4.dp),
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun AtRiskBanner(
    localePrefs: LocalePreferences,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    LmsCard(modifier = modifier.fillMaxWidth()) {
        Row(verticalAlignment = Alignment.Top) {
            Icon(
                Icons.Default.Warning,
                contentDescription = null,
                tint = textSecondary(),
            )
            Column(modifier = Modifier.padding(start = 10.dp)) {
                Text(
                    L.text(context, localePrefs, R.string.mobile_advising_atRiskTitle),
                    fontWeight = FontWeight.SemiBold,
                    color = textPrimary(),
                )
                Text(
                    L.text(context, localePrefs, R.string.mobile_advising_atRiskMessage),
                    fontSize = 12.sp,
                    color = textSecondary(),
                )
            }
        }
    }
}

@Composable
private fun AdvisorCard(
    displayName: String,
    email: String?,
    localePrefs: LocalePreferences,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    LmsCard(modifier = modifier.fillMaxWidth()) {
        Text(
            L.text(context, localePrefs, R.string.mobile_advising_advisorTitle),
            fontSize = 12.sp,
            color = textSecondary(),
        )
        Text(displayName, fontWeight = FontWeight.SemiBold, color = textPrimary())
        if (!email.isNullOrBlank() && email != displayName) {
            Text(email, fontSize = 12.sp, color = textSecondary())
        }
    }
}

@Composable
private fun DegreeProgressCard(
    progress: DegreeProgress,
    localePrefs: LocalePreferences,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    LmsCard(modifier = modifier.fillMaxWidth()) {
        Text(
            L.text(context, localePrefs, R.string.mobile_advising_degreeProgressTitle),
            fontSize = 12.sp,
            color = textSecondary(),
        )
        if (progress.configured && progress.completionPercent != null) {
            Text(
                context.getString(
                    R.string.mobile_advising_completionPercent,
                    progress.completionPercent,
                ),
                fontWeight = FontWeight.SemiBold,
                color = textPrimary(),
            )
            Text(
                context.getString(
                    R.string.mobile_advising_remainingCourses,
                    progress.remainingRequiredCount ?: 0,
                ),
                fontSize = 14.sp,
                color = textSecondary(),
            )
            progress.remainingRequirements.orEmpty().take(3).forEach { req ->
                Text(requirementLabel(req), fontSize = 12.sp, color = textSecondary())
            }
            progress.lastUpdated?.let { updated ->
                val formatted = AdvisingLogic.formatAuditDate(updated)
                val text = if (progress.stale == true) {
                    context.getString(R.string.mobile_advising_staleAudit, formatted)
                } else {
                    context.getString(R.string.mobile_advising_updatedAudit, formatted)
                }
                Text(text, fontSize = 11.sp, color = textSecondary())
            }
        } else {
            Text(
                L.text(context, localePrefs, R.string.mobile_advising_noAudit),
                fontSize = 14.sp,
                color = textSecondary(),
            )
        }
    }
}

@Composable
private fun NoteRow(
    note: AdvisingNote,
    advisorFallback: String,
    localePrefs: LocalePreferences,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val advisorName = AdvisingLogic.advisorLabel(note.advisorDisplayName, note.advisorEmail)
        .let { if (it == "Your advisor") advisorFallback else it }
    val formattedDate = AdvisingLogic.formatNoteDate(note.createdAt)

    LmsCard(modifier = modifier.fillMaxWidth()) {
        Text(
            "$formattedDate · $advisorName",
            fontSize = 12.sp,
            color = textSecondary(),
        )
        Text(
            note.content,
            fontSize = 14.sp,
            color = textPrimary(),
            modifier = Modifier.padding(top = 6.dp),
        )
    }
}

private fun requirementLabel(req: AdvisingRequirementGroup): String =
    if (req.coursesRemaining > 0) "${req.group} (${req.coursesRemaining} left)" else req.group
