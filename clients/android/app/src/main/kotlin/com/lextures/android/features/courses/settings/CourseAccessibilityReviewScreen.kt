package com.lextures.android.features.courses.settings

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.Computer
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateMapOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import coil.compose.AsyncImage
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.OfflineBanner
import com.lextures.android.core.design.StalenessChip
import com.lextures.android.core.design.UnsavedChangesBanner
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalLocalePreferences
import com.lextures.android.core.lms.CourseAccessibilityReviewLogic
import com.lextures.android.core.lms.CourseAccessibilityInfo
import com.lextures.android.core.lms.CourseStructureItem
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.PatchItemMarkdownBody
import com.lextures.android.core.lms.UncoveredAccessibilityItem
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsSkeletonList
import kotlinx.coroutines.launch
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json

private val accessibilityJson = Json { ignoreUnknownKeys = true }

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun CourseAccessibilityReviewScreen(
    session: AuthSession,
    course: CourseSummary,
    offline: OfflineService,
) {
    val scope = rememberCoroutineScope()
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    val isOnline by offline.networkMonitor.isOnline.collectAsState()

    var data by remember { mutableStateOf<CourseAccessibilityInfo?>(null) }
    var listPage by remember { mutableIntStateOf(0) }
    var loading by remember { mutableStateOf(true) }
    var loadError by remember { mutableStateOf<String?>(null) }
    var actionSuccess by remember { mutableStateOf<String?>(null) }
    var cacheLabel by remember { mutableStateOf<String?>(null) }
    var selectedItem by remember { mutableStateOf<UncoveredAccessibilityItem?>(null) }

    val coverage = data?.altTextCoverage
    val uncoveredItems = coverage?.uncoveredItems.orEmpty()
    val visibleItems = remember(uncoveredItems, listPage) {
        CourseAccessibilityReviewLogic.paginatedUncoveredItems(uncoveredItems, listPage)
    }
    val savedMessage = L.text(R.string.mobile_courseSettings_accessibility_saved)

    LaunchedEffect(course.courseCode) {
        val token = session.accessToken.value ?: return@LaunchedEffect
        loading = true
        loadError = null
        runCatching {
            val cached = offline.cachedFetch(
                key = CourseAccessibilityReviewLogic.cacheKey(course.courseCode),
                accessToken = token,
                serializer = CourseAccessibilityInfo.serializer(),
            ) {
                LmsApi.fetchCourseAccessibility(course.courseCode, token)
            }
            data = cached.first
            listPage = 0
            cacheLabel = cached.second?.takeIf { it.isStale(isOnline) }?.lastUpdatedLabel()
        }.onFailure { loadError = it.message }
        loading = false
    }

    Column(modifier = Modifier.fillMaxSize()) {
        LazyColumn(
            modifier = Modifier.weight(1f),
            verticalArrangement = Arrangement.spacedBy(12.dp),
            contentPadding = androidx.compose.foundation.layout.PaddingValues(16.dp),
        ) {
            if (!isOnline) item { OfflineBanner() }
            loadError?.let { msg -> item { LmsErrorBanner(message = msg) } }
            cacheLabel?.let { label -> item { StalenessChip(label = label) } }
            actionSuccess?.let { msg ->
                item {
                    LmsCard {
                        Text(msg, fontWeight = FontWeight.SemiBold, color = Color(0xFF0D9488))
                    }
                }
            }

            if (loading) {
                item { LmsSkeletonList(count = 4) }
            } else {
                item {
                    LmsCard {
                        Text(
                            L.text(R.string.mobile_courseSettings_accessibility_introTitle),
                            fontWeight = FontWeight.SemiBold,
                        )
                        Text(L.text(R.string.mobile_courseSettings_accessibility_introDescription))
                    }
                }
                coverage?.let { cov ->
                    item {
                        LmsCard {
                            Text(
                                L.text(R.string.mobile_courseSettings_accessibility_coverageTitle),
                                fontWeight = FontWeight.SemiBold,
                            )
                            Text(L.text(R.string.mobile_courseSettings_accessibility_coverageDescription))
                            LinearProgressIndicator(
                                progress = { cov.percent / 100f },
                                modifier = Modifier.fillMaxWidth().padding(vertical = 8.dp),
                            )
                            Text(
                                L.format(
                                    R.string.mobile_courseSettings_accessibility_coverageValue,
                                    cov.percent,
                                    cov.withAlt,
                                    cov.total,
                                ),
                                fontWeight = FontWeight.Bold,
                            )
                            if (data?.hardBlockSave == true) {
                                Text(
                                    L.text(R.string.mobile_courseSettings_accessibility_hardBlockNote),
                                    color = Color(0xFFD97706),
                                )
                            }
                        }
                    }
                    if (cov.uncoveredItems.isEmpty()) {
                        item {
                            LmsCard {
                                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                                    androidx.compose.material3.Icon(
                                        Icons.Default.CheckCircle,
                                        contentDescription = null,
                                        tint = Color(0xFF0D9488),
                                    )
                                    Text(
                                        L.text(R.string.mobile_courseSettings_accessibility_emptyState),
                                        fontWeight = FontWeight.SemiBold,
                                    )
                                }
                            }
                        }
                    } else {
                        item {
                            LmsCard {
                                Text(
                                    L.text(R.string.mobile_courseSettings_accessibility_gapsTitle),
                                    fontWeight = FontWeight.SemiBold,
                                )
                                visibleItems.forEachIndexed { index, item ->
                                    UncoveredItemRow(item = item, onClick = { selectedItem = item })
                                    if (index < visibleItems.lastIndex) HorizontalDivider()
                                }
                                if (CourseAccessibilityReviewLogic.hasMorePages(uncoveredItems, listPage)) {
                                    TextButton(onClick = { listPage += 1 }) {
                                        Text(L.text(R.string.mobile_courseSettings_accessibility_loadMore))
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
    }

    selectedItem?.let { item ->
        AltTextItemFixerSheet(
            session = session,
            course = course,
            offline = offline,
            item = item,
            onDismiss = { selectedItem = null },
            onSaved = {
                selectedItem = null
                actionSuccess = savedMessage
                scope.launch {
                    val token = session.accessToken.value ?: return@launch
                    runCatching {
                        data = LmsApi.fetchCourseAccessibility(course.courseCode, token)
                        listPage = 0
                    }
                }
            },
        )
    }
}

@Composable
private fun UncoveredItemRow(
    item: UncoveredAccessibilityItem,
    onClick: () -> Unit,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onClick)
            .padding(vertical = 8.dp),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Column(modifier = Modifier.weight(1f)) {
            Text(
                text = item.title.ifBlank { L.text(R.string.mobile_courseSettings_accessibility_untitled) },
                fontWeight = FontWeight.Medium,
            )
            Text(L.text(CourseAccessibilityReviewLogic.kindLabelRes(item.kind)))
            Text(
                L.format(
                    R.string.mobile_courseSettings_accessibility_itemMissing,
                    item.missing,
                    item.total,
                ),
                color = Color(0xFFD97706),
            )
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun AltTextItemFixerSheet(
    session: AuthSession,
    course: CourseSummary,
    offline: OfflineService,
    item: UncoveredAccessibilityItem,
    onDismiss: () -> Unit,
    onSaved: () -> Unit,
) {
    val scope = rememberCoroutineScope()
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    val isOnline by offline.networkMonitor.isOnline.collectAsState()
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)

    var markdown by remember { mutableStateOf("") }
    var missingImages by remember { mutableStateOf<List<CourseAccessibilityReviewLogic.MarkdownImageRef>>(emptyList()) }
    val drafts = remember { mutableStateMapOf<Int, CourseAccessibilityReviewLogic.ImageAltDraft>() }
    var loading by remember { mutableStateOf(true) }
    var loadError by remember { mutableStateOf<String?>(null) }
    var actionError by remember { mutableStateOf<String?>(null) }
    var saving by remember { mutableStateOf(false) }
    var suggestingIndex by remember { mutableStateOf<Int?>(null) }
    var aiUnavailable by remember { mutableStateOf(false) }

    val supportsEdit = CourseAccessibilityReviewLogic.supportsInlineEdit(item.kind)
    val pendingUpdates = remember(missingImages, drafts.toMap()) {
        CourseAccessibilityReviewLogic.pendingUpdates(missingImages, drafts)
    }
    val isDirty = pendingUpdates.isNotEmpty()
    val saveLabel = L.text(R.string.mobile_courseSettings_accessibility_saveLabel)
    val saveErrorMessage = L.text(R.string.mobile_courseSettings_accessibility_saveError)
    val loadItemErrorMessage = L.text(R.string.mobile_courseSettings_accessibility_loadItemError)

    LaunchedEffect(item.itemId) {
        val token = session.accessToken.value ?: return@LaunchedEffect
        if (!supportsEdit) {
            loading = false
            return@LaunchedEffect
        }
        loading = true
        loadError = null
        runCatching {
            val structureItem = CourseStructureItem(id = item.itemId, kind = item.kind, title = item.title)
            val detail = LmsApi.fetchItemDetail(course.courseCode, structureItem, token)
                ?: throw IllegalStateException(loadItemErrorMessage)
            val loadedMarkdown = detail.markdown.orEmpty()
            markdown = loadedMarkdown
            val images = CourseAccessibilityReviewLogic.missingImages(loadedMarkdown)
            missingImages = images
            drafts.clear()
            drafts.putAll(CourseAccessibilityReviewLogic.drafts(images))
        }.onFailure { loadError = it.message }
        loading = false
    }

    ModalBottomSheet(onDismissRequest = onDismiss, sheetState = sheetState) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 16.dp)
                .padding(bottom = 24.dp),
        ) {
            Text(
                text = item.title.ifBlank { L.text(R.string.mobile_courseSettings_accessibility_untitled) },
                fontWeight = FontWeight.SemiBold,
                modifier = Modifier.padding(bottom = 12.dp),
            )

            if (loading) {
                CircularProgressIndicator(modifier = Modifier.align(Alignment.CenterHorizontally))
            } else if (!supportsEdit) {
                LmsEmptyState(
                    icon = Icons.Default.Computer,
                    title = L.text(R.string.mobile_courseSettings_accessibility_linkOutTitle),
                    message = L.text(R.string.mobile_courseSettings_accessibility_linkOutMessage),
                )
            } else {
                if (!isOnline) OfflineBanner()
                loadError?.let { LmsErrorBanner(message = it) }
                actionError?.let { LmsErrorBanner(message = it) }
                if (aiUnavailable) {
                    Text(L.text(R.string.mobile_courseSettings_accessibility_aiUnavailable))
                }

                Column(
                    modifier = Modifier
                        .weight(1f, fill = false)
                        .verticalScroll(rememberScrollState()),
                    verticalArrangement = Arrangement.spacedBy(12.dp),
                ) {
                    if (missingImages.isEmpty()) {
                        LmsCard {
                            Text(
                                L.text(R.string.mobile_courseSettings_accessibility_itemComplete),
                                fontWeight = FontWeight.SemiBold,
                                color = Color(0xFF0D9488),
                            )
                        }
                    } else {
                        missingImages.forEach { image ->
                            ImageEditorCard(
                                image = image,
                                draft = drafts[image.globalIndex]
                                    ?: CourseAccessibilityReviewLogic.ImageAltDraft(image.alt, image.decorative),
                                onDraftChange = { drafts[image.globalIndex] = it },
                                suggesting = suggestingIndex == image.globalIndex,
                                onSuggest = {
                                    scope.launch {
                                        val token = session.accessToken.value ?: return@launch
                                        suggestingIndex = image.globalIndex
                                        runCatching {
                                            val suggestion = LmsApi.suggestAltText(
                                                course.courseCode,
                                                image.src,
                                                "",
                                                token,
                                            )
                                            if (suggestion.suggestion.isNotBlank()) {
                                                drafts[image.globalIndex] = CourseAccessibilityReviewLogic.ImageAltDraft(
                                                    alt = suggestion.suggestion,
                                                    decorative = false,
                                                )
                                                aiUnavailable = false
                                            }
                                        }.onFailure { aiUnavailable = true }
                                        suggestingIndex = null
                                    }
                                },
                            )
                        }
                    }
                }

                if (isDirty) {
                    UnsavedChangesBanner(
                        isSaving = saving,
                        onSave = {
                            scope.launch {
                                val token = session.accessToken.value ?: return@launch
                                val path = CourseAccessibilityReviewLogic.markdownPatchPath(
                                    course.courseCode,
                                    item.itemId,
                                    item.kind,
                                ) ?: return@launch
                                val updatedMarkdown = CourseAccessibilityReviewLogic.applyAltTextUpdates(
                                    markdown,
                                    pendingUpdates,
                                ) ?: run {
                                    actionError = saveErrorMessage
                                    return@launch
                                }
                                saving = true
                                actionError = null
                                runCatching {
                                    offline.enqueueMutation(
                                        method = "PATCH",
                                        path = path,
                                        bodyJson = accessibilityJson.encodeToString(
                                            PatchItemMarkdownBody(updatedMarkdown),
                                        ),
                                        label = saveLabel,
                                        accessToken = token,
                                        idempotencyKey = CourseAccessibilityReviewLogic.saveMarkdownIdempotencyKey(
                                            course.courseCode,
                                            item.itemId,
                                            item.kind,
                                        ),
                                    )
                                    markdown = updatedMarkdown
                                    val images = CourseAccessibilityReviewLogic.missingImages(updatedMarkdown)
                                    missingImages = images
                                    drafts.clear()
                                    drafts.putAll(CourseAccessibilityReviewLogic.drafts(images))
                                    if (images.isEmpty()) onSaved()
                                }.onFailure { actionError = it.message }
                                saving = false
                            }
                        },
                        onDiscard = {
                            drafts.clear()
                            drafts.putAll(CourseAccessibilityReviewLogic.drafts(missingImages))
                            actionError = null
                        },
                    )
                }

                TextButton(onClick = onDismiss, modifier = Modifier.align(Alignment.End)) {
                    Text(L.text(R.string.mobile_common_cancel))
                }
            }
        }
    }
}

@Composable
private fun ImageEditorCard(
    image: CourseAccessibilityReviewLogic.MarkdownImageRef,
    draft: CourseAccessibilityReviewLogic.ImageAltDraft,
    onDraftChange: (CourseAccessibilityReviewLogic.ImageAltDraft) -> Unit,
    suggesting: Boolean,
    onSuggest: () -> Unit,
) {
    LmsCard {
        AsyncImage(
            model = image.src,
            contentDescription = null,
            modifier = Modifier
                .fillMaxWidth()
                .heightIn(max = 160.dp),
        )
        Text(L.format(R.string.mobile_courseSettings_accessibility_imageLine, image.line))
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(L.text(R.string.mobile_courseSettings_accessibility_decorativeLabel))
            Switch(
                checked = draft.decorative,
                onCheckedChange = { onDraftChange(draft.copy(decorative = it)) },
            )
        }
        OutlinedTextField(
            value = if (draft.decorative) "" else draft.alt,
            onValueChange = { onDraftChange(draft.copy(alt = it, decorative = false)) },
            label = { Text(L.text(R.string.mobile_courseSettings_accessibility_altTextLabel)) },
            modifier = Modifier.fillMaxWidth(),
            enabled = !draft.decorative,
            minLines = 2,
        )
        Button(
            onClick = onSuggest,
            enabled = !draft.decorative && !suggesting,
            modifier = Modifier.fillMaxWidth(),
        ) {
            if (suggesting) {
                CircularProgressIndicator(modifier = Modifier.size(18.dp))
            } else {
                Text(L.text(R.string.mobile_courseSettings_accessibility_suggestButton))
            }
        }
    }
}
