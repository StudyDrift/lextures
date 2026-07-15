package com.lextures.android.features.settings.admin

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Lock
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
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalePreferences
import com.lextures.android.core.lms.AiModelOption
import com.lextures.android.core.lms.AiModelsAdminLogic
import com.lextures.android.core.lms.AiSettingsResponse
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.network.ApiError
import com.lextures.android.features.home.HomeShellState
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsSkeletonList
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun AiModelsSettingsScreen(
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

    var imageModelId by remember { mutableStateOf("") }
    var courseSetupModelId by remember { mutableStateOf("") }
    var notebookFlashcardsModelId by remember { mutableStateOf("") }
    var vibeActivityModelId by remember { mutableStateOf("") }
    var graderAgentModelId by remember { mutableStateOf("") }
    var openRouterApiKey by remember { mutableStateOf("") }
    var openRouterApiKeyBaseline by remember { mutableStateOf("") }
    var textModels by remember { mutableStateOf(AiModelsAdminLogic.FALLBACK_TEXT_MODELS) }
    var imageModels by remember { mutableStateOf(AiModelsAdminLogic.FALLBACK_IMAGE_MODELS) }
    var modelsConfigured by remember { mutableStateOf(false) }
    var modelsError by remember { mutableStateOf<String?>(null) }
    var modelsRefreshing by remember { mutableStateOf(false) }
    var loading by remember { mutableStateOf(true) }
    var saving by remember { mutableStateOf(false) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var savedMessage by remember { mutableStateOf<String?>(null) }

    fun applySettings(settings: AiSettingsResponse) {
        imageModelId = settings.imageModelId
        courseSetupModelId = settings.courseSetupModelId
        notebookFlashcardsModelId = settings.notebookFlashcardsModelId
        vibeActivityModelId = settings.vibeActivityModelId
        graderAgentModelId = settings.graderAgentModelId
        val key = settings.openRouterApiKey.orEmpty()
        openRouterApiKey = key
        openRouterApiKeyBaseline = key
    }

    suspend fun loadModels(token: String) {
        modelsError = null
        try {
            val text = LmsApi.fetchAiModels("text", token)
            val image = LmsApi.fetchAiModels("image", token)
            modelsConfigured = text.configured || image.configured
            textModels = text.models.ifEmpty { AiModelsAdminLogic.FALLBACK_TEXT_MODELS }
            imageModels = image.models.ifEmpty { AiModelsAdminLogic.FALLBACK_IMAGE_MODELS }
        } catch (e: Exception) {
            modelsError = userFacing(e, context, localePrefs, R.string.mobile_admin_ai_models_modelsError)
            textModels = AiModelsAdminLogic.FALLBACK_TEXT_MODELS
            imageModels = AiModelsAdminLogic.FALLBACK_IMAGE_MODELS
            modelsConfigured = false
        }
    }

    suspend fun load(token: String) {
        loading = true
        errorMessage = null
        try {
            applySettings(LmsApi.fetchAiSettings(token))
            loadModels(token)
        } catch (e: Exception) {
            errorMessage = userFacing(e, context, localePrefs, R.string.mobile_admin_ai_models_loadError)
            textModels = AiModelsAdminLogic.FALLBACK_TEXT_MODELS
            imageModels = AiModelsAdminLogic.FALLBACK_IMAGE_MODELS
            modelsConfigured = false
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
                    title = { Text(L.text(context, localePrefs, R.string.mobile_admin_ai_models_title)) },
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

    val saveDisabled = AiModelsAdminLogic.isSaveDisabled(
        saving = saving,
        imageModelId = imageModelId,
        courseSetupModelId = courseSetupModelId,
        notebookFlashcardsModelId = notebookFlashcardsModelId,
        vibeActivityModelId = vibeActivityModelId,
    )

    Scaffold(
        modifier = modifier.fillMaxSize(),
        topBar = {
            TopAppBar(
                title = { Text(L.text(context, localePrefs, R.string.mobile_admin_ai_models_title)) },
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
                text = L.text(context, localePrefs, R.string.mobile_admin_ai_models_description),
                fontSize = 14.sp,
                color = textSecondary(),
            )
            errorMessage?.let { LmsErrorBanner(message = it) }
            modelsError?.let { LmsErrorBanner(message = it) }

            if (loading) {
                LmsSkeletonList(count = 5)
            } else {
                if (!modelsConfigured) {
                    Text(
                        text = L.text(context, localePrefs, R.string.mobile_admin_ai_models_keyRequired),
                        fontSize = 12.sp,
                        color = LexturesColors.Amber,
                    )
                }

                LmsCard {
                    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                        Text(
                            L.text(context, localePrefs, R.string.mobile_admin_ai_models_apiKey),
                            fontWeight = FontWeight.SemiBold,
                        )
                        OutlinedTextField(
                            value = openRouterApiKey,
                            onValueChange = { openRouterApiKey = it },
                            modifier = Modifier.fillMaxWidth(),
                            singleLine = true,
                            visualTransformation = PasswordVisualTransformation(),
                            placeholder = { Text(AiModelsAdminLogic.PLATFORM_SECRET_PLACEHOLDER) },
                        )
                        Text(
                            L.text(context, localePrefs, R.string.mobile_admin_ai_models_apiKeyHint),
                            fontSize = 12.sp,
                            color = textSecondary(),
                        )
                    }
                }

                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                ) {
                    Text(
                        L.text(context, localePrefs, R.string.mobile_admin_ai_models_pickersTitle),
                        fontWeight = FontWeight.Bold,
                        fontSize = 18.sp,
                    )
                    OutlinedButton(
                        onClick = {
                            val token = accessToken ?: return@OutlinedButton
                            scope.launch {
                                modelsRefreshing = true
                                loadModels(token)
                                modelsRefreshing = false
                            }
                        },
                        enabled = !modelsRefreshing && !saving,
                    ) {
                        if (modelsRefreshing) {
                            CircularProgressIndicator(modifier = Modifier.size(16.dp), strokeWidth = 2.dp)
                        } else {
                            Text(L.text(context, localePrefs, R.string.mobile_admin_ai_models_refresh))
                        }
                    }
                }

                ModelDropdown(
                    label = L.text(context, localePrefs, R.string.mobile_admin_ai_models_courseSetup),
                    hint = L.text(context, localePrefs, R.string.mobile_admin_ai_models_courseSetupHint),
                    selectedId = courseSetupModelId,
                    models = AiModelsAdminLogic.modelsWithSelection(textModels, courseSetupModelId),
                    onSelect = { courseSetupModelId = it },
                    enabled = !saving,
                )
                ModelDropdown(
                    label = L.text(context, localePrefs, R.string.mobile_admin_ai_models_flashcards),
                    hint = L.text(context, localePrefs, R.string.mobile_admin_ai_models_flashcardsHint),
                    selectedId = notebookFlashcardsModelId,
                    models = AiModelsAdminLogic.modelsWithSelection(textModels, notebookFlashcardsModelId),
                    onSelect = { notebookFlashcardsModelId = it },
                    enabled = !saving,
                )
                ModelDropdown(
                    label = L.text(context, localePrefs, R.string.mobile_admin_ai_models_vibe),
                    hint = L.text(context, localePrefs, R.string.mobile_admin_ai_models_vibeHint),
                    selectedId = vibeActivityModelId,
                    models = AiModelsAdminLogic.modelsWithSelection(textModels, vibeActivityModelId),
                    onSelect = { vibeActivityModelId = it },
                    enabled = !saving,
                )
                ModelDropdown(
                    label = L.text(context, localePrefs, R.string.mobile_admin_ai_models_grader),
                    hint = L.text(context, localePrefs, R.string.mobile_admin_ai_models_graderHint),
                    selectedId = graderAgentModelId,
                    models = AiModelsAdminLogic.modelsWithSelection(textModels, graderAgentModelId),
                    onSelect = { graderAgentModelId = it },
                    enabled = !saving,
                )
                ModelDropdown(
                    label = L.text(context, localePrefs, R.string.mobile_admin_ai_models_image),
                    hint = L.text(context, localePrefs, R.string.mobile_admin_ai_models_imageHint),
                    selectedId = imageModelId,
                    models = AiModelsAdminLogic.modelsWithSelection(imageModels, imageModelId),
                    onSelect = { imageModelId = it },
                    enabled = !saving,
                )

                Button(
                    onClick = {
                        val token = accessToken ?: return@Button
                        scope.launch {
                            saving = true
                            savedMessage = null
                            errorMessage = null
                            val body = AiModelsAdminLogic.buildAiSettingsSaveRequest(
                                imageModelId = imageModelId,
                                courseSetupModelId = courseSetupModelId,
                                notebookFlashcardsModelId = notebookFlashcardsModelId,
                                vibeActivityModelId = vibeActivityModelId,
                                graderAgentModelId = graderAgentModelId,
                                openRouterApiKey = openRouterApiKey,
                                openRouterApiKeyBaseline = openRouterApiKeyBaseline,
                            )
                            try {
                                applySettings(LmsApi.putAiSettings(body, token))
                                savedMessage = L.text(context, localePrefs, R.string.mobile_admin_ai_models_saved)
                                loadModels(token)
                            } catch (e: Exception) {
                                errorMessage = userFacing(
                                    e,
                                    context,
                                    localePrefs,
                                    R.string.mobile_admin_ai_models_saveError,
                                )
                            }
                            saving = false
                        }
                    },
                    enabled = !saveDisabled,
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    if (saving) {
                        CircularProgressIndicator(modifier = Modifier.size(18.dp), strokeWidth = 2.dp)
                    } else {
                        Text(L.text(context, localePrefs, R.string.mobile_admin_ai_models_save))
                    }
                }
                savedMessage?.let {
                    Text(it, color = LexturesColors.BrandTeal, fontSize = 12.sp)
                }
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun ModelDropdown(
    label: String,
    hint: String,
    selectedId: String,
    models: List<AiModelOption>,
    onSelect: (String) -> Unit,
    enabled: Boolean,
) {
    var expanded by remember { mutableStateOf(false) }
    val selected = models.firstOrNull { it.id == selectedId }
    LmsCard {
        Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
            Text(label, fontWeight = FontWeight.SemiBold)
            ExposedDropdownMenuBox(expanded = expanded, onExpandedChange = { if (enabled) expanded = it }) {
                OutlinedTextField(
                    value = selected?.let { AiModelsAdminLogic.modelDisplayLabel(it) } ?: selectedId,
                    onValueChange = {},
                    readOnly = true,
                    enabled = enabled,
                    modifier = Modifier
                        .menuAnchor()
                        .fillMaxWidth(),
                    trailingIcon = { ExposedDropdownMenuDefaults.TrailingIcon(expanded = expanded) },
                )
                ExposedDropdownMenu(expanded = expanded, onDismissRequest = { expanded = false }) {
                    models.forEach { model ->
                        DropdownMenuItem(
                            text = { Text(AiModelsAdminLogic.modelDisplayLabel(model)) },
                            onClick = {
                                onSelect(model.id)
                                expanded = false
                            },
                        )
                    }
                }
            }
            Text(hint, fontSize = 12.sp, color = textSecondary())
        }
    }
}

private fun userFacing(
    error: Exception,
    context: android.content.Context,
    localePrefs: LocalePreferences,
    fallbackRes: Int,
): String {
    if (error is ApiError.HttpStatus) {
        val message = error.message
        if (!message.isNullOrBlank()) return message
    }
    return L.text(context, localePrefs, fallbackRes)
}
