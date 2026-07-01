package com.lextures.android.features.library

import android.content.Intent
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Search
import androidx.compose.material3.Button
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
import kotlinx.coroutines.launch
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.ui.window.Dialog
import androidx.compose.ui.window.DialogProperties
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.lms.LibraryBrowseTab
import com.lextures.android.core.lms.LibraryCatalogResult
import com.lextures.android.core.lms.LibraryResourceLogic
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.OERSearchResult
import com.lextures.android.features.courses.WebItemScreen
import com.lextures.android.features.home.HomeShellState
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsSegmentedChips
import com.lextures.android.features.home.LmsSkeletonList

@Composable
fun LibraryBrowseScreen(
    session: AuthSession,
    shell: HomeShellState,
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val platform = shell.platformFeatures
    val librarySearchEnabled = platform.ffLibrary
    val oerSearchEnabled = platform.oerLibraryEnabled

    var tab by remember {
        mutableStateOf(
            if (librarySearchEnabled && !oerSearchEnabled) LibraryBrowseTab.Library else LibraryBrowseTab.Oer,
        )
    }
    var query by remember { mutableStateOf("") }
    var catalogResults by remember { mutableStateOf<List<LibraryCatalogResult>>(emptyList()) }
    var oerResults by remember { mutableStateOf<List<OERSearchResult>>(emptyList()) }
    var oerProviders by remember { mutableStateOf<List<String>>(emptyList()) }
    var selectedProvider by remember { mutableStateOf<String?>(null) }
    var selectedOer by remember { mutableStateOf<OERSearchResult?>(null) }
    var selectedCatalog by remember { mutableStateOf<LibraryCatalogResult?>(null) }
    var loading by remember { mutableStateOf(false) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var hasSearched by remember { mutableStateOf(false) }

    LaunchedEffect(accessToken) {
        val token = accessToken ?: return@LaunchedEffect
        if (!oerSearchEnabled) return@LaunchedEffect
        runCatching {
            val providers = LmsApi.fetchOerProviders(token)
            oerProviders = providers
            selectedProvider = LibraryResourceLogic.defaultOerProvider(providers)
        }.onFailure { errorMessage = session.mapError(it) }
    }

    Column(modifier = modifier.fillMaxSize().padding(16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
        if (librarySearchEnabled && oerSearchEnabled) {
            LmsSegmentedChips(
                options = listOf("library" to libraryTabCatalog(), "oer" to libraryTabOer()),
                selectedId = if (tab == LibraryBrowseTab.Library) "library" else "oer",
                onSelect = { tab = if (it == "library") LibraryBrowseTab.Library else LibraryBrowseTab.Oer },
            )
        }

        OutlinedTextField(
            value = query,
            onValueChange = { query = it },
            modifier = Modifier.fillMaxWidth(),
            placeholder = {
                Text(if (tab == LibraryBrowseTab.Library && librarySearchEnabled) librarySearchCatalog() else librarySearchOer())
            },
            singleLine = true,
        )
        Button(
            onClick = {
                val token = accessToken ?: return@Button
                val q = query.trim()
                if (q.isEmpty()) return@Button
                scope.launch {
                    loading = true
                    errorMessage = null
                    try {
                        if (tab == LibraryBrowseTab.Library && librarySearchEnabled) {
                            catalogResults = LmsApi.searchLibraryCatalog(q, token)
                            oerResults = emptyList()
                        } else {
                            val provider = selectedProvider ?: LibraryResourceLogic.defaultOerProvider(oerProviders)
                            if (provider != null) {
                                oerResults = LmsApi.searchOer(provider, q, token).results
                                catalogResults = emptyList()
                            }
                        }
                        hasSearched = true
                    } catch (e: Exception) {
                        errorMessage = session.mapError(e)
                    } finally {
                        loading = false
                    }
                }
            },
            modifier = Modifier.fillMaxWidth(),
        ) {
            Text(librarySearchButton())
        }

        errorMessage?.let { LmsErrorBanner(it) }

        when {
            loading -> LmsSkeletonList(count = 4)
            !hasSearched -> LmsEmptyState(
                icon = Icons.Default.Search,
                title = librarySearchPromptTitle(),
                message = librarySearchPromptMessage(),
            )
            activeResultsEmpty(tab, librarySearchEnabled, catalogResults, oerResults) -> LmsEmptyState(
                icon = Icons.Default.Search,
                title = libraryNoResultsTitle(),
                message = libraryNoResultsMessage(),
            )
            tab == LibraryBrowseTab.Library && librarySearchEnabled -> {
                LazyColumn(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                    items(catalogResults, key = { it.mmsId ?: it.title }) { hit ->
                        ResultCard(
                            title = hit.title,
                            subtitle = hit.author,
                            detail = hit.isbn ?: hit.issn,
                            onClick = { selectedCatalog = hit },
                        )
                    }
                }
            }
            else -> {
                LazyColumn(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                    items(oerResults, key = { it.id }) { hit ->
                        ResultCard(
                            title = hit.title,
                            subtitle = hit.licenseLabel ?: hit.licenseSpdx,
                            detail = hit.subject,
                            onClick = { selectedOer = hit },
                        )
                    }
                }
            }
        }
    }

    selectedOer?.let { hit ->
        Dialog(onDismissRequest = { selectedOer = null }, properties = DialogProperties(usePlatformDefaultWidth = false)) {
            WebItemScreen(
                title = hit.title,
                urlString = hit.url,
                accessToken = accessToken,
                onOpenExternal = { uri -> context.startActivity(Intent(Intent.ACTION_VIEW, uri)) },
                modifier = Modifier.fillMaxSize(),
            )
        }
    }

    selectedCatalog?.let { hit ->
        Dialog(onDismissRequest = { selectedCatalog = null }) {
            Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                Text(hit.title, fontWeight = FontWeight.SemiBold, color = textPrimary(), fontSize = 18.sp)
                hit.author?.let { Text(it, color = textSecondary()) }
                hit.isbn?.let { Text("ISBN $it", fontSize = 12.sp, color = textSecondary()) }
                Text(libraryCatalogBrowseHint(), color = textSecondary())
            }
        }
    }
}

@Composable
private fun ResultCard(
    title: String,
    subtitle: String?,
    detail: String?,
    onClick: () -> Unit,
) {
    LmsCard(Modifier.fillMaxWidth().clickable(onClick = onClick)) {
        Text(title, fontWeight = FontWeight.SemiBold, color = textPrimary())
        subtitle?.takeIf { it.isNotBlank() }?.let { Text(it, fontSize = 12.sp, color = textSecondary()) }
        detail?.takeIf { it.isNotBlank() }?.let { Text(it, fontSize = 11.sp, color = textSecondary()) }
    }
}

private fun activeResultsEmpty(
    tab: LibraryBrowseTab,
    librarySearchEnabled: Boolean,
    catalog: List<LibraryCatalogResult>,
    oer: List<OERSearchResult>,
): Boolean = if (tab == LibraryBrowseTab.Library && librarySearchEnabled) catalog.isEmpty() else oer.isEmpty()