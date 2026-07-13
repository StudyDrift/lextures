package com.lextures.android.features.settings.admin

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
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
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalePreferences
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.OrgBrandingAdminLogic
import com.lextures.android.core.network.ApiError
import com.lextures.android.features.home.LmsCard
import kotlinx.coroutines.launch

@Composable
fun AiGovernanceScreen(
    session: AuthSession,
    localePrefs: LocalePreferences,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val accessToken by session.accessToken.collectAsState()

    var available by remember { mutableStateOf(false) }
    var enabled by remember { mutableStateOf<Map<String, Boolean>>(emptyMap()) }
    var allowedModelsText by remember { mutableStateOf("") }
    var loading by remember { mutableStateOf(true) }
    var saving by remember { mutableStateOf(false) }
    var statusMessage by remember { mutableStateOf<String?>(null) }
    var errorMessage by remember { mutableStateOf<String?>(null) }

    LaunchedEffect(accessToken) {
        val token = accessToken ?: return@LaunchedEffect
        loading = true
        try {
            val response = LmsApi.fetchAiConfig(token)
            available = true
            enabled = response.featuresEnabled.orEmpty()
            allowedModelsText = OrgBrandingAdminLogic.allowedModelsText(response.allowedModels)
        } catch (error: Throwable) {
            if (error is ApiError.HttpStatus && (error.code == 404 || error.code == 403)) {
                available = false
            } else {
                available = true
                errorMessage = OrgBrandingAdminLogic.userFacingError(
                    error,
                    L.text(context, localePrefs, R.string.mobile_admin_orgBranding_aiGovernance_loadError),
                )
            }
        } finally {
            loading = false
        }
    }

    if (!available) return

    LmsCard(modifier = modifier) {
        Column(
            modifier = Modifier.padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Text(
                text = L.text(context, localePrefs, R.string.mobile_admin_orgBranding_aiGovernance_title),
                fontWeight = FontWeight.SemiBold,
                color = textPrimary(),
            )
            Text(
                text = L.text(context, localePrefs, R.string.mobile_admin_orgBranding_aiGovernance_intro),
                fontSize = 14.sp,
                color = textSecondary(),
            )

            if (loading) {
                CircularProgressIndicator(modifier = Modifier.align(androidx.compose.ui.Alignment.CenterHorizontally))
            } else {
                OrgBrandingAdminLogic.AI_FEATURE_KEYS.forEach { feature ->
                    val labelRes = when (feature.labelResSuffix) {
                        "aiTutor" -> R.string.mobile_admin_orgBranding_aiGovernance_feature_aiTutor
                        "notebook" -> R.string.mobile_admin_orgBranding_aiGovernance_feature_notebook
                        "syllabus" -> R.string.mobile_admin_orgBranding_aiGovernance_feature_syllabus
                        "translation" -> R.string.mobile_admin_orgBranding_aiGovernance_feature_translation
                        "quiz" -> R.string.mobile_admin_orgBranding_aiGovernance_feature_quiz
                        "lesson" -> R.string.mobile_admin_orgBranding_aiGovernance_feature_lesson
                        else -> R.string.mobile_admin_orgBranding_aiGovernance_feature_aiTutor
                    }
                    RowToggle(
                        label = L.text(context, localePrefs, labelRes),
                        checked = enabled[feature.key] != false,
                        onCheckedChange = { checked ->
                            enabled = enabled + (feature.key to checked)
                        },
                    )
                }

                OutlinedTextField(
                    value = allowedModelsText,
                    onValueChange = { allowedModelsText = it },
                    label = { Text(L.text(context, localePrefs, R.string.mobile_admin_orgBranding_aiGovernance_allowedModels)) },
                    modifier = Modifier.fillMaxWidth(),
                    minLines = 4,
                    textStyle = androidx.compose.ui.text.TextStyle(fontFamily = FontFamily.Monospace, fontSize = 12.sp),
                )
                Text(
                    text = L.text(context, localePrefs, R.string.mobile_admin_orgBranding_aiGovernance_allowedModelsHint),
                    fontSize = 12.sp,
                    color = textSecondary(),
                )

                Button(
                    onClick = {
                        val token = accessToken ?: return@Button
                        scope.launch {
                            saving = true
                            errorMessage = null
                            statusMessage = null
                            try {
                                val request = OrgBrandingAdminLogic.buildAiConfigSaveRequest(enabled, allowedModelsText)
                                val response = LmsApi.putAiConfig(request, token)
                                enabled = response.featuresEnabled ?: enabled
                                allowedModelsText = OrgBrandingAdminLogic.allowedModelsText(response.allowedModels)
                                statusMessage = L.text(context, localePrefs, R.string.mobile_admin_orgBranding_aiGovernance_saved)
                            } catch (error: Throwable) {
                                errorMessage = OrgBrandingAdminLogic.userFacingError(
                                    error,
                                    L.text(context, localePrefs, R.string.mobile_admin_orgBranding_aiGovernance_saveError),
                                )
                            } finally {
                                saving = false
                            }
                        }
                    },
                    enabled = !saving,
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    if (saving) {
                        CircularProgressIndicator(modifier = Modifier.size(18.dp))
                    } else {
                        Text(L.text(context, localePrefs, R.string.mobile_admin_orgBranding_aiGovernance_save))
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

@Composable
private fun RowToggle(
    label: String,
    checked: Boolean,
    onCheckedChange: (Boolean) -> Unit,
) {
    androidx.compose.foundation.layout.Row(
        modifier = Modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = androidx.compose.ui.Alignment.CenterVertically,
    ) {
        Text(text = label, modifier = Modifier.weight(1f))
        Switch(checked = checked, onCheckedChange = onCheckedChange)
    }
}
