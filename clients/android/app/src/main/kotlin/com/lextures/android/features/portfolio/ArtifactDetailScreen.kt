package com.lextures.android.features.portfolio

import android.content.Intent
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.core.net.toUri
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalePreferences
import com.lextures.android.core.lms.FilePreviewTarget
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.PortfolioArtifact
import com.lextures.android.core.lms.PortfolioLogic
import com.lextures.android.features.courses.MarkdownText
import com.lextures.android.features.files.FilePreviewScreen
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsErrorBanner
import kotlinx.coroutines.launch

/** Artifact detail with preview, edit, delete (M12.1). */
@Composable
fun ArtifactDetailScreen(
    session: AuthSession,
    localePrefs: LocalePreferences,
    portfolioId: String,
    artifact: PortfolioArtifact,
    onArtifactUpdated: (PortfolioArtifact) -> Unit,
    onDeleted: () -> Unit,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val accessToken by session.accessToken.collectAsState()
    val scope = rememberCoroutineScope()

    var current by remember(artifact) { mutableStateOf(artifact) }
    var showEdit by remember { mutableStateOf(false) }
    var showDeleteConfirm by remember { mutableStateOf(false) }
    var deleting by remember { mutableStateOf(false) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var openPreview by remember { mutableStateOf<FilePreviewTarget?>(null) }

    if (showEdit) {
        ArtifactEditorScreen(
            session = session,
            localePrefs = localePrefs,
            portfolioId = portfolioId,
            existing = current,
            onSaved = { updated ->
                current = updated
                onArtifactUpdated(updated)
                showEdit = false
            },
            onBack = { showEdit = false },
            modifier = modifier,
        )
        return
    }

    openPreview?.let { target ->
        FilePreviewScreen(
            session = session,
            target = target,
            onBack = { openPreview = null },
            modifier = modifier,
        )
        return
    }

    if (showDeleteConfirm) {
        AlertDialog(
            onDismissRequest = { if (!deleting) showDeleteConfirm = false },
            title = { Text(L.text(context, localePrefs, R.string.mobile_portfolio_deleteConfirm)) },
            confirmButton = {
                TextButton(
                    onClick = {
                        val token = accessToken ?: return@TextButton
                        scope.launch {
                            deleting = true
                            errorMessage = null
                            try {
                                LmsApi.deleteArtifact(portfolioId, current.id, token)
                                showDeleteConfirm = false
                                onDeleted()
                            } catch (_: Exception) {
                                errorMessage = L.text(context, localePrefs, R.string.mobile_portfolio_deleteError)
                            } finally {
                                deleting = false
                            }
                        }
                    },
                    enabled = !deleting,
                ) {
                    Text(L.text(context, localePrefs, R.string.mobile_portfolio_deleteArtifact))
                }
            },
            dismissButton = {
                TextButton(onClick = { showDeleteConfirm = false }, enabled = !deleting) {
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
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        TextButton(onClick = onBack) {
            Text(L.text(context, localePrefs, R.string.mobile_ia_close))
        }

        errorMessage?.let { LmsErrorBanner(message = it) }

        LmsCard {
            Text(current.title, fontWeight = FontWeight.Bold, fontSize = 18.sp, color = textPrimary())
            Text(
                PortfolioLogic.artifactTypeLabel(current.artifactType),
                fontSize = 12.sp,
                color = textSecondary(),
            )
            if (current.description.isNotBlank()) {
                Text(current.description, fontSize = 14.sp, color = textSecondary(), modifier = Modifier.padding(top = 8.dp))
            }
            if (current.outcomeIds.isNotEmpty()) {
                Text(
                    context.getString(R.string.mobile_portfolio_outcomeCount, current.outcomeIds.size),
                    fontSize = 12.sp,
                    color = textSecondary(),
                    modifier = Modifier.padding(top = 4.dp),
                )
            }
        }

        if (PortfolioLogic.isContentPage(current) && current.textContent.isNotBlank()) {
            LmsCard {
                MarkdownText(current.textContent)
            }
        }

        if (current.externalUrl.isNotBlank()) {
            LmsCard {
                TextButton(
                    onClick = {
                        runCatching {
                            context.startActivity(Intent(Intent.ACTION_VIEW, current.externalUrl.toUri()))
                        }
                    },
                ) {
                    Text(current.externalUrl)
                }
            }
        }

        if (PortfolioLogic.hasFile(current)) {
            LmsCard {
                Button(
                    onClick = {
                        openPreview = FilePreviewTarget.portfolioArtifact(
                            portfolioId = portfolioId,
                            artifactId = current.id,
                            fileName = current.fileName.ifBlank { current.title },
                            mimeType = current.fileMime.ifBlank { null },
                        )
                    },
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    Text(L.text(context, localePrefs, R.string.mobile_portfolio_previewFile))
                }
            }
        }

        LmsCard {
            Button(
                onClick = { showEdit = true },
                modifier = Modifier.fillMaxWidth(),
            ) {
                Text(L.text(context, localePrefs, R.string.mobile_portfolio_editArtifact))
            }
            TextButton(
                onClick = { showDeleteConfirm = true },
                enabled = !deleting,
                modifier = Modifier.fillMaxWidth(),
            ) {
                Text(L.text(context, localePrefs, R.string.mobile_portfolio_deleteArtifact))
            }
        }
    }
}