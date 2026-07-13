package com.lextures.android.features.settings.admin

import android.content.Intent
import android.net.Uri
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
import androidx.compose.material.icons.filled.Apartment
import androidx.compose.material.icons.filled.Lock
import androidx.compose.material.icons.filled.OpenInBrowser
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Tab
import androidx.compose.material3.TabRow
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
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
import com.lextures.android.core.config.AppConfiguration
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalePreferences
import com.lextures.android.core.lms.AdminOrgRow
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.OrgStructureAdminLogic
import com.lextures.android.features.home.HomeShellState
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState

private enum class OrgStructureTab {
    Organizations,
    OrgUnits,
    Terms,
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun OrgStructureAdminScreen(
    session: AuthSession,
    shell: HomeShellState,
    localePrefs: LocalePreferences,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val accessToken by session.accessToken.collectAsState()

    var organizations by remember { mutableStateOf<List<AdminOrgRow>>(emptyList()) }
    var selectedOrgId by remember { mutableStateOf("") }
    var selectedTabIndex by remember { mutableIntStateOf(0) }

    val canView = OrgStructureAdminLogic.canView(shell.platformFeatures, shell.permissions)
    val canManageOrgs = OrgStructureAdminLogic.canManageOrganizations(shell.permissions)
    val canManageUnitsAndTerms = OrgStructureAdminLogic.canManageOrgUnitsAndTerms(shell.permissions)

    val tabs = buildList {
        if (canManageOrgs) add(OrgStructureTab.Organizations)
        if (canManageUnitsAndTerms) {
            add(OrgStructureTab.OrgUnits)
            add(OrgStructureTab.Terms)
        }
    }

    LaunchedEffect(accessToken, canManageOrgs) {
        val token = accessToken ?: return@LaunchedEffect
        if (selectedOrgId.isEmpty()) {
            selectedOrgId = OrgStructureAdminLogic.resolveOrgId(token, emptyList()).orEmpty()
        }
        if (canManageOrgs) {
            organizations = runCatching { LmsApi.fetchAdminOrganizations(token) }.getOrDefault(emptyList())
            if (selectedOrgId.isEmpty()) {
                selectedOrgId = organizations.firstOrNull()?.id.orEmpty()
            }
        }
    }

    LaunchedEffect(tabs) {
        if (selectedTabIndex >= tabs.size) {
            selectedTabIndex = 0
        }
    }

    if (!canView) {
        Scaffold(
            modifier = modifier.fillMaxSize(),
            topBar = {
                TopAppBar(
                    title = { Text(L.text(context, localePrefs, R.string.mobile_admin_orgStructure_title)) },
                    navigationIcon = {
                        IconButton(onClick = onBack) {
                            Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = null)
                        }
                    },
                )
            },
        ) { padding ->
            LmsEmptyState(
                icon = Icons.Default.Lock,
                title = L.text(context, localePrefs, R.string.mobile_admin_orgStructure_accessDeniedTitle),
                message = L.text(context, localePrefs, R.string.mobile_admin_orgStructure_accessDeniedMessage),
                modifier = Modifier.padding(padding).padding(16.dp),
            )
        }
        return
    }

    Scaffold(
        modifier = modifier.fillMaxSize(),
        topBar = {
            TopAppBar(
                title = { Text(L.text(context, localePrefs, R.string.mobile_admin_orgStructure_title)) },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = null)
                    }
                },
            )
        },
    ) { padding ->
        Column(modifier = Modifier.padding(padding).fillMaxSize()) {
            if (tabs.size > 1) {
                TabRow(selectedTabIndex = selectedTabIndex) {
                    tabs.forEachIndexed { index, tab ->
                        Tab(
                            selected = selectedTabIndex == index,
                            onClick = { selectedTabIndex = index },
                            text = {
                                Text(
                                    when (tab) {
                                        OrgStructureTab.Organizations ->
                                            L.text(context, localePrefs, R.string.mobile_admin_orgStructure_tab_organizations)
                                        OrgStructureTab.OrgUnits ->
                                            L.text(context, localePrefs, R.string.mobile_admin_orgStructure_tab_orgUnits)
                                        OrgStructureTab.Terms ->
                                            L.text(context, localePrefs, R.string.mobile_admin_orgStructure_tab_terms)
                                    },
                                )
                            },
                        )
                    }
                }
            }

            when (tabs.getOrNull(selectedTabIndex)) {
                OrgStructureTab.Organizations -> OrganizationsAdminPanel(
                    organizations = organizations,
                    localePrefs = localePrefs,
                    modifier = Modifier.fillMaxSize(),
                )
                OrgStructureTab.OrgUnits -> OrgStructureScreen(
                    session = session,
                    localePrefs = localePrefs,
                    orgId = selectedOrgId,
                    organizations = organizations,
                    canPickOrg = canManageOrgs,
                    selectedOrgId = selectedOrgId,
                    onOrgSelected = { selectedOrgId = it },
                    modifier = Modifier.fillMaxSize(),
                )
                OrgStructureTab.Terms -> TermsAdminScreen(
                    session = session,
                    localePrefs = localePrefs,
                    orgId = selectedOrgId,
                    organizations = organizations,
                    canPickOrg = canManageOrgs,
                    selectedOrgId = selectedOrgId,
                    onOrgSelected = { selectedOrgId = it },
                    modifier = Modifier.fillMaxSize(),
                )
                null -> Unit
            }
        }
    }
}

@Composable
private fun OrganizationsAdminPanel(
    organizations: List<AdminOrgRow>,
    localePrefs: LocalePreferences,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    Column(
        modifier = modifier
            .verticalScroll(rememberScrollState())
            .padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        Text(
            text = L.text(context, localePrefs, R.string.mobile_admin_orgStructure_description),
            color = textSecondary(),
        )
        WebLinkCard(
            localePrefs = localePrefs,
            hintRes = R.string.mobile_admin_orgStructure_webHint,
            path = OrgStructureAdminLogic.webOrganizationsPath(),
        )
        if (organizations.isEmpty()) {
            LmsEmptyState(
                icon = Icons.Filled.Apartment,
                title = L.text(context, localePrefs, R.string.mobile_admin_orgStructure_organizations_emptyTitle),
                message = L.text(context, localePrefs, R.string.mobile_admin_orgStructure_organizations_emptyMessage),
            )
        } else {
            organizations.forEach { org ->
                LmsCard {
                    Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
                        Text(org.name, fontWeight = FontWeight.SemiBold, color = textPrimary())
                        Text(org.slug, color = textSecondary(), fontSize = 12.sp)
                        Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                            Text(org.status.replaceFirstChar { it.uppercase() }, color = textSecondary())
                            org.userCount?.let {
                                Text(
                                    L.format(context, localePrefs, R.string.mobile_admin_orgStructure_organizations_users, it),
                                    color = textSecondary(),
                                )
                            }
                            org.courseCount?.let {
                                Text(
                                    L.format(context, localePrefs, R.string.mobile_admin_orgStructure_organizations_courses, it),
                                    color = textSecondary(),
                                )
                            }
                        }
                    }
                }
            }
        }
    }
}

@Composable
internal fun WebLinkCard(
    localePrefs: LocalePreferences,
    hintRes: Int,
    path: String,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    LmsCard(modifier = modifier.clickable {
        val intent = Intent(Intent.ACTION_VIEW, Uri.parse(AppConfiguration.webUrl(path)))
        context.startActivity(intent)
    }) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            Icon(Icons.Default.OpenInBrowser, contentDescription = null)
            Column(modifier = Modifier.weight(1f)) {
                Text(
                    L.text(context, localePrefs, R.string.mobile_admin_orgStructure_webTitle),
                    fontWeight = FontWeight.SemiBold,
                    color = textPrimary(),
                )
                Text(
                    L.text(context, localePrefs, hintRes),
                    color = textSecondary(),
                )
            }
        }
    }
}
