package com.lextures.android.features.courses

import android.content.Intent
import android.net.Uri
import androidx.activity.compose.BackHandler
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.OutlinedButton
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
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.config.AppConfiguration
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalLocalePreferences
import com.lextures.android.core.lms.CourseStructureItem
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.LibraryAccessState
import com.lextures.android.core.lms.LibraryResourceLogic
import com.lextures.android.core.lms.LibraryResourcePayload
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.library.libraryAccessRestricted
import com.lextures.android.features.library.libraryCatalogGated
import com.lextures.android.features.library.libraryLegantoGated
import com.lextures.android.features.library.libraryLargerScreenHint
import com.lextures.android.features.library.libraryNoAccess
import com.lextures.android.features.library.libraryNotFound
import com.lextures.android.features.library.libraryOpenOnWeb
import com.lextures.android.features.library.libraryOpenResource
import com.lextures.android.features.library.libraryReadyToOpen
import com.lextures.android.R
import kotlinx.coroutines.launch

@Composable
fun LibraryResourceScreen(
    session: AuthSession,
    course: CourseSummary,
    item: CourseStructureItem,
    nativeEnabled: Boolean = true,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    val offline = remember { OfflineService.get(context) }
    val scope = rememberCoroutineScope()

    var payload by remember { mutableStateOf<LibraryResourcePayload?>(null) }
    var accessState by remember { mutableStateOf<LibraryAccessState?>(null) }
    var openUrl by remember { mutableStateOf<String?>(null) }
    var loadError by remember { mutableStateOf<String?>(null) }
    var accessEventError by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }

    BackHandler(onBack = onBack)

    LaunchedEffect(accessToken, item.id) {
        val token = accessToken ?: return@LaunchedEffect
        loading = true
        try {
            val row = LmsApi.fetchModuleLibraryResource(course.courseCode, item.id, token)
            if (row == null) {
                loadError = L.text(context, localePrefs, R.string.mobile_library_notFound)
                return@LaunchedEffect
            }
            payload = row
            val state = LibraryResourceLogic.resolveAccess(row)
            accessState = state
            if (state is LibraryAccessState.Ready) {
                scope.launch {
                    runCatching {
                        offline.enqueueMutation(
                            method = "POST",
                            path = LibraryResourceLogic.accessEventPath(course.courseCode, item.id),
                            bodyJson = null,
                            label = L.text(context, localePrefs, R.string.mobile_library_accessEventLabel),
                            accessToken = token,
                        )
                    }.onFailure {
                        accessEventError = L.text(context, localePrefs, R.string.mobile_library_accessEventFailed)
                    }
                }
                openUrl = state.url
            }
        } catch (e: Exception) {
            loadError = session.mapError(e)
        } finally {
            loading = false
        }
    }

    val title = payload?.metadata?.title?.trim()?.takeIf { it.isNotEmpty() } ?: item.title

    Column(modifier = modifier) {
        RowHeader(title = title, onBack = onBack)
        when {
            openUrl != null -> WebItemScreen(
                title = title,
                urlString = openUrl!!,
                accessToken = accessToken,
                onOpenExternal = { uri -> context.startActivity(Intent(Intent.ACTION_VIEW, uri)) },
                modifier = Modifier.fillMaxSize(),
            )
            loading -> Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                CircularProgressIndicator(color = LexturesColors.Primary)
            }
            else -> Column(Modifier.padding(16.dp)) {
                loadError?.let { LmsErrorBanner(it) }
                accessEventError?.let { LmsErrorBanner(it) }
                payload?.let { row ->
                    LmsCard(Modifier.fillMaxWidth().padding(bottom = 12.dp)) {
                        Text(title, fontWeight = FontWeight.SemiBold, color = textPrimary())
                        Text(
                            L.text(context, localePrefs, resourceTypeRes(row.resourceType)),
                            fontSize = 12.sp,
                            color = LexturesColors.PrimaryMuted,
                        )
                        row.metadata?.author?.let {
                            Text(it, fontSize = 12.sp, color = textSecondary())
                        }
                    }
                    when (val state = accessState) {
                        is LibraryAccessState.Ready -> LmsCard(Modifier.fillMaxWidth()) {
                            Text(libraryReadyToOpen(), fontSize = 14.sp, color = textSecondary())
                            Button(onClick = { openUrl = state.url }) {
                                Text(libraryOpenResource())
                            }
                        }
                        is LibraryAccessState.Gated -> LmsCard(Modifier.fillMaxWidth()) {
                            Text(libraryAccessRestricted(), fontWeight = FontWeight.SemiBold, color = textPrimary())
                            Text(gatedMessage(state.messageKey), fontSize = 14.sp, color = textSecondary())
                            if (nativeEnabled) {
                                OutlinedButton(onClick = {
                                    val path = LibraryResourceLogic.webModulePath(course.courseCode, item.id)
                                    context.startActivity(
                                        Intent(Intent.ACTION_VIEW, Uri.parse(AppConfiguration.apiUrl(path).toString())),
                                    )
                                }) { Text(libraryOpenOnWeb()) }
                            }
                        }
                        is LibraryAccessState.RequiresWeb -> LmsCard(Modifier.fillMaxWidth()) {
                            Text(libraryLargerScreenHint(), fontSize = 14.sp, color = textSecondary())
                            OutlinedButton(onClick = {
                                context.startActivity(
                                    Intent(Intent.ACTION_VIEW, Uri.parse(AppConfiguration.apiUrl(state.path).toString())),
                                )
                            }) { Text(libraryOpenOnWeb()) }
                        }
                        null -> Unit
                    }
                }
            }
        }
    }
}

@Composable
private fun gatedMessage(messageKey: String): String = when (messageKey) {
    "mobile.library.legantoGated" -> libraryLegantoGated()
    "mobile.library.catalogGated" -> libraryCatalogGated()
    else -> libraryNoAccess()
}

private fun resourceTypeRes(resourceType: String): Int = when (resourceType) {
    "leganto_list" -> R.string.mobile_library_type_leganto
    "catalog_item" -> R.string.mobile_library_type_catalog
    else -> R.string.mobile_library_type_generic
}

