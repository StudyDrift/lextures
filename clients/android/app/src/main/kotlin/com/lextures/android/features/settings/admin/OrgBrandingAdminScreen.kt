package com.lextures.android.features.settings.admin

import android.content.Intent
import android.net.Uri
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Lock
import androidx.compose.material.icons.filled.Palette
import androidx.compose.material3.Button
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ExposedDropdownMenuBox
import androidx.compose.material3.ExposedDropdownMenuDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.OutlinedButton
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
import androidx.compose.runtime.mutableStateMapOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import coil.compose.AsyncImage
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.config.AppConfiguration
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalePreferences
import com.lextures.android.core.lms.AIGovernanceConfig
import com.lextures.android.core.lms.AIProviderSettings
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.OrgBrandingAdminLogic
import com.lextures.android.core.lms.OrgBrandingResponse
import com.lextures.android.core.network.ApiError
import com.lextures.android.features.home.HomeShellState
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsSkeletonList
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

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
    val scope = rememberCoroutineScope()
    val accessToken by session.accessToken.collectAsState()
    val genericError = L.text(context, localePrefs, R.string.mobile_admin_orgBranding_error)
    val canView = OrgBrandingAdminLogic.canView(shell.platformFeatures, shell.permissions)

    var orgId by remember { mutableStateOf("") }
    var branding by remember { mutableStateOf(OrgBrandingResponse()) }
    var governance by remember { mutableStateOf<AIGovernanceConfig?>(null) }
    var providerSettings by remember { mutableStateOf<AIProviderSettings?>(null) }
    var governanceAvailable by remember { mutableStateOf(true) }
    var providerAvailable by remember { mutableStateOf(true) }
    var loading by remember { mutableStateOf(true) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var statusMessage by remember { mutableStateOf<String?>(null) }

    // Branding form
    var primaryColor by remember { mutableStateOf(OrgBrandingAdminLogic.DEFAULT_PRIMARY_COLOR) }
    var secondaryColor by remember { mutableStateOf(OrgBrandingAdminLogic.DEFAULT_SECONDARY_COLOR) }
    var emailDisplayName by remember { mutableStateOf("") }
    var logoUrl by remember { mutableStateOf<String?>(null) }
    var faviconUrl by remember { mutableStateOf<String?>(null) }
    var customDomain by remember { mutableStateOf<String?>(null) }
    var contrastWarning by remember { mutableStateOf(false) }
    var contrastRatio by remember { mutableStateOf<Double?>(null) }
    var savingBranding by remember { mutableStateOf(false) }
    var uploadingLogo by remember { mutableStateOf(false) }

    // Governance form
    val enabledFeatures = remember { mutableStateMapOf<String, Boolean>() }
    var allowedModelsText by remember { mutableStateOf("") }
    var savingGovernance by remember { mutableStateOf(false) }

    // Provider form
    var provider by remember { mutableStateOf(OrgBrandingAdminLogic.DEFAULT_PROVIDER) }
    var modelAlias by remember { mutableStateOf(OrgBrandingAdminLogic.DEFAULT_MODEL_ALIAS) }
    var fallbackProvider by remember { mutableStateOf("") }
    var byokKey by remember { mutableStateOf("") }
    var byokConfigured by remember { mutableStateOf(false) }
    var savingProvider by remember { mutableStateOf(false) }
    var testingProvider by remember { mutableStateOf(false) }

    fun applyBranding(data: OrgBrandingResponse) {
        branding = data
        logoUrl = data.logoUrl
        faviconUrl = data.faviconUrl
        primaryColor = data.primaryColor
        secondaryColor = data.secondaryColor
        emailDisplayName = data.customEmailDisplayName.orEmpty()
        customDomain = data.customDomain
        contrastWarning = data.contrastWarningPrimary == true
        contrastRatio = data.contrastRatioPrimary
    }

    fun applyGovernance(data: AIGovernanceConfig?) {
        governance = data
        enabledFeatures.clear()
        data?.featuresEnabled?.forEach { (k, v) -> enabledFeatures[k] = v }
        allowedModelsText = OrgBrandingAdminLogic.allowedModelsText(data?.allowedModels)
    }

    fun applyProvider(data: AIProviderSettings?) {
        providerSettings = data
        provider = data?.provider ?: OrgBrandingAdminLogic.DEFAULT_PROVIDER
        modelAlias = data?.modelAlias ?: OrgBrandingAdminLogic.DEFAULT_MODEL_ALIAS
        fallbackProvider = data?.fallbackProvider.orEmpty()
        byokConfigured = data?.byokConfigured == true
        byokKey = OrgBrandingAdminLogic.displaySecretField(byokConfigured)
    }

    suspend fun load(token: String) {
        loading = true
        errorMessage = null
        if (orgId.isEmpty()) {
            orgId = OrgBrandingAdminLogic.resolveOrgId(token, emptyList()).orEmpty()
        }
        if (orgId.isEmpty()) {
            loading = false
            return
        }

        runCatching { LmsApi.fetchOrgBranding(orgId, token) }
            .onSuccess { applyBranding(it) }
            .onFailure {
                errorMessage = OrgBrandingAdminLogic.userFacingError(it, genericError)
            }

        runCatching { LmsApi.fetchAIGovernanceConfig(token) }
            .onSuccess {
                applyGovernance(it)
                governanceAvailable = true
            }
            .onFailure { err ->
                val code = (err as? ApiError.HttpStatus)?.code
                governanceAvailable = code != 403 && code != 404
                if (code == 403 || code == 404) {
                    governanceAvailable = false
                    applyGovernance(null)
                } else {
                    governanceAvailable = false
                    applyGovernance(null)
                }
            }

        runCatching { LmsApi.fetchAIProviderSettings(token) }
            .onSuccess {
                applyProvider(it)
                providerAvailable = true
            }
            .onFailure { err ->
                val code = (err as? ApiError.HttpStatus)?.code
                if (code == 403 || code == 404) {
                    providerAvailable = false
                    applyProvider(null)
                } else {
                    providerAvailable = false
                    applyProvider(null)
                }
            }

        loading = false
    }

    LaunchedEffect(accessToken) {
        val token = accessToken ?: return@LaunchedEffect
        if (canView) load(token)
    }

    val logoPicker = rememberLauncherForActivityResult(ActivityResultContracts.GetContent()) { uri ->
        val token = accessToken ?: return@rememberLauncherForActivityResult
        if (uri == null || orgId.isEmpty()) return@rememberLauncherForActivityResult
        scope.launch {
            uploadingLogo = true
            errorMessage = null
            statusMessage = null
            runCatching {
                val bytes = withContext(Dispatchers.IO) {
                    context.contentResolver.openInputStream(uri)?.use { it.readBytes() }
                        ?: error("empty")
                }
                val mime = context.contentResolver.getType(uri) ?: "image/jpeg"
                val name = uri.lastPathSegment ?: "logo.jpg"
                LmsApi.uploadOrgBrandingLogo(orgId, bytes, name, mime, token)
            }.onSuccess { upload ->
                if (!upload.url.isNullOrBlank()) {
                    logoUrl = upload.url
                    statusMessage = L.text(context, localePrefs, R.string.mobile_admin_orgBranding_branding_logoUploaded)
                }
            }.onFailure {
                errorMessage = OrgBrandingAdminLogic.userFacingError(it, genericError)
            }
            uploadingLogo = false
        }
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
            Box(Modifier.fillMaxSize().padding(padding)) {
                LmsEmptyState(
                    icon = Icons.Filled.Lock,
                    title = L.text(context, localePrefs, R.string.mobile_admin_orgBranding_accessDeniedTitle),
                    message = L.text(context, localePrefs, R.string.mobile_admin_orgBranding_accessDeniedMessage),
                )
            }
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
                .fillMaxSize()
                .padding(padding)
                .verticalScroll(rememberScrollState())
                .padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp),
        ) {
            Text(
                L.text(context, localePrefs, R.string.mobile_admin_orgBranding_description),
                color = textSecondary(),
                fontSize = 14.sp,
            )

            errorMessage?.let { LmsErrorBanner(it) }
            statusMessage?.let {
                Text(it, color = LexturesColors.BrandTeal, fontSize = 12.sp)
            }

            if (loading && orgId.isEmpty()) {
                LmsSkeletonList(count = 3)
            } else if (orgId.isEmpty()) {
                LmsEmptyState(
                    icon = Icons.Filled.Palette,
                    title = L.text(context, localePrefs, R.string.mobile_admin_orgBranding_noOrgTitle),
                    message = L.text(context, localePrefs, R.string.mobile_admin_orgBranding_noOrgMessage),
                )
            } else {
                BrandingCard(
                    localePrefs = localePrefs,
                    logoUrl = logoUrl,
                    primaryColor = primaryColor,
                    onPrimaryColorChange = { primaryColor = it },
                    secondaryColor = secondaryColor,
                    onSecondaryColorChange = { secondaryColor = it },
                    emailDisplayName = emailDisplayName,
                    onEmailChange = { emailDisplayName = it },
                    customDomain = customDomain,
                    contrastWarning = contrastWarning,
                    contrastRatio = contrastRatio,
                    saving = savingBranding,
                    uploading = uploadingLogo,
                    onPickLogo = { logoPicker.launch("image/*") },
                    onSave = {
                        val token = accessToken ?: return@BrandingCard
                        scope.launch {
                            savingBranding = true
                            errorMessage = null
                            statusMessage = null
                            if (!OrgBrandingAdminLogic.isValidHexColor(primaryColor) ||
                                !OrgBrandingAdminLogic.isValidHexColor(secondaryColor)
                            ) {
                                errorMessage = L.text(
                                    context,
                                    localePrefs,
                                    R.string.mobile_admin_orgBranding_branding_invalidColor,
                                )
                                savingBranding = false
                                return@launch
                            }
                            runCatching {
                                LmsApi.putOrgBranding(
                                    orgId,
                                    OrgBrandingAdminLogic.brandingPutBody(
                                        logoUrl = logoUrl,
                                        faviconUrl = faviconUrl,
                                        primaryColor = primaryColor,
                                        secondaryColor = secondaryColor,
                                        customEmailDisplayName = emailDisplayName,
                                    ),
                                    token,
                                )
                            }.onSuccess {
                                applyBranding(it)
                                statusMessage = L.text(
                                    context,
                                    localePrefs,
                                    R.string.mobile_admin_orgBranding_branding_saved,
                                )
                            }.onFailure {
                                errorMessage = OrgBrandingAdminLogic.userFacingError(it, genericError)
                            }
                            savingBranding = false
                        }
                    },
                    onOpenWeb = {
                        val url = AppConfiguration.webUrl(OrgBrandingAdminLogic.webBrandingPath())
                        context.startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(url)))
                    },
                )

                GovernanceCard(
                    localePrefs = localePrefs,
                    available = governanceAvailable,
                    enabledFeatures = enabledFeatures,
                    allowedModelsText = allowedModelsText,
                    onAllowedModelsChange = { allowedModelsText = it },
                    saving = savingGovernance,
                    onSave = {
                        val token = accessToken ?: return@GovernanceCard
                        scope.launch {
                            savingGovernance = true
                            errorMessage = null
                            statusMessage = null
                            runCatching {
                                LmsApi.putAIGovernanceConfig(
                                    OrgBrandingAdminLogic.aiConfigPutBody(
                                        enabled = enabledFeatures.toMap(),
                                        allowedModelsText = allowedModelsText,
                                    ),
                                    token,
                                )
                            }.onSuccess {
                                applyGovernance(it)
                                statusMessage = L.text(
                                    context,
                                    localePrefs,
                                    R.string.mobile_admin_orgBranding_ai_saved,
                                )
                            }.onFailure {
                                errorMessage = OrgBrandingAdminLogic.userFacingError(it, genericError)
                            }
                            savingGovernance = false
                        }
                    },
                )

                ProviderCard(
                    localePrefs = localePrefs,
                    available = providerAvailable,
                    settings = providerSettings,
                    provider = provider,
                    onProviderChange = { provider = it },
                    modelAlias = modelAlias,
                    onModelAliasChange = { modelAlias = it },
                    fallbackProvider = fallbackProvider,
                    onFallbackChange = { fallbackProvider = it },
                    byokKey = byokKey,
                    onByokChange = { byokKey = it },
                    byokConfigured = byokConfigured,
                    saving = savingProvider,
                    testing = testingProvider,
                    onSave = {
                        val token = accessToken ?: return@ProviderCard
                        scope.launch {
                            savingProvider = true
                            errorMessage = null
                            statusMessage = null
                            runCatching {
                                LmsApi.putAIProviderSettings(
                                    OrgBrandingAdminLogic.aiProviderPutBody(
                                        provider = provider,
                                        modelAlias = modelAlias,
                                        fallbackProvider = fallbackProvider,
                                        byokKey = byokKey,
                                    ),
                                    token,
                                )
                            }.onSuccess {
                                applyProvider(it)
                                statusMessage = L.text(
                                    context,
                                    localePrefs,
                                    R.string.mobile_admin_orgBranding_provider_saved,
                                )
                            }.onFailure {
                                errorMessage = OrgBrandingAdminLogic.userFacingError(it, genericError)
                            }
                            savingProvider = false
                        }
                    },
                    onTest = {
                        val token = accessToken ?: return@ProviderCard
                        scope.launch {
                            testingProvider = true
                            errorMessage = null
                            statusMessage = null
                            runCatching {
                                LmsApi.testAIProviderSettings(token)
                            }.onSuccess { result ->
                                val name = result.provider ?: provider
                                val ms = result.latencyMs?.toInt()?.toString() ?: "?"
                                val preview = result.responsePreview ?: "OK"
                                statusMessage = L.format(
                                    context,
                                    localePrefs,
                                    R.string.mobile_admin_orgBranding_provider_testSuccess,
                                    name,
                                    ms,
                                    preview,
                                )
                            }.onFailure {
                                errorMessage = OrgBrandingAdminLogic.userFacingError(it, genericError)
                            }
                            testingProvider = false
                        }
                    },
                )
            }
        }
    }
}

@Composable
private fun BrandingCard(
    localePrefs: LocalePreferences,
    logoUrl: String?,
    primaryColor: String,
    onPrimaryColorChange: (String) -> Unit,
    secondaryColor: String,
    onSecondaryColorChange: (String) -> Unit,
    emailDisplayName: String,
    onEmailChange: (String) -> Unit,
    customDomain: String?,
    contrastWarning: Boolean,
    contrastRatio: Double?,
    saving: Boolean,
    uploading: Boolean,
    onPickLogo: () -> Unit,
    onSave: () -> Unit,
    onOpenWeb: () -> Unit,
) {
    val context = LocalContext.current
    val hasContrast = OrgBrandingAdminLogic.hasContrastWarning(
        primaryColor = primaryColor,
        serverWarning = contrastWarning,
        serverRatio = contrastRatio,
    )
    val colorsValid = OrgBrandingAdminLogic.isValidHexColor(primaryColor) &&
        OrgBrandingAdminLogic.isValidHexColor(secondaryColor)

    LmsCard {
        Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
            Text(
                L.text(context, localePrefs, R.string.mobile_admin_orgBranding_branding_title),
                fontWeight = FontWeight.SemiBold,
                color = textPrimary(),
            )
            Text(
                L.text(context, localePrefs, R.string.mobile_admin_orgBranding_branding_intro),
                fontSize = 12.sp,
                color = textSecondary(),
            )

            Row(
                horizontalArrangement = Arrangement.spacedBy(12.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                val resolved = resolveAssetUrl(logoUrl)
                if (resolved != null) {
                    AsyncImage(
                        model = resolved,
                        contentDescription = L.text(
                            context,
                            localePrefs,
                            R.string.mobile_admin_orgBranding_branding_logoPreview,
                        ),
                        modifier = Modifier
                            .size(72.dp)
                            .clip(RoundedCornerShape(8.dp)),
                        contentScale = ContentScale.Fit,
                    )
                } else {
                    Box(
                        modifier = Modifier
                            .size(72.dp)
                            .border(1.dp, textSecondary().copy(alpha = 0.3f), RoundedCornerShape(8.dp)),
                        contentAlignment = Alignment.Center,
                    ) {
                        Icon(Icons.Filled.Palette, contentDescription = null, tint = textSecondary())
                    }
                }
                OutlinedButton(onClick = onPickLogo, enabled = !uploading && !saving) {
                    Text(
                        if (uploading) {
                            L.text(context, localePrefs, R.string.mobile_admin_orgBranding_branding_uploading)
                        } else {
                            L.text(context, localePrefs, R.string.mobile_admin_orgBranding_branding_uploadLogo)
                        },
                    )
                }
            }

            ColorField(
                label = L.text(context, localePrefs, R.string.mobile_admin_orgBranding_branding_primaryColor),
                value = primaryColor,
                onValueChange = onPrimaryColorChange,
            )
            ColorField(
                label = L.text(context, localePrefs, R.string.mobile_admin_orgBranding_branding_secondaryColor),
                value = secondaryColor,
                onValueChange = onSecondaryColorChange,
            )

            if (hasContrast) {
                val ratio = contrastRatio ?: OrgBrandingAdminLogic.contrastRatioAgainstWhite(primaryColor)
                Text(
                    if (ratio != null) {
                        L.format(
                            context,
                            localePrefs,
                            R.string.mobile_admin_orgBranding_branding_contrastWarningWithRatio,
                            String.format("%.2f", ratio),
                        )
                    } else {
                        L.text(context, localePrefs, R.string.mobile_admin_orgBranding_branding_contrastWarning)
                    },
                    color = Color(0xFFE65100),
                    fontSize = 12.sp,
                )
            }

            Box(
                modifier = Modifier
                    .fillMaxWidth()
                    .height(36.dp)
                    .clip(RoundedCornerShape(8.dp))
                    .background(parseHexColor(primaryColor) ?: Color(0xFF4F46E5)),
                contentAlignment = Alignment.Center,
            ) {
                Text(
                    L.text(context, localePrefs, R.string.mobile_admin_orgBranding_branding_previewButton),
                    color = Color.White,
                    fontWeight = FontWeight.SemiBold,
                    fontSize = 12.sp,
                )
            }

            OutlinedTextField(
                value = emailDisplayName,
                onValueChange = onEmailChange,
                modifier = Modifier.fillMaxWidth(),
                label = {
                    Text(L.text(context, localePrefs, R.string.mobile_admin_orgBranding_branding_emailDisplayName))
                },
                singleLine = true,
            )
            Text(
                L.text(context, localePrefs, R.string.mobile_admin_orgBranding_branding_emailDisplayNameHint),
                fontSize = 11.sp,
                color = textSecondary(),
            )

            Text(
                L.text(context, localePrefs, R.string.mobile_admin_orgBranding_branding_customDomain),
                fontWeight = FontWeight.Medium,
                color = textPrimary(),
            )
            Text(
                customDomain?.takeIf { it.isNotBlank() }
                    ?: L.text(context, localePrefs, R.string.mobile_admin_orgBranding_branding_customDomainNone),
                fontFamily = FontFamily.Monospace,
                color = textSecondary(),
            )
            Text(
                L.text(context, localePrefs, R.string.mobile_admin_orgBranding_branding_customDomainHint),
                fontSize = 11.sp,
                color = textSecondary(),
            )
            TextButton(onClick = onOpenWeb) {
                Text(L.text(context, localePrefs, R.string.mobile_admin_orgBranding_webTitle))
            }

            Button(
                onClick = onSave,
                enabled = !saving && !uploading && colorsValid,
                modifier = Modifier.fillMaxWidth().height(48.dp),
            ) {
                Text(
                    if (saving) {
                        L.text(context, localePrefs, R.string.mobile_admin_orgBranding_saving)
                    } else {
                        L.text(context, localePrefs, R.string.mobile_admin_orgBranding_branding_save)
                    },
                )
            }
        }
    }
}

@Composable
private fun GovernanceCard(
    localePrefs: LocalePreferences,
    available: Boolean,
    enabledFeatures: MutableMap<String, Boolean>,
    allowedModelsText: String,
    onAllowedModelsChange: (String) -> Unit,
    saving: Boolean,
    onSave: () -> Unit,
) {
    val context = LocalContext.current
    LmsCard {
        Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
            Text(
                L.text(context, localePrefs, R.string.mobile_admin_orgBranding_ai_title),
                fontWeight = FontWeight.SemiBold,
                color = textPrimary(),
            )
            Text(
                L.text(context, localePrefs, R.string.mobile_admin_orgBranding_ai_intro),
                fontSize = 12.sp,
                color = textSecondary(),
            )
            if (!available) {
                Text(
                    L.text(context, localePrefs, R.string.mobile_admin_orgBranding_ai_unavailable),
                    color = textSecondary(),
                )
            } else {
                OrgBrandingAdminLogic.FEATURE_KEYS.forEach { feature ->
                    val resId = context.resources.getIdentifier(feature.labelResName, "string", context.packageName)
                    val label = if (resId != 0) {
                        L.text(context, localePrefs, resId)
                    } else {
                        feature.key
                    }
                    Row(
                        modifier = Modifier.fillMaxWidth().height(48.dp),
                        horizontalArrangement = Arrangement.SpaceBetween,
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Text(label, color = textPrimary(), modifier = Modifier.weight(1f))
                        Switch(
                            checked = OrgBrandingAdminLogic.isFeatureEnabled(enabledFeatures, feature.key),
                            onCheckedChange = { enabledFeatures[feature.key] = it },
                        )
                    }
                }
                OutlinedTextField(
                    value = allowedModelsText,
                    onValueChange = onAllowedModelsChange,
                    modifier = Modifier.fillMaxWidth().height(120.dp),
                    label = {
                        Text(L.text(context, localePrefs, R.string.mobile_admin_orgBranding_ai_allowedModels))
                    },
                )
                Text(
                    L.text(context, localePrefs, R.string.mobile_admin_orgBranding_ai_allowedModelsHint),
                    fontSize = 11.sp,
                    color = textSecondary(),
                )
                Button(
                    onClick = onSave,
                    enabled = !saving,
                    modifier = Modifier.fillMaxWidth().height(48.dp),
                ) {
                    Text(
                        if (saving) {
                            L.text(context, localePrefs, R.string.mobile_admin_orgBranding_saving)
                        } else {
                            L.text(context, localePrefs, R.string.mobile_admin_orgBranding_ai_save)
                        },
                    )
                }
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun ProviderCard(
    localePrefs: LocalePreferences,
    available: Boolean,
    settings: AIProviderSettings?,
    provider: String,
    onProviderChange: (String) -> Unit,
    modelAlias: String,
    onModelAliasChange: (String) -> Unit,
    fallbackProvider: String,
    onFallbackChange: (String) -> Unit,
    byokKey: String,
    onByokChange: (String) -> Unit,
    byokConfigured: Boolean,
    saving: Boolean,
    testing: Boolean,
    onSave: () -> Unit,
    onTest: () -> Unit,
) {
    val context = LocalContext.current
    val providers = OrgBrandingAdminLogic.providerOptions(settings)
    val aliases = OrgBrandingAdminLogic.modelAliasOptions(settings)

    LmsCard {
        Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
            Text(
                L.text(context, localePrefs, R.string.mobile_admin_orgBranding_provider_title),
                fontWeight = FontWeight.SemiBold,
                color = textPrimary(),
            )
            Text(
                L.text(context, localePrefs, R.string.mobile_admin_orgBranding_provider_intro),
                fontSize = 12.sp,
                color = textSecondary(),
            )
            if (!available) {
                Text(
                    L.text(context, localePrefs, R.string.mobile_admin_orgBranding_provider_unavailable),
                    color = textSecondary(),
                )
            } else {
                DropdownField(
                    label = L.text(context, localePrefs, R.string.mobile_admin_orgBranding_provider_provider),
                    value = provider,
                    options = providers,
                    optionLabel = { p ->
                        val key = OrgBrandingAdminLogic.providerLabelKey(p)
                        if (key != null) {
                            val resId = context.resources.getIdentifier(key, "string", context.packageName)
                            if (resId != 0) L.text(context, localePrefs, resId) else p
                        } else {
                            p
                        }
                    },
                    onSelect = onProviderChange,
                )
                DropdownField(
                    label = L.text(context, localePrefs, R.string.mobile_admin_orgBranding_provider_modelAlias),
                    value = modelAlias,
                    options = aliases,
                    optionLabel = { it },
                    onSelect = onModelAliasChange,
                )
                DropdownField(
                    label = L.text(context, localePrefs, R.string.mobile_admin_orgBranding_provider_fallback),
                    value = fallbackProvider,
                    options = listOf("") + providers,
                    optionLabel = { p ->
                        if (p.isEmpty()) {
                            L.text(context, localePrefs, R.string.mobile_admin_orgBranding_provider_fallbackNone)
                        } else {
                            val key = OrgBrandingAdminLogic.providerLabelKey(p)
                            if (key != null) {
                                val resId = context.resources.getIdentifier(key, "string", context.packageName)
                                if (resId != 0) L.text(context, localePrefs, resId) else p
                            } else {
                                p
                            }
                        }
                    },
                    onSelect = onFallbackChange,
                )
                OutlinedTextField(
                    value = byokKey,
                    onValueChange = onByokChange,
                    modifier = Modifier.fillMaxWidth(),
                    label = {
                        Text(L.text(context, localePrefs, R.string.mobile_admin_orgBranding_provider_byok))
                    },
                    visualTransformation = PasswordVisualTransformation(),
                    singleLine = true,
                    placeholder = { Text(OrgBrandingAdminLogic.SECRET_PLACEHOLDER) },
                )
                Text(
                    if (byokConfigured) {
                        L.text(context, localePrefs, R.string.mobile_admin_orgBranding_provider_byokConfigured)
                    } else {
                        L.text(context, localePrefs, R.string.mobile_admin_orgBranding_provider_byokHint)
                    },
                    fontSize = 11.sp,
                    color = if (byokConfigured) LexturesColors.BrandTeal else textSecondary(),
                )
                Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                    Button(
                        onClick = onSave,
                        enabled = !saving && !testing,
                        modifier = Modifier.weight(1f).height(48.dp),
                    ) {
                        Text(
                            if (saving) {
                                L.text(context, localePrefs, R.string.mobile_admin_orgBranding_saving)
                            } else {
                                L.text(context, localePrefs, R.string.mobile_admin_orgBranding_provider_save)
                            },
                        )
                    }
                    OutlinedButton(
                        onClick = onTest,
                        enabled = !saving && !testing,
                        modifier = Modifier.weight(1f).height(48.dp),
                    ) {
                        Text(
                            if (testing) {
                                L.text(context, localePrefs, R.string.mobile_admin_orgBranding_provider_testing)
                            } else {
                                L.text(context, localePrefs, R.string.mobile_admin_orgBranding_provider_test)
                            },
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun ColorField(
    label: String,
    value: String,
    onValueChange: (String) -> Unit,
) {
    Row(
        horizontalArrangement = Arrangement.spacedBy(10.dp),
        verticalAlignment = Alignment.CenterVertically,
        modifier = Modifier.fillMaxWidth(),
    ) {
        Box(
            modifier = Modifier
                .size(44.dp)
                .clip(RoundedCornerShape(8.dp))
                .background(parseHexColor(value) ?: Color.Gray.copy(alpha = 0.3f))
                .border(1.dp, textSecondary().copy(alpha = 0.25f), RoundedCornerShape(8.dp)),
        )
        OutlinedTextField(
            value = value,
            onValueChange = onValueChange,
            modifier = Modifier.weight(1f),
            label = { Text(label) },
            singleLine = true,
            textStyle = androidx.compose.ui.text.TextStyle(fontFamily = FontFamily.Monospace),
        )
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun DropdownField(
    label: String,
    value: String,
    options: List<String>,
    optionLabel: (String) -> String,
    onSelect: (String) -> Unit,
) {
    var expanded by remember { mutableStateOf(false) }
    ExposedDropdownMenuBox(expanded = expanded, onExpandedChange = { expanded = it }) {
        OutlinedTextField(
            value = optionLabel(value),
            onValueChange = {},
            readOnly = true,
            label = { Text(label) },
            trailingIcon = { ExposedDropdownMenuDefaults.TrailingIcon(expanded = expanded) },
            modifier = Modifier
                .menuAnchor()
                .fillMaxWidth(),
        )
        ExposedDropdownMenu(expanded = expanded, onDismissRequest = { expanded = false }) {
            options.forEach { option ->
                DropdownMenuItem(
                    text = { Text(optionLabel(option)) },
                    onClick = {
                        onSelect(option)
                        expanded = false
                    },
                )
            }
        }
    }
}

private fun parseHexColor(hex: String): Color? {
    val normalized = OrgBrandingAdminLogic.normalizeHexColor(hex, "")
    if (!normalized.startsWith("#") || normalized.length != 7) return null
    val raw = normalized.drop(1).toLongOrNull(16) ?: return null
    return Color(
        red = ((raw shr 16) and 0xFF) / 255f,
        green = ((raw shr 8) and 0xFF) / 255f,
        blue = (raw and 0xFF) / 255f,
    )
}

private fun resolveAssetUrl(pathOrUrl: String?): String? {
    val s = pathOrUrl?.trim().orEmpty()
    if (s.isEmpty()) return null
    if (s.startsWith("http://") || s.startsWith("https://")) return s
    val path = if (s.startsWith("/")) s else "/$s"
    return AppConfiguration.apiUrl(path).toString()
}
