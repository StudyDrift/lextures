package com.lextures.android.features.courses.settings

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.Edit
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalLayoutDirection
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.LayoutDirection
import androidx.compose.ui.unit.dp
import androidx.compose.runtime.CompositionLocalProvider
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.OfflineBanner
import com.lextures.android.core.design.StalenessChip
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalLocalePreferences
import com.lextures.android.core.lms.AddGlossaryEntryBody
import com.lextures.android.core.lms.CourseGlossaryEntry
import com.lextures.android.core.lms.CourseGlossaryListResponse
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.CourseTranslationListItem
import com.lextures.android.core.lms.CourseTranslationListResponse
import com.lextures.android.core.lms.CourseTranslationsLogic
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.TranslationCoverage
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsSkeletonList
import kotlinx.coroutines.launch
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import java.util.UUID

private val translationsJson = Json { ignoreUnknownKeys = true }

private fun localeLabelResId(tag: String): Int? = when (CourseTranslationsLogic.localeLabelResKey(tag)) {
    "mobile_courseSettings_translations_locale_es" -> R.string.mobile_courseSettings_translations_locale_es
    "mobile_courseSettings_translations_locale_fr" -> R.string.mobile_courseSettings_translations_locale_fr
    "mobile_courseSettings_translations_locale_ar" -> R.string.mobile_courseSettings_translations_locale_ar
    "mobile_courseSettings_translations_locale_he" -> R.string.mobile_courseSettings_translations_locale_he
    "mobile_courseSettings_translations_locale_esES" -> R.string.mobile_courseSettings_translations_locale_esES
    "mobile_courseSettings_translations_locale_esMX" -> R.string.mobile_courseSettings_translations_locale_esMX
    "mobile_courseSettings_translations_locale_frFR" -> R.string.mobile_courseSettings_translations_locale_frFR
    "mobile_courseSettings_translations_locale_frCA" -> R.string.mobile_courseSettings_translations_locale_frCA
    "mobile_courseSettings_translations_locale_arSA" -> R.string.mobile_courseSettings_translations_locale_arSA
    "mobile_courseSettings_translations_locale_heIL" -> R.string.mobile_courseSettings_translations_locale_heIL
    else -> null
}

private fun localeDisplayName(
    context: android.content.Context,
    prefs: com.lextures.android.core.i18n.LocalePreferences,
    tag: String,
): String {
    val resId = localeLabelResId(tag)
    return if (resId != null) L.text(context, prefs, resId)
    else CourseTranslationsLogic.fallbackLocaleDisplayName(tag)
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun CourseTranslationsSettingsScreen(
    session: AuthSession,
    course: CourseSummary,
    offline: OfflineService,
) {
    val scope = rememberCoroutineScope()
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    val isOnline by offline.networkMonitor.isOnline.collectAsState()

    var locales by remember { mutableStateOf<List<TranslationCoverage>>(emptyList()) }
    var trackedLocales by remember {
        mutableStateOf(CourseTranslationsLogic.loadTrackedLocales(context, course.courseCode))
    }
    var selectedLocale by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }
    var loadError by remember { mutableStateOf<String?>(null) }
    var actionSuccess by remember { mutableStateOf<String?>(null) }
    var cacheLabel by remember { mutableStateOf<String?>(null) }
    var showAddLocale by remember { mutableStateOf(false) }

    fun localeName(tag: String): String = localeDisplayName(context, localePrefs, tag)

    fun reloadLocales(force: Boolean = false) {
        scope.launch {
            val token = session.accessToken.value ?: return@launch
            loading = locales.isEmpty()
            loadError = null
            trackedLocales = CourseTranslationsLogic.mergeTracked(
                CourseTranslationsLogic.loadTrackedLocales(context, course.courseCode),
                trackedLocales,
            )
            runCatching {
                val cached = offline.cachedFetch(
                    key = CourseTranslationsLogic.cacheKeyLocales(course.courseCode),
                    accessToken = token,
                    serializer = kotlinx.serialization.builtins.ListSerializer(TranslationCoverage.serializer()),
                ) {
                    LmsApi.fetchTranslationLocales(course.courseCode, token)
                }
                var enriched = cached.first
                if (isOnline) {
                    for (tag in trackedLocales) {
                        if (enriched.none { CourseTranslationsLogic.normalizeLocaleTag(it.targetLocale) == tag }) {
                            runCatching {
                                LmsApi.fetchTranslationCoverage(course.courseCode, tag, token)
                            }.onSuccess { cov -> enriched = enriched + cov }
                        }
                    }
                }
                locales = CourseTranslationsLogic.mergeLocales(enriched, trackedLocales)
                cacheLabel = cached.second?.takeIf { it.isStale(isOnline) }?.lastUpdatedLabel()
            }.onFailure { loadError = it.message }
            loading = false
        }
    }

    LaunchedEffect(course.courseCode) { reloadLocales() }

    if (selectedLocale != null) {
        CourseTranslationLocaleDetailScreen(
            session = session,
            course = course,
            offline = offline,
            targetLocale = selectedLocale!!,
            localeName = localeName(selectedLocale!!),
            onBack = {
                selectedLocale = null
                reloadLocales(force = true)
            },
        )
        return
    }

    Column(modifier = Modifier.fillMaxSize()) {
        LazyColumn(
            modifier = Modifier.weight(1f),
            verticalArrangement = Arrangement.spacedBy(12.dp),
            contentPadding = PaddingValues(16.dp),
        ) {
            if (!isOnline) item { OfflineBanner() }
            loadError?.let { msg -> item { LmsErrorBanner(message = msg) } }
            cacheLabel?.let { label -> item { StalenessChip(label = label) } }
            actionSuccess?.let { msg ->
                item {
                    LmsCard {
                        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                            Icon(Icons.Default.CheckCircle, contentDescription = null, tint = Color(0xFF0D9488))
                            Text(msg, fontWeight = FontWeight.SemiBold, color = Color(0xFF0D9488))
                        }
                    }
                }
            }

            if (loading) {
                item { LmsSkeletonList(count = 4) }
            } else {
                item {
                    LmsCard {
                        Text(
                            L.text(R.string.mobile_courseSettings_translations_introTitle),
                            fontWeight = FontWeight.SemiBold,
                        )
                        Text(L.text(R.string.mobile_courseSettings_translations_introDescription))
                    }
                }

                if (locales.isEmpty()) {
                    item {
                        LmsCard {
                            Text(
                                L.text(R.string.mobile_courseSettings_translations_emptyTitle),
                                fontWeight = FontWeight.SemiBold,
                            )
                            Text(L.text(R.string.mobile_courseSettings_translations_emptyMessage))
                        }
                    }
                } else {
                    item {
                        LmsCard {
                            Text(
                                L.text(R.string.mobile_courseSettings_translations_localesTitle),
                                fontWeight = FontWeight.SemiBold,
                            )
                            locales.forEachIndexed { index, locale ->
                                Row(
                                    modifier = Modifier
                                        .fillMaxWidth()
                                        .clickable { selectedLocale = locale.targetLocale }
                                        .padding(vertical = 8.dp),
                                    horizontalArrangement = Arrangement.SpaceBetween,
                                    verticalAlignment = Alignment.CenterVertically,
                                ) {
                                    Column(modifier = Modifier.weight(1f)) {
                                        Text(localeName(locale.targetLocale), fontWeight = FontWeight.Medium)
                                        Text(locale.targetLocale, color = Color.Gray)
                                        Text(
                                            L.format(
                                                R.string.mobile_courseSettings_translations_coverageValue,
                                                locale.percentInt,
                                                locale.translatedItems,
                                                locale.totalItems,
                                                localeName(locale.targetLocale),
                                            ),
                                            color = Color.Gray,
                                        )
                                    }
                                    Text(
                                        CourseTranslationsLogic.formatCoveragePercentOnly(locale.percent),
                                        fontWeight = FontWeight.Bold,
                                        color = Color(0xFF0D9488),
                                    )
                                }
                                if (index < locales.lastIndex) HorizontalDivider()
                            }
                        }
                    }
                }

                item {
                    Button(
                        onClick = { showAddLocale = true },
                        enabled = CourseTranslationsLogic.availableLocalesToAdd(locales).isNotEmpty(),
                        modifier = Modifier.fillMaxWidth(),
                    ) {
                        Icon(Icons.Default.Add, contentDescription = null)
                        Text(
                            L.text(R.string.mobile_courseSettings_translations_addLocale),
                            modifier = Modifier.padding(start = 8.dp),
                        )
                    }
                }
            }
        }
    }

    if (showAddLocale) {
        val options = CourseTranslationsLogic.availableLocalesToAdd(locales)
        AlertDialog(
            onDismissRequest = { showAddLocale = false },
            title = { Text(L.text(R.string.mobile_courseSettings_translations_addLocaleTitle)) },
            text = {
                Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
                    Text(L.text(R.string.mobile_courseSettings_translations_addLocaleConfirm))
                    options.forEach { option ->
                        TextButton(
                            onClick = {
                                showAddLocale = false
                                trackedLocales = CourseTranslationsLogic.trackLocale(option.tag, trackedLocales)
                                CourseTranslationsLogic.saveTrackedLocales(
                                    context,
                                    course.courseCode,
                                    trackedLocales,
                                )
                                locales = CourseTranslationsLogic.mergeLocales(locales, trackedLocales)
                                selectedLocale = option.tag
                                actionSuccess = L.format(
                                    context,
                                    localePrefs,
                                    R.string.mobile_courseSettings_translations_localeAdded,
                                    localeDisplayName(context, localePrefs, option.tag),
                                )
                            },
                            modifier = Modifier.fillMaxWidth(),
                        ) {
                            Text(localeDisplayName(context, localePrefs, option.tag))
                        }
                    }
                }
            },
            confirmButton = {
                TextButton(onClick = { showAddLocale = false }) {
                    Text(L.text(R.string.mobile_courseSettings_translations_cancel))
                }
            },
        )
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun CourseTranslationLocaleDetailScreen(
    session: AuthSession,
    course: CourseSummary,
    offline: OfflineService,
    targetLocale: String,
    localeName: String,
    onBack: () -> Unit,
) {
    val scope = rememberCoroutineScope()
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    val isOnline by offline.networkMonitor.isOnline.collectAsState()

    var coverage by remember { mutableStateOf<TranslationCoverage?>(null) }
    var items by remember { mutableStateOf<List<CourseTranslationListItem>>(emptyList()) }
    var glossary by remember { mutableStateOf<List<CourseGlossaryEntry>>(emptyList()) }
    var itemQuery by remember { mutableStateOf("") }
    var glossaryQuery by remember { mutableStateOf("") }
    var itemPage by remember { mutableIntStateOf(0) }
    var glossaryPage by remember { mutableIntStateOf(0) }
    var loading by remember { mutableStateOf(true) }
    var loadError by remember { mutableStateOf<String?>(null) }
    var actionError by remember { mutableStateOf<String?>(null) }
    var actionSuccess by remember { mutableStateOf<String?>(null) }
    var cacheLabel by remember { mutableStateOf<String?>(null) }
    var glossaryDraft by remember { mutableStateOf<CourseTranslationsLogic.GlossaryDraft?>(null) }
    var savingGlossary by remember { mutableStateOf(false) }

    val filteredItems = remember(items, itemQuery) {
        CourseTranslationsLogic.filterItems(items, itemQuery)
    }
    val filteredGlossary = remember(glossary, glossaryQuery) {
        CourseTranslationsLogic.filterGlossary(glossary, glossaryQuery)
    }
    val visibleItems = remember(filteredItems, itemPage) {
        CourseTranslationsLogic.paginatedItems(filteredItems, itemPage)
    }
    val visibleGlossary = remember(filteredGlossary, glossaryPage) {
        CourseTranslationsLogic.paginatedGlossary(filteredGlossary, glossaryPage)
    }

    LaunchedEffect(course.courseCode, targetLocale) {
        val token = session.accessToken.value ?: return@LaunchedEffect
        loading = true
        loadError = null
        runCatching {
            val listCached = offline.cachedFetch(
                key = CourseTranslationsLogic.cacheKeyLocaleDetail(course.courseCode, targetLocale),
                accessToken = token,
                serializer = CourseTranslationListResponse.serializer(),
            ) {
                LmsApi.fetchCourseTranslations(course.courseCode, targetLocale, token)
            }
            val glossaryCached = offline.cachedFetch(
                key = CourseTranslationsLogic.cacheKeyGlossary(course.courseCode, targetLocale),
                accessToken = token,
                serializer = CourseGlossaryListResponse.serializer(),
            ) {
                CourseGlossaryListResponse(
                    entries = LmsApi.fetchCourseGlossary(course.courseCode, targetLocale, accessToken = token),
                )
            }
            items = listCached.first.items
            coverage = listCached.first.coverage
            glossary = glossaryCached.first.entries
            itemPage = 0
            glossaryPage = 0
            cacheLabel = listCached.second?.takeIf { it.isStale(isOnline) }?.lastUpdatedLabel()
        }.onFailure { loadError = it.message }
        loading = false
    }

    Column(modifier = Modifier.fillMaxSize()) {
        Row(
            verticalAlignment = Alignment.CenterVertically,
            modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp),
        ) {
            IconButton(onClick = onBack) {
                Icon(
                    Icons.AutoMirrored.Filled.ArrowBack,
                    contentDescription = L.text(R.string.mobile_courseSettings_translations_backToLocales),
                )
            }
            Text(
                localeName,
                fontWeight = FontWeight.SemiBold,
                modifier = Modifier.padding(start = 4.dp),
            )
        }

        LazyColumn(
            modifier = Modifier.weight(1f),
            verticalArrangement = Arrangement.spacedBy(12.dp),
            contentPadding = PaddingValues(16.dp),
        ) {
            if (!isOnline) item { OfflineBanner() }
            loadError?.let { msg -> item { LmsErrorBanner(message = msg) } }
            actionError?.let { msg -> item { LmsErrorBanner(message = msg) } }
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
                        Text(localeName, fontWeight = FontWeight.SemiBold)
                        Text(targetLocale, color = Color.Gray)
                        Text(L.text(R.string.mobile_courseSettings_translations_localeDetailHint))
                    }
                }

                coverage?.let { cov ->
                    item {
                        LmsCard {
                            Text(
                                L.text(R.string.mobile_courseSettings_translations_coverageTitle),
                                fontWeight = FontWeight.SemiBold,
                            )
                            LinearProgressIndicator(
                                progress = { (cov.percent / 100.0).toFloat().coerceIn(0f, 1f) },
                                modifier = Modifier.fillMaxWidth().padding(vertical = 8.dp),
                            )
                            Text(
                                L.format(
                                    R.string.mobile_courseSettings_translations_coverageValue,
                                    cov.percentInt,
                                    cov.translatedItems,
                                    cov.totalItems,
                                    localeName,
                                ),
                                fontWeight = FontWeight.Bold,
                            )
                            Text(
                                L.format(
                                    R.string.mobile_courseSettings_translations_unpublishedCount,
                                    CourseTranslationsLogic.unpublishedCount(items),
                                ),
                                color = Color.Gray,
                            )
                        }
                    }
                }

                item {
                    LmsCard {
                        Text(
                            L.text(R.string.mobile_courseSettings_translations_itemsTitle),
                            fontWeight = FontWeight.SemiBold,
                        )
                        Text(L.text(R.string.mobile_courseSettings_translations_itemsReadOnlyHint))
                        OutlinedTextField(
                            value = itemQuery,
                            onValueChange = {
                                itemQuery = it
                                itemPage = 0
                            },
                            label = { Text(L.text(R.string.mobile_courseSettings_translations_searchItems)) },
                            modifier = Modifier.fillMaxWidth().padding(top = 8.dp),
                            singleLine = true,
                        )
                        if (visibleItems.isEmpty()) {
                            Text(
                                L.text(R.string.mobile_courseSettings_translations_itemsEmpty),
                                color = Color.Gray,
                                modifier = Modifier.padding(top = 8.dp),
                            )
                        } else {
                            visibleItems.forEachIndexed { index, item ->
                                Column(modifier = Modifier.padding(vertical = 6.dp)) {
                                    Text(
                                        item.title.ifBlank {
                                            L.text(R.string.mobile_courseSettings_translations_untitled)
                                        },
                                        fontWeight = FontWeight.Medium,
                                    )
                                    val statusRes = when (CourseTranslationsLogic.statusLabelResKey(item)) {
                                        "mobile_courseSettings_translations_status_published" ->
                                            R.string.mobile_courseSettings_translations_status_published
                                        "mobile_courseSettings_translations_status_draft" ->
                                            R.string.mobile_courseSettings_translations_status_draft
                                        else -> R.string.mobile_courseSettings_translations_status_missing
                                    }
                                    Text(
                                        L.text(statusRes),
                                        color = when {
                                            item.hasPublished == true -> Color(0xFF0D9488)
                                            item.hasDraft == true || item.isDraft == true -> Color(0xFFD97706)
                                            else -> Color.Gray
                                        },
                                    )
                                }
                                if (index < visibleItems.lastIndex) HorizontalDivider()
                            }
                            if (CourseTranslationsLogic.hasMoreItemPages(filteredItems, itemPage)) {
                                TextButton(onClick = { itemPage += 1 }) {
                                    Text(L.text(R.string.mobile_courseSettings_translations_loadMore))
                                }
                            }
                        }
                    }
                }

                item {
                    LmsCard {
                        Row(
                            modifier = Modifier.fillMaxWidth(),
                            horizontalArrangement = Arrangement.SpaceBetween,
                            verticalAlignment = Alignment.CenterVertically,
                        ) {
                            Text(
                                L.text(R.string.mobile_courseSettings_translations_glossaryTitle),
                                fontWeight = FontWeight.SemiBold,
                            )
                            TextButton(onClick = {
                                glossaryDraft = CourseTranslationsLogic.GlossaryDraft()
                            }) {
                                Icon(Icons.Default.Add, contentDescription = null)
                                Text(
                                    L.text(R.string.mobile_courseSettings_translations_addTerm),
                                    modifier = Modifier.padding(start = 4.dp),
                                )
                            }
                        }
                        OutlinedTextField(
                            value = glossaryQuery,
                            onValueChange = {
                                glossaryQuery = it
                                glossaryPage = 0
                            },
                            label = { Text(L.text(R.string.mobile_courseSettings_translations_searchGlossary)) },
                            modifier = Modifier.fillMaxWidth(),
                            singleLine = true,
                        )
                        if (visibleGlossary.isEmpty()) {
                            Text(
                                L.text(R.string.mobile_courseSettings_translations_glossaryEmpty),
                                color = Color.Gray,
                                modifier = Modifier.padding(top = 8.dp),
                            )
                        } else {
                            val rtl = CourseTranslationsLogic.isRTLLocale(targetLocale)
                            CompositionLocalProvider(
                                LocalLayoutDirection provides if (rtl) LayoutDirection.Rtl else LayoutDirection.Ltr,
                            ) {
                                visibleGlossary.forEachIndexed { index, entry ->
                                    Row(
                                        modifier = Modifier
                                            .fillMaxWidth()
                                            .clickable {
                                                glossaryDraft = CourseTranslationsLogic.draft(entry)
                                            }
                                            .padding(vertical = 8.dp),
                                        horizontalArrangement = Arrangement.SpaceBetween,
                                    ) {
                                        Column(modifier = Modifier.weight(1f)) {
                                            Text(entry.sourceTerm, fontWeight = FontWeight.Medium)
                                            Text(entry.targetTerm, color = Color.Gray)
                                        }
                                        Icon(Icons.Default.Edit, contentDescription = null)
                                    }
                                    if (index < visibleGlossary.lastIndex) HorizontalDivider()
                                }
                            }
                            if (CourseTranslationsLogic.hasMoreGlossaryPages(filteredGlossary, glossaryPage)) {
                                TextButton(onClick = { glossaryPage += 1 }) {
                                    Text(L.text(R.string.mobile_courseSettings_translations_loadMore))
                                }
                            }
                        }
                    }
                }
            }
        }
    }

    glossaryDraft?.let { draft ->
        val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
        var sourceTerm by remember(draft) { mutableStateOf(draft.sourceTerm) }
        var targetTerm by remember(draft) { mutableStateOf(draft.targetTerm) }

        ModalBottomSheet(
            onDismissRequest = { if (!savingGlossary) glossaryDraft = null },
            sheetState = sheetState,
        ) {
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(horizontal = 16.dp, vertical = 8.dp),
                verticalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                Text(
                    if (draft.isEditing) {
                        L.text(R.string.mobile_courseSettings_translations_editTerm)
                    } else {
                        L.text(R.string.mobile_courseSettings_translations_addTerm)
                    },
                    fontWeight = FontWeight.SemiBold,
                )
                OutlinedTextField(
                    value = sourceTerm,
                    onValueChange = { sourceTerm = it },
                    label = { Text(L.text(R.string.mobile_courseSettings_translations_sourceTerm)) },
                    modifier = Modifier.fillMaxWidth(),
                    singleLine = true,
                )
                CompositionLocalProvider(
                    LocalLayoutDirection provides if (CourseTranslationsLogic.isRTLLocale(targetLocale)) {
                        LayoutDirection.Rtl
                    } else {
                        LayoutDirection.Ltr
                    },
                ) {
                    OutlinedTextField(
                        value = targetTerm,
                        onValueChange = { targetTerm = it },
                        label = { Text(L.text(R.string.mobile_courseSettings_translations_targetTerm)) },
                        modifier = Modifier.fillMaxWidth(),
                        singleLine = true,
                    )
                }
                Text(L.text(R.string.mobile_courseSettings_translations_glossaryEditorHint), color = Color.Gray)
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.End,
                ) {
                    TextButton(
                        onClick = { glossaryDraft = null },
                        enabled = !savingGlossary,
                    ) {
                        Text(L.text(R.string.mobile_courseSettings_translations_cancel))
                    }
                    Button(
                        onClick = {
                            val updated = CourseTranslationsLogic.GlossaryDraft(
                                id = draft.id,
                                sourceTerm = sourceTerm,
                                targetTerm = targetTerm,
                            )
                            when (CourseTranslationsLogic.validateGlossaryDraft(updated)) {
                                CourseTranslationsLogic.GlossaryValidation.SourceRequired -> {
                                    actionError = L.text(
                                        context,
                                        localePrefs,
                                        R.string.mobile_courseSettings_translations_validation_sourceRequired,
                                    )
                                    return@Button
                                }
                                CourseTranslationsLogic.GlossaryValidation.TargetRequired -> {
                                    actionError = L.text(
                                        context,
                                        localePrefs,
                                        R.string.mobile_courseSettings_translations_validation_targetRequired,
                                    )
                                    return@Button
                                }
                                CourseTranslationsLogic.GlossaryValidation.Ok -> Unit
                            }
                            scope.launch {
                                val token = session.accessToken.value ?: return@launch
                                savingGlossary = true
                                actionError = null
                                actionSuccess = null
                                runCatching {
                                    val body = CourseTranslationsLogic.buildGlossaryBody(updated, targetLocale)
                                    offline.enqueueMutation(
                                        method = "POST",
                                        path = CourseTranslationsLogic.glossaryPath(course.courseCode),
                                        bodyJson = translationsJson.encodeToString(
                                            AddGlossaryEntryBody.serializer(),
                                            body,
                                        ),
                                        label = L.text(
                                            context,
                                            localePrefs,
                                            R.string.mobile_courseSettings_translations_glossarySaveLabel,
                                        ),
                                        accessToken = token,
                                        idempotencyKey = CourseTranslationsLogic.glossaryIdempotencyKey(
                                            course.courseCode,
                                            targetLocale,
                                            body.sourceTerm,
                                        ),
                                    )
                                    val optimistic = CourseGlossaryEntry(
                                        id = draft.id ?: UUID.randomUUID().toString(),
                                        sourceTerm = body.sourceTerm,
                                        targetTerm = body.targetTerm,
                                        sourceLocale = body.sourceLocale,
                                        targetLocale = body.targetLocale,
                                    )
                                    glossary = CourseTranslationsLogic.upsertGlossaryEntry(optimistic, glossary)
                                    if (isOnline) {
                                        runCatching {
                                            LmsApi.fetchCourseGlossary(
                                                course.courseCode,
                                                targetLocale,
                                                accessToken = token,
                                            )
                                        }.onSuccess { glossary = it }
                                    }
                                    glossaryDraft = null
                                    actionSuccess = L.text(
                                        context,
                                        localePrefs,
                                        R.string.mobile_courseSettings_translations_glossarySaved,
                                    )
                                }.onFailure { actionError = it.message }
                                savingGlossary = false
                            }
                        },
                        enabled = !savingGlossary &&
                            sourceTerm.trim().isNotEmpty() &&
                            targetTerm.trim().isNotEmpty(),
                    ) {
                        if (savingGlossary) {
                            CircularProgressIndicator(modifier = Modifier.padding(end = 8.dp))
                        }
                        Text(L.text(R.string.mobile_courseSettings_translations_saveTerm))
                    }
                }
            }
        }
    }
}
