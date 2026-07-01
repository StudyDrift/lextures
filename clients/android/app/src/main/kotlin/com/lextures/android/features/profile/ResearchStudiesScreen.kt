package com.lextures.android.features.profile

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.rememberModalBottomSheetState
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
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.LexturesType
import com.lextures.android.core.design.accentColor
import com.lextures.android.core.design.sceneBackground
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.lms.ConsentDecision
import com.lextures.android.core.lms.ConsentHistoryEntry
import com.lextures.android.core.lms.ConsentStudy
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.ProfileDepthLogic
import com.lextures.android.features.home.LmsCard
import kotlinx.coroutines.launch

/** Research study consent management (M1.5). */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ResearchStudiesScreen(
    session: AuthSession,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    val scope = rememberCoroutineScope()
    val actionErrorText = L.text(R.string.mobile_profileDepth_research_actionError)

    var loading by remember { mutableStateOf(true) }
    var loadFailed by remember { mutableStateOf(false) }
    var pending by remember { mutableStateOf<List<ConsentStudy>>(emptyList()) }
    var history by remember { mutableStateOf<List<ConsentHistoryEntry>>(emptyList()) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var busyStudyId by remember { mutableStateOf<String?>(null) }
    var selectedStudy by remember { mutableStateOf<ConsentStudy?>(null) }

    fun load() {
        val token = accessToken ?: run {
            loading = false
            loadFailed = true
            return
        }
        scope.launch {
            loading = true
            loadFailed = false
            errorMessage = null
            try {
                pending = LmsApi.fetchPendingConsentStudies(token)
                history = LmsApi.fetchConsentHistory(token)
            } catch (_: Exception) {
                loadFailed = true
            } finally {
                loading = false
            }
        }
    }

    fun respond(studyId: String, decision: ConsentDecision) {
        val token = accessToken ?: return
        scope.launch {
            busyStudyId = studyId
            errorMessage = null
            try {
                LmsApi.respondToConsentStudy(studyId, decision, token)
                load()
            } catch (_: Exception) {
                errorMessage = actionErrorText
            } finally {
                busyStudyId = null
            }
        }
    }

    LaunchedEffect(accessToken) { load() }

    selectedStudy?.let { study ->
        ConsentStudySheet(study = study, onDismiss = { selectedStudy = null })
    }

    Column(
        modifier = modifier
            .fillMaxSize()
            .background(sceneBackground()),
    ) {
        TopAppBar(
            title = { Text(L.text(R.string.mobile_profileDepth_research_title)) },
            navigationIcon = {
                IconButton(onClick = onBack) {
                    Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = null)
                }
            },
        )

        when {
            loading -> Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                CircularProgressIndicator(color = accentColor())
            }

            loadFailed -> Column(
                modifier = Modifier.fillMaxSize().padding(32.dp),
                horizontalAlignment = Alignment.CenterHorizontally,
                verticalArrangement = Arrangement.Center,
            ) {
                Text(L.text(R.string.mobile_profileDepth_research_loadError), color = textSecondary())
                TextButton(onClick = { load() }) { Text(L.text(R.string.mobile_common_retry)) }
            }

            else -> Column(
                modifier = Modifier
                    .fillMaxSize()
                    .verticalScroll(rememberScrollState())
                    .padding(16.dp),
                verticalArrangement = Arrangement.spacedBy(16.dp),
            ) {
                Text(
                    text = L.text(R.string.mobile_profileDepth_research_description),
                    fontSize = 12.sp,
                    color = textSecondary(),
                )

                errorMessage?.let {
                    Text(text = it, color = LexturesColors.Error, fontSize = 13.sp)
                }

                if (pending.isNotEmpty()) {
                    LmsCard {
                        Text(
                            text = L.text(R.string.mobile_profileDepth_research_awaiting),
                            style = LexturesType.display(17),
                            color = textPrimary(),
                        )
                        pending.forEachIndexed { index, study ->
                            if (index > 0) HorizontalDivider(modifier = Modifier.padding(vertical = 8.dp))
                            PendingStudyRow(
                                study = study,
                                busy = busyStudyId == study.id,
                                onViewConsent = { selectedStudy = study },
                                onDecline = { respond(study.id, ConsentDecision.Declined) },
                                onConsent = { respond(study.id, ConsentDecision.Granted) },
                            )
                        }
                    }
                }

                LmsCard {
                    Text(
                        text = L.text(R.string.mobile_profileDepth_research_decisions),
                        style = LexturesType.display(17),
                        color = textPrimary(),
                    )
                    val decisions = ProfileDepthLogic.latestConsentByStudy(history)
                    if (decisions.isEmpty()) {
                        Text(
                            text = L.text(R.string.mobile_profileDepth_research_empty),
                            fontSize = 12.sp,
                            color = textSecondary(),
                            modifier = Modifier.padding(top = 8.dp),
                        )
                    } else {
                        decisions.forEachIndexed { index, entry ->
                            if (index > 0) HorizontalDivider(modifier = Modifier.padding(vertical = 8.dp))
                            HistoryRow(
                                entry = entry,
                                busy = busyStudyId == entry.studyId,
                                onWithdraw = { respond(entry.studyId, ConsentDecision.Withdrawn) },
                            )
                        }
                    }
                }
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun ConsentStudySheet(
    study: ConsentStudy,
    onDismiss: () -> Unit,
) {
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
    ModalBottomSheet(onDismissRequest = onDismiss, sheetState = sheetState) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .verticalScroll(rememberScrollState())
                .padding(horizontal = 20.dp, vertical = 8.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text(
                    text = study.title,
                    style = LexturesType.display(18, FontWeight.SemiBold),
                    color = textPrimary(),
                    modifier = Modifier.weight(1f),
                )
                TextButton(onClick = onDismiss) {
                    Text(L.text(R.string.mobile_ia_close))
                }
            }
            Text(text = study.consentText, fontSize = 14.sp, color = textPrimary())
            Text(
                text = L.text(R.string.mobile_profileDepth_research_dataUse),
                fontSize = 12.sp,
                fontWeight = FontWeight.SemiBold,
                color = textSecondary(),
            )
            Text(text = study.dataUseDescription, fontSize = 12.sp, color = textSecondary())
        }
    }
}

@Composable
private fun PendingStudyRow(
    study: ConsentStudy,
    busy: Boolean,
    onViewConsent: () -> Unit,
    onDecline: () -> Unit,
    onConsent: () -> Unit,
) {
    Column(
        modifier = Modifier.fillMaxWidth().padding(top = 8.dp),
        verticalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        Text(
            text = study.title,
            fontSize = 14.sp,
            fontWeight = FontWeight.SemiBold,
            color = textPrimary(),
        )
        Text(
            text = L.format(R.string.mobile_profileDepth_research_irb, study.irbProtocol),
            fontSize = 12.sp,
            color = textSecondary(),
        )
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(8.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            TextButton(onClick = onViewConsent, enabled = !busy) {
                Text(
                    text = L.text(R.string.mobile_profileDepth_research_viewConsent),
                    fontSize = 12.sp,
                    fontWeight = FontWeight.SemiBold,
                )
            }
            Box(modifier = Modifier.weight(1f))
            TextButton(onClick = onDecline, enabled = !busy) {
                Text(
                    text = L.text(R.string.mobile_profileDepth_research_decline),
                    fontSize = 12.sp,
                    fontWeight = FontWeight.SemiBold,
                )
            }
            Button(
                onClick = onConsent,
                enabled = !busy,
                shape = RoundedCornerShape(50),
                colors = ButtonDefaults.buttonColors(containerColor = LexturesColors.PrimaryDeep),
                modifier = Modifier.padding(start = 4.dp),
            ) {
                Text(
                    text = L.text(R.string.mobile_profileDepth_research_consent),
                    fontSize = 12.sp,
                    fontWeight = FontWeight.Bold,
                    color = Color.White,
                )
            }
        }
    }
}

@Composable
private fun HistoryRow(
    entry: ConsentHistoryEntry,
    busy: Boolean,
    onWithdraw: () -> Unit,
) {
    Row(
        modifier = Modifier.fillMaxWidth().padding(top = 8.dp),
        horizontalArrangement = Arrangement.spacedBy(12.dp),
        verticalAlignment = Alignment.Top,
    ) {
        Column(modifier = Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(4.dp)) {
            Text(
                text = entry.studyTitle ?: L.text(R.string.mobile_profileDepth_research_study),
                fontSize = 14.sp,
                fontWeight = FontWeight.Medium,
                color = textPrimary(),
            )
            Text(
                text = profileDepthConsentDecisionLabel(entry.decision),
                fontSize = 12.sp,
                color = textSecondary(),
            )
        }
        if (entry.decision == ConsentDecision.Granted) {
            TextButton(onClick = onWithdraw, enabled = !busy) {
                Text(
                    text = L.text(R.string.mobile_profileDepth_research_withdraw),
                    fontSize = 12.sp,
                    fontWeight = FontWeight.SemiBold,
                    color = LexturesColors.Error,
                )
            }
        }
    }
}