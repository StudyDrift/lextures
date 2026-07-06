package com.lextures.android.features.wallet

import android.content.Intent
import android.net.Uri
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.AccountBalanceWallet
import androidx.compose.material3.Button
import androidx.compose.material3.Checkbox
import androidx.compose.material3.HorizontalDivider
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
import com.lextures.android.core.lms.CCRAchievement
import com.lextures.android.core.lms.CCRDocument
import com.lextures.android.core.lms.CETranscriptAward
import com.lextures.android.core.lms.CCRSummaryResponse
import com.lextures.android.core.lms.CETranscriptResponse
import com.lextures.android.core.lms.CredentialsLogic
import com.lextures.android.core.lms.FilePreviewTarget
import com.lextures.android.core.lms.IssuedCredentialSummary
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.TranscriptRequestSummary
import com.lextures.android.core.lms.WalletLogic
import com.lextures.android.core.navigation.MobilePlatformFeatures
import com.lextures.android.core.offline.OfflineCacheKey
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.features.credentials.CredentialDetailScreen
import com.lextures.android.features.files.FilePreviewScreen
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsSkeletonList
import kotlinx.coroutines.launch
import kotlinx.serialization.builtins.ListSerializer

private enum class WalletRoute {
    List,
    Credential,
    Ccr,
    CeTranscript,
    OfficialTranscripts,
    PdfPreview,
}

/** Consolidated credentials wallet (M12.2). */
@Composable
fun WalletScreen(
    session: AuthSession,
    localePrefs: LocalePreferences,
    platform: MobilePlatformFeatures,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val offline = remember { OfflineService.get(context) }
    val isOnline by offline.networkMonitor.isOnline.collectAsState()
    val accessToken by session.accessToken.collectAsState()

    var route by remember { mutableStateOf(WalletRoute.List) }
    var credentials by remember { mutableStateOf<List<IssuedCredentialSummary>>(emptyList()) }
    var ccrAchievements by remember { mutableStateOf<List<CCRAchievement>>(emptyList()) }
    var ccrDocuments by remember { mutableStateOf<List<CCRDocument>>(emptyList()) }
    var ceAwards by remember { mutableStateOf<List<CETranscriptAward>>(emptyList()) }
    var transcriptRequests by remember { mutableStateOf<List<TranscriptRequestSummary>>(emptyList()) }
    var selectedCredential by remember { mutableStateOf<IssuedCredentialSummary?>(null) }
    var previewTarget by remember { mutableStateOf<FilePreviewTarget?>(null) }
    var loading by remember { mutableStateOf(true) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var cacheLabel by remember { mutableStateOf<String?>(null) }

    suspend fun load(token: String) {
        loading = credentials.isEmpty() && ccrAchievements.isEmpty() && ceAwards.isEmpty() &&
            transcriptRequests.isEmpty()
        errorMessage = null
        var sawCache = false
        var loadError = false
        var staleLabel: String? = null
        if (WalletLogic.credentialsSectionEnabled(platform)) {
            runCatching {
                val result = offline.cachedFetch(
                    key = OfflineCacheKey.credentialsList(),
                    accessToken = token,
                    serializer = ListSerializer(IssuedCredentialSummary.serializer()),
                ) { LmsApi.fetchMyCredentials(token) }
                credentials = result.first
                result.second?.takeIf { it.isStale(isOnline) }?.lastUpdatedLabel()?.let {
                    sawCache = true
                    staleLabel = it
                }
            }.onFailure { if (credentials.isEmpty()) loadError = true }
        }
        if (WalletLogic.ccrEnabled(platform)) {
            runCatching {
                val result = offline.cachedFetch(
                    key = OfflineCacheKey.walletCcr(),
                    accessToken = token,
                    serializer = CCRSummaryResponse.serializer(),
                ) { LmsApi.fetchMyCcr(token) }
                ccrAchievements = result.first.achievements.orEmpty()
                ccrDocuments = result.first.documents.orEmpty()
                result.second?.takeIf { it.isStale(isOnline) }?.lastUpdatedLabel()?.let {
                    sawCache = true
                    staleLabel = it
                }
            }.onFailure { if (ccrAchievements.isEmpty() && ccrDocuments.isEmpty()) loadError = true }
        }
        if (WalletLogic.ceTranscriptEnabled(platform)) {
            runCatching {
                val result = offline.cachedFetch(
                    key = OfflineCacheKey.walletCeTranscript(),
                    accessToken = token,
                    serializer = CETranscriptResponse.serializer(),
                ) { LmsApi.fetchCeTranscript(token) }
                ceAwards = result.first.awards.orEmpty()
                result.second?.takeIf { it.isStale(isOnline) }?.lastUpdatedLabel()?.let {
                    sawCache = true
                    staleLabel = it
                }
            }.onFailure { if (ceAwards.isEmpty()) loadError = true }
        }
        if (WalletLogic.officialTranscriptsEnabled(platform)) {
            runCatching {
                val result = offline.cachedFetch(
                    key = OfflineCacheKey.walletTranscriptRequests(),
                    accessToken = token,
                    serializer = ListSerializer(TranscriptRequestSummary.serializer()),
                ) { LmsApi.fetchTranscriptRequests(token) }
                transcriptRequests = result.first
                result.second?.takeIf { it.isStale(isOnline) }?.lastUpdatedLabel()?.let {
                    sawCache = true
                    staleLabel = it
                }
            }.onFailure { if (transcriptRequests.isEmpty()) loadError = true }
        }
        if (loadError && credentials.isEmpty() && ccrAchievements.isEmpty() && ceAwards.isEmpty() &&
            transcriptRequests.isEmpty()
        ) {
            errorMessage = L.text(context, localePrefs, R.string.mobile_wallet_loadError)
        }
        cacheLabel = staleLabel.takeIf { sawCache }
        loading = false
    }

    LaunchedEffect(accessToken, platform) {
        val token = accessToken ?: return@LaunchedEffect
        load(token)
    }

    when (route) {
        WalletRoute.Credential -> selectedCredential?.let { credential ->
            Column(modifier = modifier.fillMaxSize()) {
                TextButton(onClick = { route = WalletRoute.List }) {
                    Text(L.text(context, localePrefs, R.string.mobile_ia_close))
                }
                CredentialDetailScreen(
                    session = session,
                    localePrefs = localePrefs,
                    credential = credential,
                    modifier = Modifier.fillMaxSize(),
                )
            }
        }
        WalletRoute.Ccr -> WalletCcrDetailScreen(
            session = session,
            localePrefs = localePrefs,
            achievements = ccrAchievements,
            documents = ccrDocuments,
            onBack = { route = WalletRoute.List },
            onUpdate = { achievements, documents ->
                ccrAchievements = achievements
                ccrDocuments = documents
            },
            onPreviewPdf = { documentId ->
                previewTarget = WalletLogic.ccrPdfPreviewTarget(documentId)
                route = WalletRoute.PdfPreview
            },
            modifier = modifier,
        )
        WalletRoute.CeTranscript -> WalletCeTranscriptDetailScreen(
            localePrefs = localePrefs,
            awards = ceAwards,
            onBack = { route = WalletRoute.List },
            onPreviewPdf = {
                previewTarget = WalletLogic.ceTranscriptPdfPreviewTarget()
                route = WalletRoute.PdfPreview
            },
            modifier = modifier,
        )
        WalletRoute.OfficialTranscripts -> WalletOfficialTranscriptDetailScreen(
            localePrefs = localePrefs,
            requests = transcriptRequests,
            onBack = { route = WalletRoute.List },
            modifier = modifier,
        )
        WalletRoute.PdfPreview -> previewTarget?.let { target ->
            FilePreviewScreen(
                session = session,
                target = target,
                onBack = {
                    previewTarget = null
                    route = WalletRoute.List
                },
                modifier = modifier,
            )
        }
        WalletRoute.List -> Column(
            modifier = modifier
                .fillMaxSize()
                .verticalScroll(rememberScrollState())
                .padding(16.dp),
        ) {
            when {
                loading -> LmsSkeletonList(count = 4)
                errorMessage != null && isWalletEmpty(credentials, ccrAchievements, ceAwards, transcriptRequests) ->
                    LmsEmptyState(
                        icon = Icons.Default.AccountBalanceWallet,
                        title = L.text(context, localePrefs, R.string.mobile_wallet_errorTitle),
                        message = errorMessage!!,
                    )
                isWalletEmpty(credentials, ccrAchievements, ceAwards, transcriptRequests) ->
                    LmsEmptyState(
                        icon = Icons.Default.AccountBalanceWallet,
                        title = L.text(context, localePrefs, R.string.mobile_wallet_emptyTitle),
                        message = L.text(context, localePrefs, R.string.mobile_wallet_emptyMessage),
                    )
                else -> {
                    cacheLabel?.let { StalenessChip(label = it) }
                    if (WalletLogic.credentialsSectionEnabled(platform) && credentials.isNotEmpty()) {
                        sectionHeader(L.text(context, localePrefs, R.string.mobile_wallet_section_credentials))
                        credentials.forEach { credential ->
                            walletRow(
                                title = credential.title,
                                subtitle = CredentialsLogic.sourceTypeLabel(credential.sourceType),
                                detail = context.getString(
                                    R.string.mobile_credentials_issued,
                                    WalletLogic.dateLabel(credential.issuedAt),
                                ),
                                onClick = {
                                    selectedCredential = credential
                                    route = WalletRoute.Credential
                                },
                            )
                        }
                    }
                    if (WalletLogic.ccrEnabled(platform)) {
                        sectionHeader(L.text(context, localePrefs, R.string.mobile_wallet_section_ccr))
                        walletRow(
                            title = L.text(context, localePrefs, R.string.mobile_wallet_ccr_detailTitle),
                            subtitle = context.getString(R.string.mobile_wallet_itemCount, ccrAchievements.size),
                            detail = L.text(context, localePrefs, R.string.mobile_wallet_viewDetails),
                            onClick = { route = WalletRoute.Ccr },
                        )
                    }
                    if (WalletLogic.ceTranscriptEnabled(platform)) {
                        sectionHeader(L.text(context, localePrefs, R.string.mobile_wallet_section_ceTranscript))
                        walletRow(
                            title = L.text(context, localePrefs, R.string.mobile_wallet_ceTranscript_detailTitle),
                            subtitle = context.getString(R.string.mobile_wallet_itemCount, ceAwards.size),
                            detail = L.text(context, localePrefs, R.string.mobile_wallet_ceTranscript_previewPdf),
                            onClick = { route = WalletRoute.CeTranscript },
                        )
                    }
                    if (WalletLogic.officialTranscriptsEnabled(platform)) {
                        sectionHeader(L.text(context, localePrefs, R.string.mobile_wallet_section_officialTranscripts))
                        walletRow(
                            title = L.text(context, localePrefs, R.string.mobile_wallet_officialTranscripts_detailTitle),
                            subtitle = context.getString(R.string.mobile_wallet_itemCount, transcriptRequests.size),
                            detail = L.text(context, localePrefs, R.string.mobile_wallet_openWebRequest),
                            onClick = { route = WalletRoute.OfficialTranscripts },
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun sectionHeader(title: String) {
    Text(title, fontWeight = FontWeight.SemiBold, color = textPrimary(), modifier = Modifier.padding(vertical = 8.dp))
}

@Composable
private fun walletRow(title: String, subtitle: String, detail: String, onClick: () -> Unit) {
    LmsCard(
        modifier = Modifier
            .fillMaxWidth()
            .padding(bottom = 12.dp)
            .clickable(onClick = onClick),
    ) {
        Text(title, fontWeight = FontWeight.SemiBold, color = textPrimary())
        Text(subtitle, fontSize = 12.sp, color = textSecondary())
        Text(detail, fontSize = 12.sp, color = textSecondary())
    }
}

private fun isWalletEmpty(
    credentials: List<IssuedCredentialSummary>,
    ccrAchievements: List<CCRAchievement>,
    ceAwards: List<CETranscriptAward>,
    requests: List<TranscriptRequestSummary>,
): Boolean = credentials.isEmpty() && ccrAchievements.isEmpty() && ceAwards.isEmpty() && requests.isEmpty()

@Composable
private fun WalletCcrDetailScreen(
    session: AuthSession,
    localePrefs: LocalePreferences,
    achievements: List<CCRAchievement>,
    documents: List<CCRDocument>,
    onBack: () -> Unit,
    onUpdate: (List<CCRAchievement>, List<CCRDocument>) -> Unit,
    onPreviewPdf: (String) -> Unit,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val accessToken by session.accessToken.collectAsState()
    val scope = rememberCoroutineScope()
    var localAchievements by remember(achievements) { mutableStateOf(achievements) }
    var localDocuments by remember(documents) { mutableStateOf(documents) }
    var sharePublicly by remember { mutableStateOf(false) }
    var generating by remember { mutableStateOf(false) }
    var actionError by remember { mutableStateOf<String?>(null) }

    Column(
        modifier = modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(16.dp),
    ) {
        TextButton(onClick = onBack) { Text(L.text(context, localePrefs, R.string.mobile_ia_close)) }
        actionError?.let { Text(it, color = textSecondary()) }
        LmsCard(modifier = Modifier.fillMaxWidth().padding(bottom = 12.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Checkbox(checked = sharePublicly, onCheckedChange = { sharePublicly = it })
                Text(L.text(context, localePrefs, R.string.mobile_wallet_ccr_sharePublicly))
            }
            Button(
                onClick = {
                    val token = accessToken ?: return@Button
                    scope.launch {
                        generating = true
                        actionError = null
                        runCatching {
                            val result = LmsApi.generateMyCcr(sharePublicly, token)
                            localDocuments = listOf(result.document) + localDocuments.filter { it.id != result.document.id }
                            result.achievements?.let { localAchievements = it }
                            onUpdate(localAchievements, localDocuments)
                        }.onFailure {
                            actionError = L.text(context, localePrefs, R.string.mobile_wallet_ccr_generateError)
                        }
                        generating = false
                    }
                },
                enabled = !generating,
                modifier = Modifier.fillMaxWidth(),
            ) {
                Text(
                    if (generating) {
                        L.text(context, localePrefs, R.string.mobile_wallet_ccr_generating)
                    } else {
                        L.text(context, localePrefs, R.string.mobile_wallet_ccr_generate)
                    },
                )
            }
        }
        if (localAchievements.isEmpty()) {
            Text(L.text(context, localePrefs, R.string.mobile_wallet_ccr_emptyAchievements), color = textSecondary())
        } else {
            localAchievements.groupBy { it.type }.forEach { (type, items) ->
                Text(WalletLogic.achievementTypeLabel(type), fontWeight = FontWeight.SemiBold, color = textPrimary())
                items.forEach { item ->
                    LmsCard(modifier = Modifier.fillMaxWidth().padding(vertical = 4.dp)) {
                        Text(item.title, fontWeight = FontWeight.SemiBold, color = textPrimary())
                        item.description?.let { Text(it, fontSize = 12.sp, color = textSecondary()) }
                        Text(WalletLogic.dateLabel(item.issuedAt), fontSize = 12.sp, color = textSecondary())
                    }
                }
            }
        }
        if (localDocuments.isNotEmpty()) {
            Text(
                L.text(context, localePrefs, R.string.mobile_wallet_ccr_documents),
                fontWeight = FontWeight.SemiBold,
                color = textPrimary(),
                modifier = Modifier.padding(top = 12.dp),
            )
            localDocuments.forEach { doc ->
                LmsCard(modifier = Modifier.fillMaxWidth().padding(vertical = 4.dp)) {
                    Text(WalletLogic.dateLabel(doc.generatedAt), fontWeight = FontWeight.SemiBold, color = textPrimary())
                    doc.verificationUrl?.let { url ->
                        Text(url, fontSize = 11.sp, color = textSecondary())
                        TextButton(onClick = {
                            context.startActivity(Intent(Intent.ACTION_SEND).apply {
                                type = "text/plain"
                                putExtra(Intent.EXTRA_TEXT, url)
                            })
                        }) { Text(L.text(context, localePrefs, R.string.mobile_wallet_shareVerify)) }
                        TextButton(onClick = {
                            context.startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(url)))
                        }) { Text(L.text(context, localePrefs, R.string.mobile_wallet_openVerify)) }
                    } ?: Text(
                        L.text(context, localePrefs, R.string.mobile_wallet_verification_private),
                        fontSize = 12.sp,
                        color = textSecondary(),
                    )
                    TextButton(onClick = { onPreviewPdf(doc.id) }) {
                        Text(L.text(context, localePrefs, R.string.mobile_wallet_downloadPdf))
                    }
                }
            }
        }
    }
}

@Composable
private fun WalletCeTranscriptDetailScreen(
    localePrefs: LocalePreferences,
    awards: List<CETranscriptAward>,
    onBack: () -> Unit,
    onPreviewPdf: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    Column(
        modifier = modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(16.dp),
    ) {
        TextButton(onClick = onBack) { Text(L.text(context, localePrefs, R.string.mobile_ia_close)) }
        Button(onClick = onPreviewPdf, modifier = Modifier.fillMaxWidth()) {
            Text(L.text(context, localePrefs, R.string.mobile_wallet_ceTranscript_previewPdf))
        }
        if (awards.isEmpty()) {
            Text(L.text(context, localePrefs, R.string.mobile_wallet_ceTranscript_empty), color = textSecondary())
        } else {
            LmsCard(modifier = Modifier.fillMaxWidth().padding(top = 12.dp)) {
                awards.forEachIndexed { index, award ->
                    if (index > 0) HorizontalDivider()
                    Text(award.courseTitle, fontWeight = FontWeight.SemiBold, color = textPrimary())
                    Text(
                        "${String.format("%.2f", award.ceuCredit)} CEU · " +
                            "${String.format("%.1f", award.contactHours)} hrs · " +
                            WalletLogic.dateLabel(award.completedAt),
                        fontSize = 12.sp,
                        color = textSecondary(),
                    )
                }
            }
        }
    }
}

@Composable
private fun WalletOfficialTranscriptDetailScreen(
    localePrefs: LocalePreferences,
    requests: List<TranscriptRequestSummary>,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    Column(
        modifier = modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(16.dp),
    ) {
        TextButton(onClick = onBack) { Text(L.text(context, localePrefs, R.string.mobile_ia_close)) }
        LmsCard(modifier = Modifier.fillMaxWidth()) {
            Text(L.text(context, localePrefs, R.string.mobile_wallet_officialTranscriptsHint), color = textSecondary())
            Button(
                onClick = {
                    context.startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(WalletLogic.officialTranscriptWebUrl())))
                },
                modifier = Modifier.fillMaxWidth().padding(top = 8.dp),
            ) {
                Text(L.text(context, localePrefs, R.string.mobile_wallet_openWebRequest))
            }
        }
        Text(
            L.text(context, localePrefs, R.string.mobile_wallet_requestHistory),
            fontWeight = FontWeight.SemiBold,
            color = textPrimary(),
            modifier = Modifier.padding(vertical = 8.dp),
        )
        if (requests.isEmpty()) {
            Text(L.text(context, localePrefs, R.string.mobile_wallet_noRequests), color = textSecondary())
        } else {
            requests.forEach { request ->
                LmsCard(modifier = Modifier.fillMaxWidth().padding(bottom = 8.dp)) {
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween,
                    ) {
                        Text(WalletLogic.dateLabel(request.requestedAt), fontWeight = FontWeight.SemiBold, color = textPrimary())
                        Text(WalletLogic.transcriptStatusLabel(request.status), fontSize = 12.sp, color = textSecondary())
                    }
                    Text(WalletLogic.deliveryTypeLabel(request.deliveryType), fontSize = 12.sp, color = textSecondary())
                }
            }
        }
    }
}
