package com.lextures.android.features.reading

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.MenuBook
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.FilterChip
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
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalLocalePreferences
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.LibraryBook
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.PostReadingLogBody
import com.lextures.android.core.lms.ReadingLogEntry
import com.lextures.android.core.lms.ReadingLogic
import com.lextures.android.core.offline.OfflineCacheKey
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.core.design.OfflineBanner
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsSkeletonList
import kotlinx.coroutines.launch
import kotlinx.serialization.json.Json
import kotlinx.serialization.serializer

@Composable
fun ReadingDashboardScreen(
    session: AuthSession,
    onOpenBookClub: (CourseSummary) -> Unit,
    onOpenLibrary: (String) -> Unit,
    initialLogBook: LibraryBook? = null,
    onConsumeInitialLogBook: () -> Unit = {},
    modifier: Modifier = Modifier,
) {
    val context = androidx.compose.ui.platform.LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    val scope = rememberCoroutineScope()
    val accessToken by session.accessToken.collectAsState()
    val offline = remember { OfflineService.get(context) }
    val isOnline by offline.networkMonitor.isOnline.collectAsState()

    var entries by remember { mutableStateOf<List<ReadingLogEntry>>(emptyList()) }
    var bookClubCourses by remember { mutableStateOf<List<CourseSummary>>(emptyList()) }
    var orgId by remember { mutableStateOf<String?>(null) }
    var loginStreakDays by remember { mutableStateOf(0) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }
    var showLogDialog by remember { mutableStateOf(false) }
    var logDraft by remember { mutableStateOf(LogReadingDraft()) }
    var saving by remember { mutableStateOf(false) }
    var logError by remember { mutableStateOf<String?>(null) }

    suspend fun load(token: String) {
        loading = true
        errorMessage = null
        try {
            val entriesResult = offline.cachedFetch(
                key = OfflineCacheKey.readingLog(),
                accessToken = token,
                serializer = serializer<List<ReadingLogEntry>>(),
            ) { LmsApi.fetchReadingLogEntries(accessToken = token) }
            entries = entriesResult.first
            val courses = LmsApi.fetchCourses(token)
            orgId = ReadingLogic.resolveOrgId(courses)
            bookClubCourses = ReadingLogic.bookClubCourses(courses)
            runCatching { LmsApi.fetchStudyStats(token) }.getOrNull()?.let {
                loginStreakDays = it.loginStreakDays
            }
        } catch (_: Exception) {
            errorMessage = L.text(context, localePrefs, R.string.mobile_reading_error_load)
        } finally {
            loading = false
        }
    }

    LaunchedEffect(accessToken) {
        val token = accessToken ?: return@LaunchedEffect
        load(token)
    }

    LaunchedEffect(initialLogBook) {
        initialLogBook?.let { book ->
            logDraft = LogReadingDraft(
                bookId = book.id,
                bookTitle = book.title,
                logDate = ReadingLogic.todayIso(),
            )
            logError = null
            showLogDialog = true
            onConsumeInitialLogBook()
        }
    }

    Column(
        modifier = modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        if (!isOnline) {
            OfflineBanner()
        }
        errorMessage?.let { LmsErrorBanner(message = it) }

        Row(horizontalArrangement = Arrangement.spacedBy(12.dp), modifier = Modifier.fillMaxWidth()) {
            StatCard(title = readingWeeklyPages(), value = "${ReadingLogic.weeklyPages(entries)}", modifier = Modifier.weight(1f))
            StatCard(
                title = readingReadingStreak(),
                value = readingStreakDays(ReadingLogic.readingStreakDays(entries)),
                modifier = Modifier.weight(1f),
            )
            if (loginStreakDays > 0) {
                StatCard(
                    title = readingLoginStreak(),
                    value = readingStreakDays(loginStreakDays),
                    modifier = Modifier.weight(1f),
                )
            }
        }

        Button(
            onClick = {
                logDraft = LogReadingDraft(logDate = ReadingLogic.todayIso())
                logError = null
                showLogDialog = true
            },
            modifier = Modifier.fillMaxWidth(),
        ) {
            Text(readingLogAction())
        }

        orgId?.let { id ->
            LmsCard(modifier = Modifier.fillMaxWidth().clickable { onOpenLibrary(id) }) {
                Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                    Column {
                        Text(readingLibraryTitle(), fontWeight = FontWeight.SemiBold, color = textPrimary())
                        Text(readingLibraryHint(), color = textSecondary())
                    }
                }
            }
        }

        Text(readingBookClubTitle(), fontWeight = FontWeight.Bold, color = textPrimary())
        Text(readingBookClubHint(), color = textSecondary())
        if (bookClubCourses.isEmpty()) {
            Text(readingBookClubEmpty(), color = textSecondary())
        } else {
            bookClubCourses.forEach { course ->
                LmsCard(modifier = Modifier.fillMaxWidth().clickable { onOpenBookClub(course) }) {
                    Text(course.displayTitle, fontWeight = FontWeight.SemiBold, color = textPrimary())
                }
            }
        }

        Text(readingHistoryTitle(), fontWeight = FontWeight.Bold, color = textPrimary())
        when {
            loading && entries.isEmpty() -> LmsSkeletonList(count = 3)
            entries.isEmpty() -> LmsEmptyState(
                icon = Icons.Default.MenuBook,
                title = readingHistoryEmptyTitle(),
                message = readingHistoryEmptyMessage(),
            )
            else -> entries.forEach { entry ->
                LmsCard(modifier = Modifier.fillMaxWidth()) {
                    Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
                        Text(entry.bookTitle ?: readingUnknownBook(), fontWeight = FontWeight.SemiBold)
                        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                            Text(entry.logDate, color = textSecondary())
                            entry.pagesRead?.let { Text(readingPagesCount(it), color = textSecondary()) }
                        }
                        entry.reflection?.takeIf { it.isNotBlank() }?.let {
                            Text(it, color = textSecondary(), maxLines = 2)
                        }
                    }
                }
            }
        }
    }

    if (showLogDialog) {
        LogReadingDialog(
            draft = logDraft,
            saving = saving,
            errorMessage = logError,
            onDismiss = { showLogDialog = false },
            onSave = { draft ->
                val token = accessToken ?: return@LogReadingDialog
                if (!ReadingLogic.logEntryValid(draft.bookTitle, draft.bookId, draft.logDate)) {
                    logError = L.text(context, localePrefs, R.string.mobile_reading_error_validation)
                    return@LogReadingDialog
                }
                scope.launch {
                    saving = true
                    logError = null
                    try {
                        val body = PostReadingLogBody(
                            bookId = draft.bookId,
                            bookTitle = draft.bookTitle.trim().takeIf { it.isNotEmpty() },
                            logDate = draft.logDate,
                            pagesRead = draft.pagesRead.trim().toIntOrNull(),
                            reflection = draft.reflection.trim().takeIf { it.isNotEmpty() },
                        )
                        if (isOnline) {
                            LmsApi.createReadingLogEntry(body, token)
                        } else {
                            offline.enqueueMutation(
                                method = "POST",
                                path = "/api/v1/me/reading-log",
                                bodyJson = Json.encodeToString(PostReadingLogBody.serializer(), body),
                                label = L.text(context, localePrefs, R.string.mobile_reading_logSave),
                                accessToken = token,
                            )
                        }
                        showLogDialog = false
                        load(token)
                    } catch (_: Exception) {
                        logError = L.text(context, localePrefs, R.string.mobile_reading_error_save)
                    } finally {
                        saving = false
                    }
                }
            },
        )
    }
}

@Composable
fun LeveledLibraryScreen(
    session: AuthSession,
    orgId: String,
    onLogBook: (LibraryBook) -> Unit,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    val accessToken by session.accessToken.collectAsState()
    val offline = remember { OfflineService.get(context) }

    var gradeBand by remember { mutableStateOf("") }
    var books by remember { mutableStateOf<List<LibraryBook>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var selectedBook by remember { mutableStateOf<LibraryBook?>(null) }


    LaunchedEffect(accessToken, orgId, gradeBand) {
        val token = accessToken ?: return@LaunchedEffect
        loading = true
        errorMessage = null
        try {
            val result = offline.cachedFetch(
                key = OfflineCacheKey.libraryBooks(orgId, gradeBand),
                accessToken = token,
                serializer = serializer<List<LibraryBook>>(),
            ) {
                LmsApi.fetchLibraryBooks(
                    orgId = orgId,
                    gradeBand = gradeBand.takeIf { it.isNotEmpty() },
                    accessToken = token,
                )
            }
            books = result.first.sortedBy { it.title.lowercase() }
        } catch (_: Exception) {
            errorMessage = L.text(context, localePrefs, R.string.mobile_reading_error_load)
        } finally {
            loading = false
        }
    }

    Column(modifier = modifier.fillMaxSize().padding(16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
        errorMessage?.let { LmsErrorBanner(message = it) }
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            ReadingLogic.GRADE_BANDS.forEach { band ->
                FilterChip(
                    selected = gradeBand == band,
                    onClick = { gradeBand = band },
                    label = { Text(if (band.isEmpty()) readingAllLevels() else band) },
                )
            }
        }
        when {
            loading && books.isEmpty() -> LmsSkeletonList(count = 4)
            books.isEmpty() -> LmsEmptyState(
                icon = Icons.Default.MenuBook,
                title = readingLibraryEmptyTitle(),
                message = readingLibraryEmptyMessage(),
            )
            else -> LazyColumn(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                items(books, key = { it.id }) { book ->
                    LmsCard(modifier = Modifier.fillMaxWidth().clickable { selectedBook = book }) {
                        Column {
                            Text(book.title, fontWeight = FontWeight.SemiBold, color = textPrimary())
                            ReadingLogic.bookSubtitle(book)?.let { Text(it, color = textSecondary()) }
                        }
                    }
                }
            }
        }
    }

    selectedBook?.let { book ->
        AlertDialog(
            onDismissRequest = { selectedBook = null },
            title = { Text(book.title) },
            text = {
                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    book.author?.let { Text(it) }
                    ReadingLogic.formatLexile(book.lexileLevel)?.let { Text(it) }
                    book.summary?.takeIf { it.isNotBlank() }?.let { Text(it, color = textSecondary()) }
                    book.coverUrl?.takeIf { it.isNotBlank() }?.let { cover ->
                        TextButton(onClick = {
                            runCatching {
                                val intent = android.content.Intent(
                                    android.content.Intent.ACTION_VIEW,
                                    android.net.Uri.parse(cover),
                                )
                                context.startActivity(intent)
                            }
                        }) { Text(readingPreviewOpen()) }
                    }
                }
            },
            confirmButton = {
                Button(onClick = {
                    selectedBook = null
                    onLogBook(book)
                }) { Text(readingLogThisBook()) }
            },
            dismissButton = {
                TextButton(onClick = { selectedBook = null }) {
                    Text(L.text(context, LocalLocalePreferences.current, R.string.mobile_ia_close))
                }
            },
        )
    }
}

@Composable
private fun StatCard(title: String, value: String, modifier: Modifier = Modifier) {
    LmsCard(modifier = modifier) {
        Column {
            Text(title, color = textSecondary())
            Text(value, fontWeight = FontWeight.Bold, color = textPrimary())
        }
    }
}

private data class LogReadingDraft(
    val bookId: String? = null,
    val bookTitle: String = "",
    val logDate: String = ReadingLogic.todayIso(),
    val pagesRead: String = "",
    val reflection: String = "",
)

@Composable
private fun LogReadingDialog(
    draft: LogReadingDraft,
    saving: Boolean,
    errorMessage: String?,
    onDismiss: () -> Unit,
    onSave: (LogReadingDraft) -> Unit,
) {
    var local by remember(draft) { mutableStateOf(draft) }
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(readingLogTitle()) },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                errorMessage?.let { LmsErrorBanner(message = it) }
                OutlinedTextField(
                    value = local.bookTitle,
                    onValueChange = { local = local.copy(bookTitle = it) },
                    label = { Text(readingLogBookTitle()) },
                    placeholder = { Text(readingLogBookPlaceholder()) },
                    enabled = local.bookId == null,
                    modifier = Modifier.fillMaxWidth(),
                )
                OutlinedTextField(
                    value = local.logDate,
                    onValueChange = { local = local.copy(logDate = it) },
                    label = { Text(readingLogDate()) },
                    modifier = Modifier.fillMaxWidth(),
                )
                OutlinedTextField(
                    value = local.pagesRead,
                    onValueChange = { local = local.copy(pagesRead = it) },
                    label = { Text(readingLogPages()) },
                    placeholder = { Text(readingLogPagesPlaceholder()) },
                    modifier = Modifier.fillMaxWidth(),
                )
                OutlinedTextField(
                    value = local.reflection,
                    onValueChange = { local = local.copy(reflection = it) },
                    label = { Text(readingLogReflection()) },
                    placeholder = { Text(readingLogReflectionPlaceholder()) },
                    modifier = Modifier.fillMaxWidth(),
                )
            }
        },
        confirmButton = {
            Button(
                onClick = { onSave(local) },
                enabled = !saving && ReadingLogic.logEntryValid(local.bookTitle, local.bookId, local.logDate),
            ) {
                if (saving) CircularProgressIndicator() else Text(readingLogSave())
            }
        },
        dismissButton = {
            val ctx = LocalContext.current
            TextButton(onClick = onDismiss) {
                Text(L.text(ctx, LocalLocalePreferences.current, R.string.mobile_ia_close))
            }
        },
    )
}