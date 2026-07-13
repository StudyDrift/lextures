package com.lextures.android.features.settings.admin

import android.content.Intent
import android.net.Uri
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
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
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import coil.compose.AsyncImage
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.config.AppConfiguration
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalePreferences
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.OrgBrandingAdminLogic
import com.lextures.android.core.lms.OrgBrandingResponse
import com.lextures.android.core.lms.PutOrgBrandingRequest
import com.lextures.android.features.home.LmsCard
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

@Composable
fun OrgBrandingScreen(
    session: AuthSession,
    localePrefs: LocalePreferences,
    orgId: String,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val accessToken by session.accessToken.collectAsState()

    var branding by remember { mutableStateOf<OrgBrandingResponse?>(null) }
    var primaryColor by remember { mutableStateOf(OrgBrandingAdminLogic.DEFAULT_PRIMARY_COLOR) }
    var secondaryColor by remember { mutableStateOf(OrgBrandingAdminLogic.DEFAULT_SECONDARY_COLOR) }
    var emailDisplayName by remember { mutableStateOf("") }
    var previewLogoUrl by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }
    var saving by remember { mutableStateOf(false) }
    var uploading by remember { mutableStateOf(false) }
    var statusMessage by remember { mutableStateOf<String?>(null) }
    var errorMessage by remember { mutableStateOf<String?>(null) }

    val pickLogo = rememberLauncherForActivityResult(ActivityResultContracts.GetContent()) { uri ->
        val token = accessToken ?: return@rememberLauncherForActivityResult
        uri ?: return@rememberLauncherForActivityResult
        scope.launch {
            uploading = true
            errorMessage = null
            try {
                val bytes = withContext(Dispatchers.IO) {
                    context.contentResolver.openInputStream(uri)?.use { it.readBytes() }
                } ?: return@launch
                if (bytes.size > OrgBrandingAdminLogic.MAX_LOGO_UPLOAD_BYTES) {
                    errorMessage = L.text(context, localePrefs, R.string.mobile_admin_orgBranding_branding_fileTooLarge)
                    return@launch
                }
                val upload = LmsApi.uploadOrgBrandingLogo(
                    orgId = orgId,
                    fileName = "logo.jpg",
                    mimeType = "image/jpeg",
                    fileBytes = bytes,
                    accessToken = token,
                )
                upload.url?.let { url ->
                    branding = branding?.copy(logoUrl = url)
                    previewLogoUrl = OrgBrandingAdminLogic.resolveBrandAssetUrl(url)
                }
                statusMessage = L.text(context, localePrefs, R.string.mobile_admin_orgBranding_branding_saved)
            } catch (error: Throwable) {
                errorMessage = OrgBrandingAdminLogic.userFacingError(
                    error,
                    L.text(context, localePrefs, R.string.mobile_admin_orgBranding_branding_uploadError),
                )
            } finally {
                uploading = false
            }
        }
    }

    LaunchedEffect(orgId, accessToken) {
        val token = accessToken ?: return@LaunchedEffect
        loading = true
        errorMessage = null
        try {
            val response = LmsApi.fetchOrgBranding(orgId, token)
            branding = response
            primaryColor = response.primaryColor
            secondaryColor = response.secondaryColor
            emailDisplayName = response.customEmailDisplayName.orEmpty()
            previewLogoUrl = OrgBrandingAdminLogic.resolveBrandAssetUrl(response.logoUrl)
        } catch (error: Throwable) {
            errorMessage = OrgBrandingAdminLogic.userFacingError(
                error,
                L.text(context, localePrefs, R.string.mobile_admin_orgBranding_branding_loadError),
            )
        } finally {
            loading = false
        }
    }

    LmsCard(modifier = modifier) {
        Column(
            modifier = Modifier.padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Text(
                text = L.text(context, localePrefs, R.string.mobile_admin_orgBranding_branding_title),
                fontWeight = FontWeight.SemiBold,
                color = textPrimary(),
            )

            if (loading) {
                CircularProgressIndicator(modifier = Modifier.align(Alignment.CenterHorizontally))
            } else {
                Row(horizontalArrangement = Arrangement.spacedBy(12.dp), verticalAlignment = Alignment.CenterVertically) {
                    previewLogoUrl?.let { url ->
                        AsyncImage(
                            model = url,
                            contentDescription = null,
                            modifier = Modifier.size(72.dp),
                        )
                    }
                    OutlinedButton(
                        onClick = { pickLogo.launch("image/*") },
                        enabled = !uploading && !saving,
                    ) {
                        Text(
                            if (uploading) {
                                L.text(context, localePrefs, R.string.mobile_admin_orgBranding_branding_uploading)
                            } else {
                                L.text(context, localePrefs, R.string.mobile_admin_orgBranding_branding_uploadLogo)
                            },
                        )
                    }
                }
                Text(
                    text = L.text(context, localePrefs, R.string.mobile_admin_orgBranding_branding_logoHint),
                    fontSize = 12.sp,
                    color = textSecondary(),
                )

                OutlinedTextField(
                    value = primaryColor,
                    onValueChange = { primaryColor = it },
                    label = { Text(L.text(context, localePrefs, R.string.mobile_admin_orgBranding_branding_primaryColor)) },
                    modifier = Modifier.fillMaxWidth(),
                    textStyle = androidx.compose.ui.text.TextStyle(fontFamily = FontFamily.Monospace),
                )
                OutlinedTextField(
                    value = secondaryColor,
                    onValueChange = { secondaryColor = it },
                    label = { Text(L.text(context, localePrefs, R.string.mobile_admin_orgBranding_branding_secondaryColor)) },
                    modifier = Modifier.fillMaxWidth(),
                    textStyle = androidx.compose.ui.text.TextStyle(fontFamily = FontFamily.Monospace),
                )

                if (OrgBrandingAdminLogic.showsContrastWarning(
                        primaryColor,
                        branding?.contrastWarningPrimary == true,
                        branding?.contrastRatioPrimary,
                    )
                ) {
                    Text(
                        text = L.text(context, localePrefs, R.string.mobile_admin_orgBranding_branding_contrastWarning),
                        fontSize = 12.sp,
                        color = androidx.compose.ui.graphics.Color(0xFFD97706),
                    )
                }

                OutlinedTextField(
                    value = emailDisplayName,
                    onValueChange = { emailDisplayName = it },
                    label = { Text(L.text(context, localePrefs, R.string.mobile_admin_orgBranding_branding_emailSender)) },
                    placeholder = {
                        Text(L.text(context, localePrefs, R.string.mobile_admin_orgBranding_branding_emailSenderPlaceholder))
                    },
                    modifier = Modifier.fillMaxWidth(),
                )
                Text(
                    text = L.text(context, localePrefs, R.string.mobile_admin_orgBranding_branding_emailSenderHint),
                    fontSize = 12.sp,
                    color = textSecondary(),
                )

                Text(
                    text = L.text(context, localePrefs, R.string.mobile_admin_orgBranding_branding_customDomain),
                    fontWeight = FontWeight.Medium,
                )
                Text(
                    text = branding?.customDomain?.takeIf { it.isNotBlank() }
                        ?: L.text(context, localePrefs, R.string.mobile_admin_orgBranding_branding_customDomainNone),
                    fontSize = 14.sp,
                    color = textSecondary(),
                )
                OutlinedButton(
                    onClick = {
                        val intent = Intent(
                            Intent.ACTION_VIEW,
                            Uri.parse(AppConfiguration.webUrl(OrgBrandingAdminLogic.webOrgBrandingPath()).toString()),
                        )
                        context.startActivity(intent)
                    },
                ) {
                    Text(L.text(context, localePrefs, R.string.mobile_admin_orgBranding_branding_configureOnWeb))
                }

                Column(
                    modifier = Modifier
                        .fillMaxWidth()
                        .clip(RoundedCornerShape(12.dp))
                        .background(androidx.compose.ui.graphics.Color(0xFFF8FAFC))
                        .padding(16.dp),
                    horizontalAlignment = Alignment.CenterHorizontally,
                    verticalArrangement = Arrangement.spacedBy(8.dp),
                ) {
                    Text(
                        text = L.text(context, localePrefs, R.string.mobile_admin_orgBranding_branding_preview),
                        fontWeight = FontWeight.Medium,
                    )
                    previewLogoUrl?.let { url ->
                        AsyncImage(model = url, contentDescription = null, modifier = Modifier.height(64.dp))
                    }
                    Box(
                        modifier = Modifier
                            .fillMaxWidth()
                            .height(8.dp)
                            .clip(RoundedCornerShape(4.dp))
                            .background(OrgBrandingAdminLogic.colorFromHex(primaryColor)),
                    )
                    Text(L.text(context, localePrefs, R.string.mobile_admin_orgBranding_branding_previewSignIn))
                    Button(onClick = {}, enabled = false) {
                        Text(L.text(context, localePrefs, R.string.mobile_admin_orgBranding_branding_previewContinue))
                    }
                }

                Button(
                    onClick = {
                        val token = accessToken ?: return@Button
                        if (!OrgBrandingAdminLogic.isValidHexColor(primaryColor) ||
                            !OrgBrandingAdminLogic.isValidHexColor(secondaryColor)
                        ) {
                            return@Button
                        }
                        scope.launch {
                            saving = true
                            errorMessage = null
                            statusMessage = null
                            try {
                                val trimmedEmail = emailDisplayName.trim()
                                val request = PutOrgBrandingRequest(
                                    logoUrl = branding?.logoUrl,
                                    faviconUrl = branding?.faviconUrl,
                                    primaryColor = OrgBrandingAdminLogic.normalizedHexColor(primaryColor) ?: primaryColor,
                                    secondaryColor = OrgBrandingAdminLogic.normalizedHexColor(secondaryColor) ?: secondaryColor,
                                    customDomain = branding?.customDomain,
                                    customEmailDisplayName = trimmedEmail.takeIf { it.isNotEmpty() },
                                )
                                val response = LmsApi.putOrgBranding(orgId, request, token)
                                branding = response
                                primaryColor = response.primaryColor
                                secondaryColor = response.secondaryColor
                                emailDisplayName = response.customEmailDisplayName.orEmpty()
                                previewLogoUrl = OrgBrandingAdminLogic.resolveBrandAssetUrl(response.logoUrl)
                                statusMessage = L.text(context, localePrefs, R.string.mobile_admin_orgBranding_branding_saved)
                            } catch (error: Throwable) {
                                errorMessage = OrgBrandingAdminLogic.userFacingError(
                                    error,
                                    L.text(context, localePrefs, R.string.mobile_admin_orgBranding_branding_saveError),
                                )
                            } finally {
                                saving = false
                            }
                        }
                    },
                    enabled = !saving &&
                        OrgBrandingAdminLogic.isValidHexColor(primaryColor) &&
                        OrgBrandingAdminLogic.isValidHexColor(secondaryColor),
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    if (saving) {
                        CircularProgressIndicator(modifier = Modifier.size(18.dp))
                    } else {
                        Text(L.text(context, localePrefs, R.string.mobile_admin_orgBranding_branding_save))
                    }
                }

                statusMessage?.let {
                    Text(text = it, fontSize = 12.sp, color = androidx.compose.ui.graphics.Color(0xFF059669))
                }
                errorMessage?.let {
                    Text(text = it, fontSize = 12.sp, color = androidx.compose.ui.graphics.Color(0xFFDC2626))
                }
            }
        }
    }
}
