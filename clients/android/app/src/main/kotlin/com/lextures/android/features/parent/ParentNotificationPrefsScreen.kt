package com.lextures.android.features.parent

import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.Remove
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.ui.Alignment
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.accentColor
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalLocalePreferences
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.ParentNotificationPrefs
import com.lextures.android.core.lms.PatchParentNotificationPrefsBody
import com.lextures.android.features.home.LmsSkeletonList
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ParentNotificationPrefsScreen(
    session: AuthSession,
    onBack: () -> Unit,
) {
    val accessToken by session.accessToken.collectAsState()
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    val scope = rememberCoroutineScope()

    var gradePosted by remember { mutableStateOf(true) }
    var missingAssignment by remember { mutableStateOf(true) }
    var attendanceEvent by remember { mutableStateOf(false) }
    var lowGradeEnabled by remember { mutableStateOf(true) }
    var lowGradeThreshold by remember { mutableIntStateOf(70) }
    var loading by remember { mutableStateOf(true) }
    var saving by remember { mutableStateOf(false) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var savedMessage by remember { mutableStateOf<String?>(null) }

    LaunchedEffect(accessToken) {
        val token = accessToken ?: return@LaunchedEffect
        loading = true
        try {
            val prefs = LmsApi.fetchParentNotificationPrefs(token)
            gradePosted = prefs.gradePosted
            missingAssignment = prefs.missingAssignment
            attendanceEvent = prefs.attendanceEvent
            lowGradeEnabled = prefs.lowGradeThreshold != null
            lowGradeThreshold = prefs.lowGradeThreshold ?: 70
        } catch (e: Exception) {
            errorMessage = e.message
        } finally {
            loading = false
        }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text(L.text(context, localePrefs, R.string.mobile_parent_notificationPrefs)) },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = null)
                    }
                },
            )
        },
    ) { padding ->
        if (loading) {
            LmsSkeletonList(count = 3, modifier = Modifier.padding(padding).fillMaxSize())
        } else {
            Column(Modifier.padding(padding).padding(16.dp)) {
                errorMessage?.let { Text(it, color = androidx.compose.ui.graphics.Color.Red) }
                savedMessage?.let { Text(it, color = accentColor()) }
                PrefRow(L.text(context, localePrefs, R.string.mobile_parent_prefs_gradePosted), gradePosted) {
                    gradePosted = it
                }
                PrefRow(L.text(context, localePrefs, R.string.mobile_parent_prefs_missingAssignment), missingAssignment) {
                    missingAssignment = it
                }
                PrefRow(L.text(context, localePrefs, R.string.mobile_parent_prefs_attendanceEvent), attendanceEvent) {
                    attendanceEvent = it
                }
                PrefRow(L.text(context, localePrefs, R.string.mobile_parent_prefs_lowGradeEnabled), lowGradeEnabled) {
                    lowGradeEnabled = it
                }
                if (lowGradeEnabled) {
                    Text(
                        localePrefs.localizedContext(context).getString(
                            R.string.mobile_parent_prefs_lowGradeValue,
                            lowGradeThreshold,
                        ),
                    )
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        IconButton(
                            onClick = { if (lowGradeThreshold > 0) lowGradeThreshold -= 5 },
                        ) {
                            Icon(
                                Icons.Filled.Remove,
                                contentDescription = L.text(
                                    context,
                                    localePrefs,
                                    R.string.mobile_parent_prefs_decreaseThreshold,
                                ),
                            )
                        }
                        IconButton(
                            onClick = { if (lowGradeThreshold < 100) lowGradeThreshold += 5 },
                        ) {
                            Icon(
                                Icons.Filled.Add,
                                contentDescription = L.text(
                                    context,
                                    localePrefs,
                                    R.string.mobile_parent_prefs_increaseThreshold,
                                ),
                            )
                        }
                    }
                }
                Button(
                    onClick = {
                        val token = accessToken ?: return@Button
                        scope.launch {
                            saving = true
                            errorMessage = null
                            savedMessage = null
                            try {
                                LmsApi.patchParentNotificationPrefs(
                                    PatchParentNotificationPrefsBody(
                                        gradePosted = gradePosted,
                                        missingAssignment = missingAssignment,
                                        lowGradeThreshold = if (lowGradeEnabled) lowGradeThreshold else null,
                                        clearThreshold = !lowGradeEnabled,
                                        attendanceEvent = attendanceEvent,
                                    ),
                                    token,
                                )
                                savedMessage = L.text(context, localePrefs, R.string.mobile_parent_prefs_saved)
                            } catch (e: Exception) {
                                errorMessage = e.message
                            } finally {
                                saving = false
                            }
                        }
                    },
                    enabled = !saving,
                ) {
                    if (saving) CircularProgressIndicator() else Text(L.text(context, localePrefs, R.string.mobile_parent_prefs_save))
                }
            }
        }
    }
}

@Composable
private fun PrefRow(label: String, checked: Boolean, onCheckedChange: (Boolean) -> Unit) {
    Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
        Text(label, modifier = Modifier.weight(1f))
        Switch(checked = checked, onCheckedChange = onCheckedChange)
    }
}
