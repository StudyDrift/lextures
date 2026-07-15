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
import androidx.compose.material.icons.filled.BarChart
import androidx.compose.material.icons.filled.Lock
import androidx.compose.material.icons.filled.Memory
import androidx.compose.material.icons.automirrored.filled.Notes
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
import com.lextures.android.core.lms.AiModelsAdminLogic
import com.lextures.android.features.home.HomeShellState
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun AiAdminHubScreen(
    session: AuthSession,
    shell: HomeShellState,
    localePrefs: LocalePreferences,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val canView = AiModelsAdminLogic.canView(shell.platformFeatures, shell.permissions)
    var showModels by remember { mutableStateOf(false) }
    var showPrompts by remember { mutableStateOf(false) }
    var showReports by remember { mutableStateOf(false) }

    if (showModels) {
        AiModelsSettingsScreen(
            session = session,
            shell = shell,
            localePrefs = localePrefs,
            onBack = { showModels = false },
        )
        return
    }
    if (showPrompts) {
        SystemPromptsScreen(
            session = session,
            shell = shell,
            localePrefs = localePrefs,
            onBack = { showPrompts = false },
        )
        return
    }
    if (showReports) {
        AiReportsScreen(
            session = session,
            shell = shell,
            localePrefs = localePrefs,
            onBack = { showReports = false },
        )
        return
    }

    if (!canView) {
        Scaffold(
            modifier = modifier.fillMaxSize(),
            topBar = {
                TopAppBar(
                    title = { Text(L.text(context, localePrefs, R.string.mobile_admin_ai_hub_title)) },
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
                title = L.text(context, localePrefs, R.string.mobile_admin_ai_accessDenied_title),
                message = L.text(context, localePrefs, R.string.mobile_admin_ai_accessDenied_message),
            )
        }
        return
    }

    Scaffold(
        modifier = modifier.fillMaxSize(),
        topBar = {
            TopAppBar(
                title = { Text(L.text(context, localePrefs, R.string.mobile_admin_ai_hub_title)) },
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
                .verticalScroll(rememberScrollState())
                .padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp),
        ) {
            Text(
                text = L.text(context, localePrefs, R.string.mobile_admin_ai_hub_description),
                fontSize = 14.sp,
                color = textSecondary(),
            )
            LmsCard {
                Column {
                    HubNavRow(
                        icon = Icons.Default.Memory,
                        title = L.text(context, localePrefs, R.string.mobile_admin_ai_models_title),
                        subtitle = L.text(context, localePrefs, R.string.mobile_admin_ai_models_entry_subtitle),
                        onClick = { showModels = true },
                    )
                    HubNavRow(
                        icon = Icons.AutoMirrored.Filled.Notes,
                        title = L.text(context, localePrefs, R.string.mobile_admin_ai_prompts_title),
                        subtitle = L.text(context, localePrefs, R.string.mobile_admin_ai_prompts_entry_subtitle),
                        onClick = { showPrompts = true },
                    )
                    HubNavRow(
                        icon = Icons.Default.BarChart,
                        title = L.text(context, localePrefs, R.string.mobile_admin_ai_reports_title),
                        subtitle = L.text(context, localePrefs, R.string.mobile_admin_ai_reports_entry_subtitle),
                        onClick = { showReports = true },
                    )
                }
            }
        }
    }
}

@Composable
private fun HubNavRow(
    icon: ImageVector,
    title: String,
    subtitle: String,
    onClick: () -> Unit,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onClick)
            .padding(vertical = 12.dp),
        horizontalArrangement = Arrangement.spacedBy(12.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(icon, contentDescription = null, tint = textPrimary())
        Column(modifier = Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
            Text(text = title, fontSize = 14.sp, fontWeight = FontWeight.SemiBold, color = textPrimary())
            Text(text = subtitle, fontSize = 12.sp, color = textSecondary())
        }
        Icon(
            Icons.AutoMirrored.Filled.KeyboardArrowRight,
            contentDescription = null,
            tint = textSecondary().copy(alpha = 0.6f),
        )
    }
}
