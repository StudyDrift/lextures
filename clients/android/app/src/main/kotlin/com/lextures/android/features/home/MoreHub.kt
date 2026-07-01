package com.lextures.android.features.home

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Apps
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalLocalePreferences
import com.lextures.android.core.navigation.MobileDestinations
import com.lextures.android.core.navigation.MoreDestination

@Composable
fun MoreHubScreen(
    shell: HomeShellState,
    onOpenDestination: (MoreDestination) -> Unit,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    var query by remember { mutableStateOf("") }
    val destinations = remember(shell.activeRoleContext, shell.platformFeatures) {
        MobileDestinations.moreDestinations(shell.activeRoleContext, shell.platformFeatures)
    }
    val filtered = remember(destinations, query) {
        val q = query.trim().lowercase()
        if (q.isEmpty()) {
            destinations
        } else {
            destinations.filter {
                L.text(context, localePrefs, moreLabelRes(it)).lowercase().contains(q)
            }
        }
    }

    Column(modifier = modifier.fillMaxSize().padding(16.dp)) {
        if (destinations.isEmpty()) {
            LmsEmptyState(
                icon = Icons.Default.Apps,
                title = L.text(context, localePrefs, R.string.mobile_ia_more_emptyTitle),
                message = L.text(context, localePrefs, R.string.mobile_ia_more_emptyMessage),
            )
        } else {
            LazyVerticalGrid(
                columns = GridCells.Fixed(2),
                contentPadding = PaddingValues(0.dp),
                horizontalArrangement = Arrangement.spacedBy(12.dp),
                verticalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                items(filtered, key = { it.name }) { destination ->
                    LmsCard(
                        modifier = Modifier
                            .fillMaxWidth()
                            .clickable { onOpenDestination(destination) },
                    ) {
                        Text(
                            text = L.text(context, localePrefs, moreLabelRes(destination)),
                            fontSize = 14.sp,
                            fontWeight = FontWeight.SemiBold,
                            color = textPrimary(),
                        )
                    }
                }
            }
        }
    }
}

@Composable
fun MoreDestinationPlaceholder(
    destination: MoreDestination,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    LmsEmptyState(
        icon = Icons.Default.Apps,
        title = L.text(context, localePrefs, moreLabelRes(destination)),
        message = L.text(context, localePrefs, R.string.mobile_ia_placeholder_message),
        modifier = modifier.fillMaxSize(),
    )
}

private fun moreLabelRes(destination: MoreDestination): Int = when (destination) {
    MoreDestination.Calendar -> R.string.mobile_ia_more_calendar
    MoreDestination.Planner -> R.string.mobile_ia_more_planner
    MoreDestination.Catalog -> R.string.mobile_ia_more_catalog
    MoreDestination.Paths -> R.string.mobile_ia_more_paths
    MoreDestination.Library -> R.string.mobile_ia_more_library
    MoreDestination.Reading -> R.string.mobile_ia_more_reading
    MoreDestination.Portfolio -> R.string.mobile_ia_more_portfolio
    MoreDestination.Credentials -> R.string.mobile_ia_more_credentials
    MoreDestination.Advising -> R.string.mobile_ia_more_advising
    MoreDestination.Settings -> R.string.mobile_ia_more_settings
    MoreDestination.AskAi -> R.string.mobile_tutor_askAi
}