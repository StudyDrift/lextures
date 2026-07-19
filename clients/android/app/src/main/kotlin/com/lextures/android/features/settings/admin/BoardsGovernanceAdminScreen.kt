package com.lextures.android.features.settings.admin

import android.content.Intent
import android.net.Uri
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Lock
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
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
import androidx.compose.ui.unit.dp
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.config.AppConfiguration
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalePreferences
import com.lextures.android.core.lms.BoardAdminOverview
import com.lextures.android.core.lms.BoardAnalyticsApi
import com.lextures.android.core.lms.BoardOrgPolicies
import com.lextures.android.core.lms.BoardsAdvancedLogic
import com.lextures.android.core.lms.BoardsAdvancedObservability
import com.lextures.android.core.lms.BoardsGovernanceAdminLogic
import com.lextures.android.core.lms.PatchBoardOrgPoliciesBody
import com.lextures.android.features.home.HomeShellState
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsErrorBanner
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun BoardsGovernanceAdminScreen(
    session: AuthSession,
    shell: HomeShellState,
    localePrefs: LocalePreferences,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val accessToken by session.accessToken.collectAsState()
    val canView = BoardsGovernanceAdminLogic.canView(shell.platformFeatures, shell.permissions)

    var policies by remember { mutableStateOf<BoardOrgPolicies?>(null) }
    var overview by remember { mutableStateOf<BoardAdminOverview?>(null) }
    var loading by remember { mutableStateOf(true) }
    var saving by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    var saved by remember { mutableStateOf(false) }
    var capDraft by remember { mutableStateOf("") }

    fun load() {
        val token = accessToken ?: return
        scope.launch {
            loading = true
            error = null
            try {
                policies = BoardAnalyticsApi.fetchAdminPolicies(accessToken = token)
                overview = BoardAnalyticsApi.fetchAdminOverview(accessToken = token)
                capDraft = policies?.boardCapPerCourse?.toString().orEmpty()
            } catch (_: Exception) {
                error = L.text(context, localePrefs, R.string.mobile_boards_admin_loadError)
            } finally {
                loading = false
            }
        }
    }

    LaunchedEffect(canView, accessToken) {
        if (canView) {
            BoardsAdvancedObservability.record("board_admin_analytics_viewed", mapOf("scope" to "org"))
            load()
        }
    }

    Scaffold(
        modifier = modifier.fillMaxSize(),
        topBar = {
            TopAppBar(
                title = { Text(L.text(R.string.mobile_boards_admin_title)) },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = null)
                    }
                },
            )
        },
    ) { padding ->
        if (!canView) {
            LmsEmptyState(
                icon = Icons.Default.Lock,
                title = L.text(context, localePrefs, R.string.mobile_boards_admin_accessDeniedTitle),
                message = L.text(context, localePrefs, R.string.mobile_boards_admin_accessDeniedMessage),
                modifier = Modifier.padding(padding).padding(16.dp),
            )
            return@Scaffold
        }

        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .verticalScroll(rememberScrollState())
                .padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Text(L.text(R.string.mobile_boards_admin_subtitle), color = textSecondary())
            error?.let { LmsErrorBanner(message = it) }
            if (saved) Text(L.text(R.string.mobile_boards_admin_saved), color = textSecondary())
            if (loading && (policies == null || overview == null)) {
                CircularProgressIndicator()
            } else {
                overview?.let { ov ->
                    LmsCard {
                        Column(Modifier = Modifier.padding(12.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
                            Text(L.text(R.string.mobile_boards_admin_overviewTitle), color = textPrimary())
                            Text("${L.text(R.string.mobile_boards_admin_boardCount)}: ${ov.boardCount}")
                            Text("${L.text(R.string.mobile_boards_admin_activeBoards)}: ${ov.activeBoardCount}")
                            Text("${L.text(R.string.mobile_boards_admin_coursesWithBoards)}: ${ov.coursesWithBoards}")
                            Text(
                                "${L.text(R.string.mobile_boards_admin_storage)}: ${BoardsAdvancedLogic.formatStorageBytes(ov.storageBytes)}",
                            )
                        }
                    }
                }
                policies?.let { pol ->
                    LmsCard {
                        Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                            Text(L.text(R.string.mobile_boards_admin_policiesTitle), color = textPrimary())
                            Row(verticalAlignment = Alignment.CenterVertically) {
                                Text(L.text(R.string.mobile_boards_admin_externalSharing), modifier = Modifier.weight(1f))
                                Switch(
                                    checked = pol.externalSharing,
                                    enabled = !saving,
                                    onCheckedChange = { value ->
                                        scope.launch {
                                            val token = accessToken ?: return@launch
                                            saving = true
                                            try {
                                                policies = BoardAnalyticsApi.patchAdminPolicies(
                                                    PatchBoardOrgPoliciesBody(externalSharing = value),
                                                    accessToken = token,
                                                )
                                                saved = true
                                                BoardsAdvancedObservability.record(
                                                    "board_admin_lifecycle_action",
                                                    mapOf("action" to "policy_patch"),
                                                )
                                            } catch (_: Exception) {
                                                error = L.text(context, localePrefs, R.string.mobile_boards_admin_saveError)
                                            } finally {
                                                saving = false
                                            }
                                        }
                                    },
                                )
                            }
                            Row(verticalAlignment = Alignment.CenterVertically) {
                                Text(L.text(R.string.mobile_boards_admin_minorFloor), modifier = Modifier.weight(1f))
                                Switch(
                                    checked = pol.minorModerationFloor,
                                    enabled = !saving,
                                    onCheckedChange = { value ->
                                        scope.launch {
                                            val token = accessToken ?: return@launch
                                            saving = true
                                            try {
                                                policies = BoardAnalyticsApi.patchAdminPolicies(
                                                    PatchBoardOrgPoliciesBody(minorModerationFloor = value),
                                                    accessToken = token,
                                                )
                                                saved = true
                                            } catch (_: Exception) {
                                                error = L.text(context, localePrefs, R.string.mobile_boards_admin_saveError)
                                            } finally {
                                                saving = false
                                            }
                                        }
                                    },
                                )
                            }
                            OutlinedTextField(
                                value = capDraft,
                                onValueChange = { capDraft = it },
                                label = { Text(L.text(R.string.mobile_boards_admin_boardCap)) },
                                modifier = Modifier.fillMaxWidth(),
                            )
                            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                                Button(
                                    enabled = !saving,
                                    onClick = {
                                        scope.launch {
                                            val token = accessToken ?: return@launch
                                            saving = true
                                            try {
                                                val cap = BoardsAdvancedLogic.parseBoardCapDraft(capDraft)
                                                policies = BoardAnalyticsApi.patchAdminPolicies(
                                                    if (cap == null) {
                                                        PatchBoardOrgPoliciesBody(clearBoardCap = true)
                                                    } else {
                                                        PatchBoardOrgPoliciesBody(boardCapPerCourse = cap)
                                                    },
                                                    accessToken = token,
                                                )
                                                capDraft = policies?.boardCapPerCourse?.toString().orEmpty()
                                                saved = true
                                            } catch (_: Exception) {
                                                error = L.text(context, localePrefs, R.string.mobile_boards_admin_saveError)
                                            } finally {
                                                saving = false
                                            }
                                        }
                                    },
                                ) { Text(L.text(R.string.mobile_common_save)) }
                                TextButton(
                                    enabled = !saving,
                                    onClick = {
                                        scope.launch {
                                            val token = accessToken ?: return@launch
                                            saving = true
                                            try {
                                                policies = BoardAnalyticsApi.patchAdminPolicies(
                                                    PatchBoardOrgPoliciesBody(clearBoardCap = true),
                                                    accessToken = token,
                                                )
                                                capDraft = ""
                                                saved = true
                                            } catch (_: Exception) {
                                                error = L.text(context, localePrefs, R.string.mobile_boards_admin_saveError)
                                            } finally {
                                                saving = false
                                            }
                                        }
                                    },
                                ) { Text(L.text(R.string.mobile_boards_admin_clearCap)) }
                            }
                        }
                    }
                }
            }
            TextButton(
                onClick = {
                    val intent = Intent(
                        Intent.ACTION_VIEW,
                        Uri.parse(AppConfiguration.webUrl(BoardsGovernanceAdminLogic.webPath())),
                    )
                    context.startActivity(intent)
                },
            ) {
                Text(L.text(R.string.mobile_boards_admin_openOnWeb))
            }
        }
    }
}
