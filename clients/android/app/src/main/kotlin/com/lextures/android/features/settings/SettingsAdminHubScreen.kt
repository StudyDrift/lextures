package com.lextures.android.features.settings

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
import androidx.compose.material.icons.filled.Lock
import androidx.compose.material.icons.filled.Search
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
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
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalePreferences
import com.lextures.android.core.lms.AuditLogAdminLogic
import com.lextures.android.core.lms.SettingsMenuLogic
import com.lextures.android.features.home.HomeShellState
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.settings.admin.AiAdminHubScreen
import com.lextures.android.features.settings.admin.ArchivedCoursesAdminScreen
import com.lextures.android.features.settings.admin.AuditLogAdminScreen
import com.lextures.android.features.settings.admin.IntegrationsAdminScreen
import com.lextures.android.features.settings.admin.OrgBrandingAdminScreen
import com.lextures.android.features.settings.admin.OrgStructureAdminScreen
import com.lextures.android.features.settings.admin.PeopleAdminScreen
import com.lextures.android.features.settings.admin.PlatformSettingsScreen
import com.lextures.android.features.settings.admin.RolesPermissionsAdminScreen
import com.lextures.android.features.settings.admin.TranscriptsAdvisingAdminScreen

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SettingsAdminHubScreen(
    session: AuthSession,
    shell: HomeShellState,
    localePrefs: LocalePreferences,
    onBack: () -> Unit,
    initialPage: SettingsMenuLogic.ItemId? = null,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val canView = SettingsMenuLogic.shouldShowHubEntry(shell.platformFeatures, shell.permissions)
    var searchText by remember { mutableStateOf("") }
    var openPage by remember { mutableStateOf<SettingsMenuLogic.ItemId?>(null) }

    LaunchedEffect(initialPage) {
        if (initialPage == SettingsMenuLogic.ItemId.AuditLog &&
            AuditLogAdminLogic.canView(shell.platformFeatures, shell.permissions)
        ) {
            openPage = SettingsMenuLogic.ItemId.AuditLog
        }
    }

    fun stringByName(name: String): String {
        val id = context.resources.getIdentifier(name, "string", context.packageName)
        return if (id == 0) name else L.text(context, localePrefs, id)
    }

    when (openPage) {
        SettingsMenuLogic.ItemId.PlatformSettings -> {
            PlatformSettingsScreen(session, shell, localePrefs, onBack = { openPage = null }, modifier)
            return
        }
        SettingsMenuLogic.ItemId.OrgStructure -> {
            OrgStructureAdminScreen(session, shell, localePrefs, onBack = { openPage = null }, modifier)
            return
        }
        SettingsMenuLogic.ItemId.OrgBranding -> {
            OrgBrandingAdminScreen(session, shell, localePrefs, onBack = { openPage = null }, modifier)
            return
        }
        SettingsMenuLogic.ItemId.RolesPermissions -> {
            RolesPermissionsAdminScreen(session, shell, localePrefs, onBack = { openPage = null }, modifier)
            return
        }
        SettingsMenuLogic.ItemId.People -> {
            PeopleAdminScreen(session, shell, localePrefs, onBack = { openPage = null }, modifier)
            return
        }
        SettingsMenuLogic.ItemId.ArchivedCourses -> {
            ArchivedCoursesAdminScreen(session, shell, localePrefs, onBack = { openPage = null }, modifier)
            return
        }
        SettingsMenuLogic.ItemId.AiAdmin -> {
            AiAdminHubScreen(session, shell, localePrefs, onBack = { openPage = null }, modifier)
            return
        }
        SettingsMenuLogic.ItemId.TranscriptsAdvising -> {
            TranscriptsAdvisingAdminScreen(session, shell, localePrefs, onBack = { openPage = null }, modifier)
            return
        }
        SettingsMenuLogic.ItemId.Integrations -> {
            IntegrationsAdminScreen(session, shell, localePrefs, onBack = { openPage = null }, modifier)
            return
        }
        SettingsMenuLogic.ItemId.AuditLog -> {
            AuditLogAdminScreen(session, shell, localePrefs, onBack = { openPage = null }, modifier)
            return
        }
        null -> Unit
    }

    Scaffold(
        modifier = modifier.fillMaxSize(),
        topBar = {
            TopAppBar(
                title = { Text(L.text(R.string.mobile_settings_menu_title)) },
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
                title = L.text(context, localePrefs, R.string.mobile_settings_menu_accessDenied_title),
                message = L.text(context, localePrefs, R.string.mobile_settings_menu_accessDenied_message),
                modifier = Modifier.padding(padding).padding(16.dp),
            )
            return@Scaffold
        }

        val groups = SettingsMenuLogic.visibleGroups(
            features = shell.platformFeatures,
            permissions = shell.permissions,
            query = searchText,
            titleResolver = ::stringByName,
        )

        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .verticalScroll(rememberScrollState())
                .padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp),
        ) {
            Text(
                L.text(R.string.mobile_settings_menu_description),
                color = textSecondary(),
                fontSize = 14.sp,
            )
            OutlinedTextField(
                value = searchText,
                onValueChange = { searchText = it },
                modifier = Modifier.fillMaxWidth(),
                singleLine = true,
                leadingIcon = { Icon(Icons.Default.Search, contentDescription = null) },
                label = { Text(L.text(R.string.mobile_settings_menu_search_prompt)) },
            )
            if (groups.isEmpty()) {
                LmsEmptyState(
                    icon = Icons.Default.Search,
                    title = L.text(R.string.mobile_settings_menu_empty_title),
                    message = L.text(R.string.mobile_settings_menu_empty_message),
                )
            } else {
                groups.forEach { group ->
                    Text(
                        stringByName(group.titleResName).uppercase(),
                        fontWeight = FontWeight.SemiBold,
                        fontSize = 12.sp,
                        color = textSecondary(),
                    )
                    LmsCard {
                        Column {
                            group.items.forEach { item ->
                                Row(
                                    modifier = Modifier
                                        .fillMaxWidth()
                                        .clickable { openPage = item.id }
                                        .padding(horizontal = 14.dp, vertical = 12.dp),
                                    verticalAlignment = Alignment.CenterVertically,
                                    horizontalArrangement = Arrangement.spacedBy(12.dp),
                                ) {
                                    Column(modifier = Modifier.weight(1f)) {
                                        Text(
                                            stringByName(item.titleResName),
                                            fontWeight = FontWeight.SemiBold,
                                            color = textPrimary(),
                                        )
                                        Text(
                                            stringByName(item.subtitleResName),
                                            fontSize = 12.sp,
                                            color = textSecondary(),
                                        )
                                    }
                                    Icon(
                                        Icons.AutoMirrored.Filled.KeyboardArrowRight,
                                        contentDescription = null,
                                        tint = textSecondary(),
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
