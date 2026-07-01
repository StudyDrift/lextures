package com.lextures.android.features.home

import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.FamilyRestroom
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import com.lextures.android.R
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalLocalePreferences

@Composable
fun ChildrenPlaceholderScreen(modifier: Modifier = Modifier) {
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    LmsEmptyState(
        icon = Icons.Default.FamilyRestroom,
        title = L.text(context, localePrefs, R.string.mobile_ia_children_title),
        message = L.text(context, localePrefs, R.string.mobile_ia_children_message),
        modifier = modifier.fillMaxSize(),
    )
}