package com.lextures.android.features.home

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Search
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import com.lextures.android.R
import com.lextures.android.core.design.sceneBackground
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalLocalePreferences

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun UniversalSearchPlaceholder(
    onDismiss: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    Scaffold(
        modifier = modifier,
        topBar = {
            TopAppBar(
                title = { Text(L.text(context, localePrefs, R.string.mobile_ia_search)) },
                navigationIcon = {
                    TextButton(onClick = onDismiss) {
                        Text(L.text(context, localePrefs, R.string.mobile_ia_close))
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
                icon = Icons.Default.Search,
                title = L.text(context, localePrefs, R.string.mobile_ia_search_title),
                message = L.text(context, localePrefs, R.string.mobile_ia_search_comingSoon),
            )
        }
    }
}