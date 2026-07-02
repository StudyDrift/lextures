package com.lextures.android.features.paths

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
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
import com.lextures.android.core.design.accentColor
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalLocalePreferences
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.MyPathsListResponse
import com.lextures.android.core.lms.PathProgress
import com.lextures.android.core.lms.PathsLogic
import kotlinx.serialization.serializer
import com.lextures.android.core.offline.OfflineCacheKey
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsSkeletonList
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Route

@Composable
fun MyPathsScreen(
    session: AuthSession,
    onOpenPath: (PathProgress) -> Unit,
    onBrowseCatalog: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    val offline = remember { OfflineService.get(context) }

    var paths by remember { mutableStateOf<List<PathProgress>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var errorMessage by remember { mutableStateOf<String?>(null) }

    LaunchedEffect(accessToken) {
        val token = accessToken ?: return@LaunchedEffect
        loading = true
        errorMessage = null
        try {
            paths = offline.cachedFetch(
                key = OfflineCacheKey.myPaths(),
                accessToken = token,
                serializer = serializer<MyPathsListResponse>(),
            ) { MyPathsListResponse(LmsApi.fetchMyPaths(token)) }.first.paths
        } catch (e: Exception) {
            errorMessage = L.text(context, localePrefs, R.string.mobile_paths_error_load)
            paths = emptyList()
        } finally {
            loading = false
        }
    }

    Column(modifier = modifier.fillMaxSize().padding(16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
        Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.End) {
            TextButton(onClick = onBrowseCatalog) {
                Text(L.text(context, localePrefs, R.string.mobile_paths_browse))
            }
        }

        when {
            loading && paths.isEmpty() -> LmsSkeletonList(count = 3)
            paths.isEmpty() -> LmsEmptyState(
                icon = Icons.Default.Route,
                title = L.text(
                    context,
                    localePrefs,
                    if (errorMessage != null) R.string.mobile_paths_title else R.string.mobile_paths_emptyTitle,
                ),
                message = errorMessage ?: L.text(context, localePrefs, R.string.mobile_paths_emptyMessage),
            )
            else -> LazyColumn(verticalArrangement = Arrangement.spacedBy(12.dp)) {
                items(paths, key = { it.pathId }) { path ->
                    LmsCard(onClick = { onOpenPath(path) }) {
                        Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                            Text(path.pathTitle, fontSize = 15.sp, fontWeight = FontWeight.SemiBold, color = textPrimary())
                            LinearProgressIndicator(
                                progress = { path.percent / 100f },
                                modifier = Modifier.fillMaxWidth(),
                            )
                            Text(path.progressLabel, fontSize = 12.sp, color = textSecondary())
                            PathsLogic.nextCourse(path)?.let { next ->
                                Text(
                                    context.getString(R.string.mobile_paths_continueCourse, next.title),
                                    fontSize = 12.sp,
                                    fontWeight = FontWeight.SemiBold,
                                    color = accentColor(),
                                )
                            }
                        }
                    }
                }
            }
        }
    }
}