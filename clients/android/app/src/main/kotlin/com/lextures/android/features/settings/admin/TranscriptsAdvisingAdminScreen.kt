package com.lextures.android.features.settings.admin

import androidx.compose.foundation.clickable
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
import androidx.compose.material.icons.automirrored.filled.KeyboardArrowRight
import androidx.compose.material.icons.filled.Description
import androidx.compose.material.icons.filled.Groups
import androidx.compose.material.icons.filled.Lock
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalePreferences
import com.lextures.android.core.lms.TranscriptsAdvisingAdminLogic
import com.lextures.android.features.home.HomeShellState
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun TranscriptsAdvisingAdminScreen(
    session: AuthSession,
    shell: HomeShellState,
    localePrefs: LocalePreferences,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val canView = TranscriptsAdvisingAdminLogic.canView(shell.platformFeatures, shell.permissions)
    val sections = TranscriptsAdvisingAdminLogic.visibleSections(shell.platformFeatures)
    var showTranscripts by remember { mutableStateOf(false) }
    var showAdvising by remember { mutableStateOf(false) }

    if (showTranscripts) {
        TranscriptsSettingsScreen(
            session = session,
            shell = shell,
            localePrefs = localePrefs,
            onBack = { showTranscripts = false },
        )
        return
    }
    if (showAdvising) {
        AdvisingSettingsScreen(
            session = session,
            shell = shell,
            localePrefs = localePrefs,
            onBack = { showAdvising = false },
        )
        return
    }

    if (!canView) {
        Scaffold(
            modifier = modifier.fillMaxSize(),
            topBar = {
                TopAppBar(
                    title = {
                        Text(L.text(context, localePrefs, R.string.mobile_admin_transcriptsAdvising_hub_title))
                    },
                    navigationIcon = {
                        IconButton(onClick = onBack) {
                            Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = null)
                        }
                    },
                )
            },
        ) { padding ->
            LmsEmptyState(
                modifier = Modifier.padding(padding).padding(16.dp),
                icon = Icons.Default.Lock,
                title = L.text(context, localePrefs, R.string.mobile_admin_transcriptsAdvising_accessDenied_title),
                message = L.text(context, localePrefs, R.string.mobile_admin_transcriptsAdvising_accessDenied_message),
            )
        }
        return
    }

    Scaffold(
        modifier = modifier.fillMaxSize(),
        topBar = {
            TopAppBar(
                title = {
                    Text(L.text(context, localePrefs, R.string.mobile_admin_transcriptsAdvising_hub_title))
                },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = null)
                    }
                },
            )
        },
    ) { padding ->
        Column(
            modifier = Modifier
                .padding(padding)
                .fillMaxSize()
                .verticalScroll(rememberScrollState())
                .padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp),
        ) {
            Text(
                text = L.text(context, localePrefs, R.string.mobile_admin_transcriptsAdvising_hub_description),
                fontSize = 14.sp,
                color = textSecondary(),
            )
            if (sections.isEmpty()) {
                LmsEmptyState(
                    icon = Icons.Default.Description,
                    title = L.text(context, localePrefs, R.string.mobile_admin_transcriptsAdvising_emptyTitle),
                    message = L.text(context, localePrefs, R.string.mobile_admin_transcriptsAdvising_emptyMessage),
                )
            } else {
                LmsCard {
                    Column {
                        sections.forEachIndexed { index, section ->
                            if (index > 0) {
                                androidx.compose.material3.HorizontalDivider(
                                    modifier = Modifier.padding(start = 44.dp),
                                )
                            }
                            when (section) {
                                TranscriptsAdvisingAdminLogic.Section.TRANSCRIPTS -> {
                                    HubRow(
                                        icon = Icons.Default.Description,
                                        title = L.text(context, localePrefs, R.string.mobile_admin_transcripts_title),
                                        subtitle = L.text(
                                            context,
                                            localePrefs,
                                            R.string.mobile_admin_transcripts_entry_subtitle,
                                        ),
                                        onClick = { showTranscripts = true },
                                    )
                                }
                                TranscriptsAdvisingAdminLogic.Section.ADVISING -> {
                                    HubRow(
                                        icon = Icons.Default.Groups,
                                        title = L.text(context, localePrefs, R.string.mobile_admin_advising_title),
                                        subtitle = L.text(
                                            context,
                                            localePrefs,
                                            R.string.mobile_admin_advising_entry_subtitle,
                                        ),
                                        onClick = { showAdvising = true },
                                    )
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun HubRow(
    icon: ImageVector,
    title: String,
    subtitle: String,
    onClick: () -> Unit,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onClick)
            .padding(horizontal = 12.dp, vertical = 14.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        Icon(icon, contentDescription = null)
        Column(modifier = Modifier.weight(1f)) {
            Text(title, fontWeight = FontWeight.SemiBold, color = textPrimary())
            Text(subtitle, fontSize = 13.sp, color = textSecondary())
        }
        Icon(Icons.AutoMirrored.Filled.KeyboardArrowRight, contentDescription = null)
    }
}
