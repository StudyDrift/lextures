package com.lextures.android.features.settings.admin

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Lock
import androidx.compose.material.icons.automirrored.filled.Notes
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ExposedDropdownMenuBox
import androidx.compose.material3.ExposedDropdownMenuDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
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
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalePreferences
import com.lextures.android.core.lms.AiModelsAdminLogic
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.SystemPromptItem
import com.lextures.android.core.network.ApiError
import com.lextures.android.features.home.HomeShellState
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsSkeletonList
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SystemPromptsScreen(
    session: AuthSession,
    shell: HomeShellState,
    localePrefs: LocalePreferences,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val accessToken by session.accessToken.collectAsState()
    val canView = AiModelsAdminLogic.canView(shell.platformFeatures, shell.permissions)

    var prompts by remember { mutableStateOf<List<SystemPromptItem>>(emptyList()) }
    var selectedKey by remember { mutableStateOf("") }
    var draft by remember { mutableStateOf("") }
    var loading by remember { mutableStateOf(true) }
    var saving by remember { mutableStateOf(false) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var savedMessage by remember { mutableStateOf<String?>(null) }
    var showEditor by remember { mutableStateOf(false) }
    var promptMenuExpanded by remember { mutableStateOf(false) }

    suspend fun load(token: String) {
        loading = true
        errorMessage = null
        try {
            val list = LmsApi.fetchSystemPrompts(token)
            prompts = list
            if (list.isEmpty()) {
                selectedKey = ""
                draft = ""
            } else {
                val existing = list.firstOrNull { it.key == selectedKey }
                if (existing != null) {
                    draft = existing.content
                } else {
                    selectedKey = list.first().key
                    draft = list.first().content
                }
            }
        } catch (e: Exception) {
            errorMessage = if (e is ApiError.HttpStatus && !e.message.isNullOrBlank()) {
                e.message!!
            } else {
                L.text(context, localePrefs, R.string.mobile_admin_ai_prompts_loadError)
            }
        }
        loading = false
    }

    LaunchedEffect(accessToken, canView) {
        val token = accessToken ?: return@LaunchedEffect
        if (canView) load(token)
    }

    if (!canView) {
        Scaffold(
            modifier = modifier.fillMaxSize(),
            topBar = {
                TopAppBar(
                    title = { Text(L.text(context, localePrefs, R.string.mobile_admin_ai_prompts_title)) },
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

    if (showEditor) {
        AlertDialog(
            onDismissRequest = { showEditor = false },
            title = {
                Text(prompts.firstOrNull { it.key == selectedKey }?.label
                    ?: L.text(context, localePrefs, R.string.mobile_admin_ai_prompts_title))
            },
            text = {
                OutlinedTextField(
                    value = draft,
                    onValueChange = { draft = it },
                    modifier = Modifier
                        .fillMaxWidth()
                        .heightIn(min = 280.dp),
                    textStyle = androidx.compose.ui.text.TextStyle(fontFamily = FontFamily.Monospace, fontSize = 13.sp),
                )
            },
            confirmButton = {
                TextButton(onClick = { showEditor = false }) {
                    Text(L.text(context, localePrefs, R.string.mobile_common_close))
                }
            },
            dismissButton = {
                TextButton(onClick = {
                    draft = prompts.firstOrNull { it.key == selectedKey }?.content.orEmpty()
                    showEditor = false
                }) {
                    Text(L.text(context, localePrefs, R.string.mobile_cancel))
                }
            },
        )
    }

    Scaffold(
        modifier = modifier.fillMaxSize(),
        topBar = {
            TopAppBar(
                title = { Text(L.text(context, localePrefs, R.string.mobile_admin_ai_prompts_title)) },
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
                text = L.text(context, localePrefs, R.string.mobile_admin_ai_prompts_description),
                fontSize = 14.sp,
                color = textSecondary(),
            )
            errorMessage?.let { LmsErrorBanner(message = it) }

            if (loading) {
                LmsSkeletonList(count = 3)
            } else if (prompts.isEmpty()) {
                LmsEmptyState(
                    icon = Icons.AutoMirrored.Filled.Notes,
                    title = L.text(context, localePrefs, R.string.mobile_admin_ai_prompts_emptyTitle),
                    message = L.text(context, localePrefs, R.string.mobile_admin_ai_prompts_emptyMessage),
                )
            } else {
                val selected = prompts.firstOrNull { it.key == selectedKey }
                LmsCard {
                    Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                        Text(
                            L.text(context, localePrefs, R.string.mobile_admin_ai_prompts_select),
                            fontWeight = FontWeight.SemiBold,
                        )
                        ExposedDropdownMenuBox(
                            expanded = promptMenuExpanded,
                            onExpandedChange = { promptMenuExpanded = it },
                        ) {
                            OutlinedTextField(
                                value = selected?.label ?: selectedKey,
                                onValueChange = {},
                                readOnly = true,
                                modifier = Modifier.menuAnchor().fillMaxWidth(),
                                trailingIcon = { ExposedDropdownMenuDefaults.TrailingIcon(expanded = promptMenuExpanded) },
                            )
                            ExposedDropdownMenu(
                                expanded = promptMenuExpanded,
                                onDismissRequest = { promptMenuExpanded = false },
                            ) {
                                prompts.forEach { prompt ->
                                    DropdownMenuItem(
                                        text = { Text(prompt.label) },
                                        onClick = {
                                            selectedKey = prompt.key
                                            draft = prompt.content
                                            promptMenuExpanded = false
                                        },
                                    )
                                }
                            }
                        }
                        selected?.updatedAt?.let { updated ->
                            Text(
                                L.format(
                                    context,
                                    localePrefs,
                                    R.string.mobile_admin_ai_prompts_updatedAt,
                                    updated,
                                ),
                                fontSize = 12.sp,
                                color = textSecondary(),
                            )
                        }
                        OutlinedButton(
                            onClick = { showEditor = true },
                            modifier = Modifier.fillMaxWidth(),
                        ) {
                            Text(L.text(context, localePrefs, R.string.mobile_admin_ai_prompts_edit))
                        }
                        Text(
                            text = draft,
                            fontFamily = FontFamily.Monospace,
                            fontSize = 12.sp,
                            color = textSecondary(),
                            maxLines = 8,
                        )
                    }
                }

                Button(
                    onClick = {
                        val token = accessToken ?: return@Button
                        val key = selectedKey.trim()
                        if (key.isEmpty()) return@Button
                        scope.launch {
                            saving = true
                            savedMessage = null
                            errorMessage = null
                            try {
                                val row = LmsApi.putSystemPrompt(key, draft, token)
                                prompts = prompts.map {
                                    if (it.key == row.key) {
                                        it.copy(
                                            content = row.content,
                                            label = row.label.ifEmpty { it.label },
                                            updatedAt = row.updatedAt,
                                        )
                                    } else {
                                        it
                                    }
                                }
                                draft = row.content
                                savedMessage = L.text(context, localePrefs, R.string.mobile_admin_ai_prompts_saved)
                            } catch (e: Exception) {
                                errorMessage = if (e is ApiError.HttpStatus && !e.message.isNullOrBlank()) {
                                    e.message!!
                                } else {
                                    L.text(context, localePrefs, R.string.mobile_admin_ai_prompts_saveError)
                                }
                            }
                            saving = false
                        }
                    },
                    enabled = selectedKey.isNotEmpty() && !saving,
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    if (saving) {
                        CircularProgressIndicator(modifier = Modifier.size(18.dp), strokeWidth = 2.dp)
                    } else {
                        Text(L.text(context, localePrefs, R.string.mobile_admin_ai_prompts_save))
                    }
                }
                savedMessage?.let {
                    Text(it, color = LexturesColors.BrandTeal, fontSize = 12.sp)
                }
            }
        }
    }
}
