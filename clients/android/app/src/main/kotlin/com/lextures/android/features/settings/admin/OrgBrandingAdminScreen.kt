package com.lextures.android.features.settings.admin

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Apartment
import androidx.compose.material.icons.filled.Lock
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalePreferences
import com.lextures.android.core.lms.OrgBrandingAdminLogic
import com.lextures.android.features.home.HomeShellState
import com.lextures.android.features.home.LmsEmptyState

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun OrgBrandingAdminScreen(
    session: AuthSession,
    shell: HomeShellState,
    localePrefs: LocalePreferences,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val accessToken by session.accessToken.collectAsState()
    var orgId by remember { mutableStateOf("") }

    val canView = OrgBrandingAdminLogic.canView(shell.platformFeatures, shell.permissions)

    LaunchedEffect(accessToken) {
        val token = accessToken ?: return@LaunchedEffect
        orgId = OrgBrandingAdminLogic.resolveOrgId(token, emptyList()).orEmpty()
    }

    if (!canView) {
        Scaffold(
            modifier = modifier.fillMaxSize(),
            topBar = {
                TopAppBar(
                    title = { Text(L.text(context, localePrefs, R.string.mobile_admin_orgBranding_title)) },
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
                title = L.text(context, localePrefs, R.string.mobile_admin_orgBranding_accessDeniedTitle),
                message = L.text(context, localePrefs, R.string.mobile_admin_orgBranding_accessDeniedMessage),
            )
        }
        return
    }

    Scaffold(
        modifier = modifier.fillMaxSize(),
        topBar = {
            TopAppBar(
                title = { Text(L.text(context, localePrefs, R.string.mobile_admin_orgBranding_title)) },
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
            verticalArrangement = androidx.compose.foundation.layout.Arrangement.spacedBy(16.dp),
        ) {
            Text(
                text = L.text(context, localePrefs, R.string.mobile_admin_orgBranding_description),
                fontSize = 14.sp,
                color = textSecondary(),
            )

            if (orgId.isEmpty()) {
                LmsEmptyState(
                    icon = Icons.Filled.Apartment,
                    title = L.text(context, localePrefs, R.string.mobile_admin_orgBranding_noOrgTitle),
                    message = L.text(context, localePrefs, R.string.mobile_admin_orgBranding_noOrgMessage),
                )
            } else {
                OrgBrandingScreen(
                    session = session,
                    localePrefs = localePrefs,
                    orgId = orgId,
                )
                AiGovernanceScreen(session = session, localePrefs = localePrefs)
                AiProviderSettingsScreen(session = session, localePrefs = localePrefs)
            }
        }
    }
}
