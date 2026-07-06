package com.lextures.android.features.portfolio

import android.content.ClipData
import android.content.ClipboardManager
import android.content.Context
import android.content.Intent
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
import androidx.compose.material.icons.filled.Description
import androidx.compose.material.icons.filled.Visibility
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Switch
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
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalePreferences
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.PatchPortfolioRequest
import com.lextures.android.core.lms.PortfolioArtifact
import com.lextures.android.core.lms.PortfolioLogic
import com.lextures.android.core.lms.PortfolioSummary
import com.lextures.android.core.offline.OfflineCacheKey
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsSectionHeader
import com.lextures.android.features.home.LmsSkeletonList
import kotlinx.coroutines.launch
import kotlinx.serialization.serializer

/** Portfolio editor: artifacts, visibility, share (M12.1). */
@Composable
fun PortfolioDetailScreen(
    session: AuthSession,
    localePrefs: LocalePreferences,
    portfolioId: String,
    initialTitle: String,
    onOpenArtifact: (PortfolioArtifact) -> Unit,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val offline = remember { OfflineService.get(context) }
    val accessToken by session.accessToken.collectAsState()
    val scope = rememberCoroutineScope()

    var portfolio by remember { mutableStateOf<PortfolioSummary?>(null) }
    var artifacts by remember { mutableStateOf<List<PortfolioArtifact>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var togglingPublic by remember { mutableStateOf(false) }
    var copiedLink by remember { mutableStateOf(false) }
    var showAddArtifact by remember { mutableStateOf(false) }

    val orderedArtifacts = remember(artifacts, portfolio) {
        PortfolioLogic.orderedArtifacts(artifacts, portfolio?.order.orEmpty())
    }

    suspend fun load(token: String) {
        loading = portfolio == null
        errorMessage = null
        try {
            val result = offline.cachedFetch(
                key = OfflineCacheKey.portfolioDetail(portfolioId),
                accessToken = token,
                serializer = serializer<com.lextures.android.core.lms.PortfolioDetailResponse>(),
            ) {
                LmsApi.fetchMyPortfolio(portfolioId, token)
            }
            portfolio = result.first.portfolio
            artifacts = result.first.artifacts
        } catch (_: Exception) {
            errorMessage = L.text(context, localePrefs, R.string.mobile_portfolio_loadError)
        } finally {
            loading = false
        }
    }

    LaunchedEffect(accessToken, portfolioId) {
        val token = accessToken ?: return@LaunchedEffect
        load(token)
    }

    if (showAddArtifact) {
        ArtifactEditorScreen(
            session = session,
            localePrefs = localePrefs,
            portfolioId = portfolioId,
            existing = null,
            onSaved = { created ->
                artifacts = artifacts + created
                showAddArtifact = false
            },
            onBack = { showAddArtifact = false },
            modifier = modifier,
        )
        return
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
                portfolio?.title ?: initialTitle,
                fontWeight = FontWeight.SemiBold,
                fontSize = 18.sp,
                color = textPrimary(),
            )
            IconButton(onClick = { showAddArtifact = true }) {
                Icon(
                    Icons.Default.Add,
                    contentDescription = L.text(context, localePrefs, R.string.mobile_portfolio_addArtifact),
                )
            }
        }

        when {
            loading -> LmsSkeletonList(count = 4)
            errorMessage != null && portfolio == null -> LmsEmptyState(
                icon = Icons.Default.Description,
                title = L.text(context, localePrefs, R.string.mobile_portfolio_errorTitle),
                message = errorMessage!!,
            )
            else -> {
                errorMessage?.let { LmsErrorBanner(message = it, modifier = Modifier.padding(bottom = 12.dp)) }

                portfolio?.let { current ->
                    LmsCard(modifier = Modifier.fillMaxWidth().padding(bottom = 12.dp)) {
                        if (current.introText.isNotBlank()) {
                            Text(
                                current.introText,
                                fontSize = 14.sp,
                                color = textSecondary(),
                                modifier = Modifier.padding(bottom = 8.dp),
                            )
                        }
                        Row(
                            modifier = Modifier.fillMaxWidth(),
                            horizontalArrangement = Arrangement.SpaceBetween,
                            verticalAlignment = Alignment.CenterVertically,
                        ) {
                            Text(
                                L.text(context, localePrefs, R.string.mobile_portfolio_publicToggle),
                                fontSize = 14.sp,
                                fontWeight = FontWeight.Medium,
                                color = textPrimary(),
                            )
                            Switch(
                                checked = current.isPublic,
                                onCheckedChange = { isPublic ->
                                    val token = accessToken ?: return@Switch
                                    scope.launch {
                                        togglingPublic = true
                                        errorMessage = null
                                        try {
                                            val updated = LmsApi.patchPortfolio(
                                                portfolioId = portfolioId,
                                                payload = PatchPortfolioRequest(isPublic = isPublic),
                                                accessToken = token,
                                            )
                                            portfolio = updated
                                        } catch (_: Exception) {
                                            errorMessage = L.text(
                                                context,
                                                localePrefs,
                                                R.string.mobile_portfolio_visibilityError,
                                            )
                                        } finally {
                                            togglingPublic = false
                                        }
                                    }
                                },
                                enabled = !togglingPublic,
                            )
                        }
                    }

                    if (current.isPublic) {
                        val slug = current.publicSlug?.trim().orEmpty()
                        if (slug.isNotEmpty()) {
                            val url = PortfolioLogic.publicPortfolioUrl(slug)
                            LmsCard(modifier = Modifier.fillMaxWidth().padding(bottom = 12.dp)) {
                                Text(
                                    L.text(context, localePrefs, R.string.mobile_portfolio_shareTitle),
                                    fontWeight = FontWeight.SemiBold,
                                    color = textPrimary(),
                                )
                                Text(url, fontSize = 12.sp, color = textSecondary(), modifier = Modifier.padding(vertical = 8.dp))
                                Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                                    TextButton(
                                        onClick = {
                                            val intent = Intent(Intent.ACTION_SEND).apply {
                                                type = "text/plain"
                                                putExtra(
                                                    Intent.EXTRA_TEXT,
                                                    context.getString(
                                                        R.string.mobile_portfolio_shareText,
                                                        current.title,
                                                        url,
                                                    ),
                                                )
                                            }
                                            context.startActivity(Intent.createChooser(intent, null))
                                        },
                                    ) {
                                        Text(L.text(context, localePrefs, R.string.mobile_portfolio_share))
                                    }
                                    TextButton(
                                        onClick = {
                                            val clipboard = context.getSystemService(Context.CLIPBOARD_SERVICE) as ClipboardManager
                                            clipboard.setPrimaryClip(ClipData.newPlainText("portfolio", url))
                                            copiedLink = true
                                        },
                                    ) {
                                        Text(
                                            L.text(
                                                context,
                                                localePrefs,
                                                if (copiedLink) {
                                                    R.string.mobile_portfolio_copied
                                                } else {
                                                    R.string.mobile_portfolio_copyLink
                                                },
                                            ),
                                        )
                                    }
                                }
                            }
                        }
                    }
                }

                if (orderedArtifacts.isEmpty()) {
                    LmsEmptyState(
                        icon = Icons.Default.Description,
                        title = L.text(context, localePrefs, R.string.mobile_portfolio_noArtifactsTitle),
                        message = L.text(context, localePrefs, R.string.mobile_portfolio_noArtifactsMessage),
                    )
                } else {
                    LmsSectionHeader(
                        title = L.text(context, localePrefs, R.string.mobile_portfolio_artifacts),
                        modifier = Modifier.padding(bottom = 8.dp),
                    )
                    orderedArtifacts.forEach { artifact ->
                        LmsCard(
                            modifier = Modifier
                                .fillMaxWidth()
                                .padding(bottom = 12.dp),
                            onClick = { onOpenArtifact(artifact) },
                        ) {
                            Row(verticalAlignment = Alignment.Top) {
                                Icon(
                                    imageVector = if (artifact.isPublic) Icons.Default.Visibility else Icons.Default.Description,
                                    contentDescription = null,
                                    modifier = Modifier.padding(end = 10.dp),
                                )
                                Column(modifier = Modifier.weight(1f)) {
                                    Text(artifact.title, fontWeight = FontWeight.SemiBold, color = textPrimary())
                                    Text(
                                        PortfolioLogic.artifactTypeLabel(artifact.artifactType),
                                        fontSize = 12.sp,
                                        color = textSecondary(),
                                    )
                                    if (artifact.description.isNotBlank()) {
                                        Text(
                                            artifact.description,
                                            fontSize = 12.sp,
                                            color = textSecondary(),
                                            maxLines = 2,
                                        )
                                    }
                                }
                            }
                        }
                    }
                    if (orderedArtifacts.size > 1) {
                        LmsCard(modifier = Modifier.fillMaxWidth()) {
                            Text(
                                L.text(context, localePrefs, R.string.mobile_portfolio_reorderHint),
                                fontSize = 12.sp,
                                color = textSecondary(),
                            )
                        }
                    }
                }
            }
        }
    }
}