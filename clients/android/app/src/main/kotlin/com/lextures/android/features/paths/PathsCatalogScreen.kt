package com.lextures.android.features.paths

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Route
import androidx.compose.material.icons.filled.Search
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
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
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalLocalePreferences
import com.lextures.android.core.lms.CatalogPathSummary
import com.lextures.android.core.lms.CatalogPathsListResponse
import com.lextures.android.core.lms.LmsApi
import kotlinx.serialization.serializer
import com.lextures.android.core.lms.PathsLogic
import com.lextures.android.core.offline.OfflineCacheKey
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsSkeletonList

@Composable
fun PathsCatalogScreen(
    session: AuthSession,
    onOpenPath: (String) -> Unit,
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    val offline = remember { OfflineService.get(context) }

    var query by remember { mutableStateOf("") }
    var paths by remember { mutableStateOf<List<CatalogPathSummary>>(emptyList()) }
    var loading by remember { mutableStateOf(false) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var hasSearched by remember { mutableStateOf(false) }

    LaunchedEffect(accessToken, query) {
        loading = true
        errorMessage = null
        hasSearched = true
        try {
            val token = accessToken
            paths = if (token != null) {
                offline.cachedFetch(
                    key = OfflineCacheKey.catalogPaths(query.trim()),
                    accessToken = token,
                    serializer = serializer<CatalogPathsListResponse>(),
                ) {
                    CatalogPathsListResponse(LmsApi.fetchCatalogPaths(query, accessToken = token))
                }.first.paths
            } else {
                LmsApi.fetchCatalogPaths(query, accessToken = null)
            }
        } catch (e: Exception) {
            errorMessage = L.text(context, localePrefs, R.string.mobile_paths_error_catalog)
            paths = emptyList()
        } finally {
            loading = false
        }
    }

    Column(modifier = modifier.fillMaxSize().padding(16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
        OutlinedTextField(
            value = query,
            onValueChange = { query = it },
            modifier = Modifier.fillMaxWidth(),
            placeholder = { Text(L.text(context, localePrefs, R.string.mobile_paths_catalogSearch)) },
            leadingIcon = { androidx.compose.material3.Icon(Icons.Default.Search, contentDescription = null) },
            singleLine = true,
        )

        when {
            loading && paths.isEmpty() -> LmsSkeletonList(count = 4)
            paths.isEmpty() -> LmsEmptyState(
                icon = if (errorMessage != null) Icons.Default.Route else Icons.Default.Search,
                title = L.text(
                    context,
                    localePrefs,
                    if (errorMessage != null) R.string.mobile_paths_catalogTitle else R.string.mobile_paths_catalogEmptyTitle,
                ),
                message = errorMessage ?: L.text(
                    context,
                    localePrefs,
                    if (hasSearched) R.string.mobile_paths_catalogEmptyMessage else R.string.mobile_paths_catalogPrompt,
                ),
            )
            else -> LazyColumn(verticalArrangement = Arrangement.spacedBy(12.dp)) {
                items(paths, key = { it.id }) { path ->
                    LmsCard(onClick = { onOpenPath(path.slug) }) {
                        Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
                            Text(path.title, fontSize = 15.sp, fontWeight = FontWeight.SemiBold, color = textPrimary())
                            if (path.description.isNotBlank()) {
                                Text(path.description, fontSize = 12.sp, color = textSecondary(), maxLines = 2)
                            }
                            val duration = PathsLogic.formatDuration(path.totalDurationMinutes)
                            val price = if (PathsLogic.isPaid(path.bundlePriceCents)) {
                                PathsLogic.formatPrice(path.bundlePriceCents ?: 0)
                            } else {
                                L.text(context, localePrefs, R.string.mobile_paths_free)
                            }
                            Text(
                                context.getString(R.string.mobile_paths_catalogMeta, path.courseCount, duration, price),
                                fontSize = 11.sp,
                                color = textSecondary(),
                            )
                        }
                    }
                }
            }
        }
    }
}