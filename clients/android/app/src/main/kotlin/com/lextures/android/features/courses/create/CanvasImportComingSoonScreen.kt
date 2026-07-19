package com.lextures.android.features.courses.create

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CloudDownload
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import com.lextures.android.R
import com.lextures.android.core.design.sceneBackground
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalLocalePreferences
import com.lextures.android.features.home.LmsEmptyState

/** MOB.1 FR-8 handoff placeholder until MOB.2 Canvas import ships. */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun CanvasImportComingSoonScreen(
    onDismiss: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    Scaffold(
        modifier = modifier,
        topBar = {
            TopAppBar(
                title = {
                    Text(L.text(context, localePrefs, R.string.mobile_createCourse_source_canvas_title))
                },
                navigationIcon = {
                    TextButton(onClick = onDismiss) {
                        Text(L.text(context, localePrefs, R.string.mobile_common_close))
                    }
                },
            )
        },
    ) { padding ->
        Box(
            modifier = Modifier
                .fillMaxSize()
                .background(sceneBackground())
                .padding(padding),
        ) {
            LmsEmptyState(
                icon = Icons.Default.CloudDownload,
                title = L.text(context, localePrefs, R.string.mobile_createCourse_canvas_comingSoon_title),
                message = L.text(context, localePrefs, R.string.mobile_createCourse_canvas_comingSoon_body),
            )
        }
    }
}
