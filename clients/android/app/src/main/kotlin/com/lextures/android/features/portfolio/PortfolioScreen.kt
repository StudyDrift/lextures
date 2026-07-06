package com.lextures.android.features.portfolio

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.Folder
import androidx.compose.material.icons.filled.Visibility
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
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
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.StalenessChip
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalePreferences
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.PortfolioSummary
import com.lextures.android.core.offline.OfflineCacheKey
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsSkeletonList
import kotlinx.coroutines.launch
import kotlinx.serialization.builtins.ListSerializer

/** Student portfolio list (M12.1). */
@Composable
fun PortfolioScreen(
    session: AuthSession,
    localePrefs: LocalePreferences,
    onOpenPortfolio: (PortfolioSummary) -> Unit,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val offline = remember { OfflineService.get(context) }
    val isOnline by offline.networkMonitor.isOnline.collectAsState()
    val accessToken by session.accessToken.collectAsState()
    val scope = rememberCoroutineScope()

    var portfolios by remember { mutableStateOf<List<PortfolioSummary>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var cacheLabel by remember { mutableStateOf<String?>(null) }
    var showCreate by remember { mutableStateOf(false) }
    var newTitle by remember { mutableStateOf("") }
    var newIntro by remember { mutableStateOf("") }
    var creating by remember { mutableStateOf(false) }
    var createError by remember { mutableStateOf<String?>(null) }

    suspend fun load(token: String) {
        loading = portfolios.isEmpty()
        errorMessage = null
        try {
            val result = offline.cachedFetch(
                key = OfflineCacheKey.portfolioList(),
                accessToken = token,
                serializer = ListSerializer(PortfolioSummary.serializer()),
            ) {
                LmsApi.fetchMyPortfolios(token)
            }
            portfolios = result.first
            cacheLabel = result.second
                ?.takeIf { it.isStale(isOnline) }
                ?.lastUpdatedLabel()
        } catch (_: Exception) {
            errorMessage = L.text(context, localePrefs, R.string.mobile_portfolio_loadError)
        } finally {
            loading = false
        }
    }

    LaunchedEffect(accessToken) {
        val token = accessToken ?: return@LaunchedEffect
        load(token)
    }

    if (showCreate) {
        AlertDialog(
            onDismissRequest = { if (!creating) showCreate = false },
            title = { Text(L.text(context, localePrefs, R.string.mobile_portfolio_create)) },
            text = {
                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    createError?.let {
                        Text(it, fontSize = 12.sp, color = textSecondary())
                    }
                    OutlinedTextField(
                        value = newTitle,
                        onValueChange = { newTitle = it },
                        label = { Text(L.text(context, localePrefs, R.string.mobile_portfolio_fieldTitle)) },
                        modifier = Modifier.fillMaxWidth(),
                        singleLine = true,
                    )
                    OutlinedTextField(
                        value = newIntro,
                        onValueChange = { newIntro = it },
                        label = { Text(L.text(context, localePrefs, R.string.mobile_portfolio_fieldIntro)) },
                        modifier = Modifier.fillMaxWidth(),
                        minLines = 2,
                        maxLines = 4,
                    )
                }
            },
            confirmButton = {
                TextButton(
                    onClick = {
                        val token = accessToken ?: return@TextButton
                        val title = newTitle.trim()
                        if (title.isEmpty()) return@TextButton
                        scope.launch {
                            creating = true
                            createError = null
                            try {
                                val created = LmsApi.createPortfolio(
                                    title = title,
                                    introText = newIntro.trim(),
                                    accessToken = token,
                                )
                                portfolios = listOf(created) + portfolios
                                showCreate = false
                                newTitle = ""
                                newIntro = ""
                                onOpenPortfolio(created)
                            } catch (_: Exception) {
                                createError = L.text(context, localePrefs, R.string.mobile_portfolio_createError)
                            } finally {
                                creating = false
                            }
                        }
                    },
                    enabled = !creating && newTitle.trim().isNotEmpty(),
                ) {
                    Text(L.text(context, localePrefs, R.string.mobile_common_save))
                }
            },
            dismissButton = {
                TextButton(
                    onClick = { showCreate = false },
                    enabled = !creating,
                ) {
                    Text(L.text(context, localePrefs, R.string.mobile_common_cancel))
                }
            },
        )
    }

    Column(
        modifier = modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(16.dp),
    ) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                L.text(context, localePrefs, R.string.mobile_portfolio_title),
                fontWeight = FontWeight.SemiBold,
                fontSize = 18.sp,
                color = textPrimary(),
            )
            IconButton(onClick = { showCreate = true }) {
                Icon(
                    Icons.Default.Add,
                    contentDescription = L.text(context, localePrefs, R.string.mobile_portfolio_create),
                )
            }
        }

        cacheLabel?.let { StalenessChip(label = it, modifier = Modifier.padding(bottom = 8.dp)) }

        when {
            loading -> LmsSkeletonList(count = 3)
            errorMessage != null && portfolios.isEmpty() -> LmsEmptyState(
                icon = Icons.Default.Folder,
                title = L.text(context, localePrefs, R.string.mobile_portfolio_errorTitle),
                message = errorMessage!!,
            )
            portfolios.isEmpty() -> LmsEmptyState(
                icon = Icons.Default.Folder,
                title = L.text(context, localePrefs, R.string.mobile_portfolio_emptyTitle),
                message = L.text(context, localePrefs, R.string.mobile_portfolio_emptyMessage),
            )
            else -> portfolios.forEach { portfolio ->
                LmsCard(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(bottom = 12.dp),
                    onClick = { onOpenPortfolio(portfolio) },
                ) {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Icon(
                            imageVector = if (portfolio.isPublic) Icons.Default.Visibility else Icons.Default.Folder,
                            contentDescription = null,
                            modifier = Modifier.padding(end = 10.dp),
                        )
                        Column(modifier = Modifier.weight(1f)) {
                            Text(portfolio.title, fontWeight = FontWeight.SemiBold, color = textPrimary())
                            if (portfolio.introText.isNotBlank()) {
                                Text(
                                    portfolio.introText,
                                    fontSize = 12.sp,
                                    color = textSecondary(),
                                    maxLines = 2,
                                )
                            }
                        }
                    }
                }
            }
        }
    }
}