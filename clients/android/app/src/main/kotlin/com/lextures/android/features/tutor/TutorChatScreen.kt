package com.lextures.android.features.tutor

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.Delete
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.i18n.LocalePreferences
import androidx.compose.material3.Button
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.OfflineBanner
import com.lextures.android.core.design.cardBackground
import com.lextures.android.core.design.sceneBackground
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary

import com.lextures.android.core.lms.CourseStructureItem
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.NotebookRagNotebookInput
import com.lextures.android.core.lms.NotebookRagQueryBody
import com.lextures.android.core.lms.TutorDisplayMessage
import com.lextures.android.core.lms.TutorLogic
import com.lextures.android.core.lms.TutorStreamClient
import com.lextures.android.core.lms.TutorStreamEvent
import com.lextures.android.core.navigation.MobilePlatformFeatures
import com.lextures.android.core.notebook.NotebookStore
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.features.home.HomeShellState
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.notebooks.NotebookContentView
import kotlinx.coroutines.Job
import kotlinx.coroutines.flow.collect
import kotlinx.coroutines.launch

sealed class TutorChatMode {
    data class Course(val course: CourseSummary, val item: CourseStructureItem? = null) : TutorChatMode()
    data object AskAi : TutorChatMode()
}

@Composable
fun TutorChatScreen(
    session: AuthSession,
    mode: TutorChatMode,
    shell: HomeShellState?,
    onClose: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    val context = LocalContext.current
    val localePrefs = remember { LocalePreferences(context.applicationContext) }
    val offline = remember { OfflineService.get(context) }
    val isOnline by offline.networkMonitor.isOnline.collectAsState()
    val scope = rememberCoroutineScope()
    val platform = shell?.platformFeatures ?: MobilePlatformFeatures()
    val persistentEnabled = platform.ffPersistentTutor
    val ragEnabled = platform.ragNotebookEnabled
    val studyBuddyEnabled = platform.aiStudyBuddyEnabled

    val messages = remember { mutableStateListOf<TutorDisplayMessage>() }
    var input by remember { mutableStateOf("") }
    var streamingText by remember { mutableStateOf("") }
    var streaming by remember { mutableStateOf(false) }
    var loading by remember { mutableStateOf(true) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var showDisclosure by remember { mutableStateOf(false) }
    var tokensUsed by remember { mutableStateOf(0) }
    var tokenLimit by remember { mutableStateOf(0) }
    var activeSessionId by remember { mutableStateOf<String?>(null) }
    var studyBuddySessionId by remember { mutableStateOf<String?>(null) }
    var selectedCourse by remember { mutableStateOf<CourseSummary?>(null) }
    var courses by remember { mutableStateOf<List<CourseSummary>>(emptyList()) }
    var sentContext by remember { mutableStateOf(false) }
    val streamClient = remember { TutorStreamClient() }
    var streamJob by remember { mutableStateOf<Job?>(null) }
    val listState = rememberLazyListState()

    val courseCode = when (mode) {
        is TutorChatMode.Course -> mode.course.courseCode
        TutorChatMode.AskAi -> selectedCourse?.courseCode
    }

    LaunchedEffect(messages.size, streamingText) {
        val index = if (streamingText.isNotEmpty()) messages.size else (messages.size - 1).coerceAtLeast(0)
        if (index >= 0) listState.animateScrollToItem(index)
    }

    suspend fun loadCourseTutor(course: CourseSummary, token: String) {
        sentContext = false
        try {
            if (persistentEnabled) {
                var list = LmsApi.fetchTutorSessions(course.courseCode, token)
                if (list.isEmpty()) {
                    list = listOf(LmsApi.createTutorSession(course.courseCode, token))
                }
                activeSessionId = list.firstOrNull()?.id
                val sessionId = activeSessionId
                if (sessionId != null) {
                    val detail = LmsApi.fetchTutorSession(course.courseCode, sessionId, token)
                    messages.clear()
                    messages.addAll(
                        detail.messages
                            .filter { it.role != "system" }
                            .map {
                                TutorDisplayMessage(
                                    id = it.id ?: java.util.UUID.randomUUID().toString(),
                                    role = it.role,
                                    content = it.content,
                                    citations = it.citations.orEmpty(),
                                )
                            },
                    )
                }
            } else {
                val conv = LmsApi.fetchTutorConversation(course.courseCode, token)
                tokensUsed = conv.tokensUsed
                tokenLimit = conv.tokenLimit
                messages.clear()
                messages.addAll(
                    conv.messages.map {
                        TutorDisplayMessage(
                            role = it.role,
                            content = it.content,
                            citations = it.citations.orEmpty(),
                        )
                    },
                )
            }
        } catch (e: Exception) {
            errorMessage = tutorMapError(context, localePrefs, e)
        }
    }

    LaunchedEffect(accessToken, mode) {
        val token = accessToken ?: return@LaunchedEffect
        loading = true
        errorMessage = null
        showDisclosure = !context
            .getSharedPreferences("lextures.tutor", android.content.Context.MODE_PRIVATE)
            .getBoolean(TutorLogic.disclosureStorageKey(courseCode), false)
        when (mode) {
            TutorChatMode.AskAi -> {
                try {
                    courses = LmsApi.fetchCourses(token)
                    val store = NotebookStore(context, token)
                    val hasNotes = store.allCourseCodes().any { store.load(it).previewText.isNotBlank() }
                    if (ragEnabled && hasNotes) {
                        selectedCourse = null
                    } else {
                        selectedCourse = courses.firstOrNull { it.isAiTutorEnabled }
                        selectedCourse?.let { loadCourseTutor(it, token) }
                    }
                } catch (e: Exception) {
                    errorMessage = tutorMapError(context, localePrefs, e)
                }
            }
            is TutorChatMode.Course -> loadCourseTutor(mode.course, token)
        }
        loading = false
    }

    fun notebookInputs(token: String): List<NotebookRagNotebookInput> {
        val store = NotebookStore(context, token)
        return store.allCourseCodes().mapNotNull { code ->
            val notebook = store.load(code)
            val body = notebook.previewText
            if (body.isBlank()) return@mapNotNull null
            val title = notebook.courseTitle
                ?: if (code == NotebookStore.GLOBAL_KEY) NotebookStore.GLOBAL_TITLE else code
            NotebookRagNotebookInput(courseCode = code, courseTitle = title, markdown = body)
        }
    }

    fun stopStreaming() {
        streamJob?.cancel()
        streamClient.cancel()
        if (streamingText.isNotBlank()) {
            messages.add(TutorDisplayMessage(role = "assistant", content = streamingText))
        }
        streamingText = ""
        streaming = false
    }

    fun sendMessage() {
        val token = accessToken ?: return
        val raw = input.trim()
        if (raw.isEmpty()) return
        if (!isOnline) {
            errorMessage = tutorOffline(context, localePrefs)
            return
        }
        if (mode is TutorChatMode.AskAi && ragEnabled && selectedCourse == null) {
            val notebooks = notebookInputs(token)
            if (notebooks.isEmpty()) {
                errorMessage = tutorNoNotebooks(context, localePrefs)
                return
            }
            input = ""
            messages.add(TutorDisplayMessage(role = "user", content = raw))
            scope.launch {
                loading = true
                try {
                    val response = LmsApi.queryNotebooks(
                        NotebookRagQueryBody(question = raw, notebooks = notebooks),
                        token,
                    )
                    messages.add(TutorDisplayMessage(role = "assistant", content = response.answerMarkdown))
                } catch (e: Exception) {
                    errorMessage = tutorMapError(context, localePrefs, e)
                } finally {
                    loading = false
                }
            }
            return
        }

        val code = courseCode ?: return
        val includeContext: Boolean
        val itemTitle: String?
        val itemKind: String?
        if (mode is TutorChatMode.Course) {
            includeContext = !sentContext
            itemTitle = mode.item?.title
            itemKind = mode.item?.kind
            sentContext = true
        } else {
            includeContext = false
            itemTitle = null
            itemKind = null
        }
        val text = TutorLogic.messageWithContext(raw, itemTitle, itemKind, includeContext)
        input = ""
        errorMessage = null
        streaming = true
        streamingText = ""
        messages.add(TutorDisplayMessage(role = "user", content = raw))

        streamJob = scope.launch {
            var fullText = ""
            var citations = emptyList<com.lextures.android.core.lms.TutorCitation>()
            try {
                val flow = when {
                    studyBuddyEnabled && mode is TutorChatMode.AskAi ->
                        LmsApi.studyBuddyMessageStream(code, text, studyBuddySessionId, token, streamClient)
                    persistentEnabled && activeSessionId != null ->
                        LmsApi.tutorSessionMessageStream(code, activeSessionId!!, text, token, streamClient)
                    else ->
                        LmsApi.tutorMessageStream(code, text, token, streamClient)
                }
                flow.collect { event ->
                    when (event) {
                        is TutorStreamEvent.Content -> {
                            fullText += event.text
                            streamingText = fullText
                        }
                        is TutorStreamEvent.Error -> errorMessage = event.message
                        is TutorStreamEvent.Done -> {
                            citations = event.citations
                            studyBuddySessionId = event.sessionId ?: studyBuddySessionId
                        }
                    }
                }
                val answer = fullText.ifBlank { streamingText }
                if (answer.isNotBlank()) {
                    messages.add(TutorDisplayMessage(role = "assistant", content = answer, citations = citations))
                }
            } catch (e: Exception) {
                if (streamingText.isNotBlank()) {
                    messages.add(TutorDisplayMessage(role = "assistant", content = streamingText))
                }
                errorMessage = tutorMapError(context, localePrefs, e)
            } finally {
                streamingText = ""
                streaming = false
            }
        }
    }

    Column(
        modifier = modifier
            .fillMaxSize()
            .background(sceneBackground()),
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 8.dp, vertical = 4.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            TextButton(onClick = onClose) { Text(tutorClose()) }
            Text(
                text = when (mode) {
                    is TutorChatMode.Course -> tutorTitle()
                    TutorChatMode.AskAi -> tutorAskAi()
                },
                modifier = Modifier.weight(1f),
                fontWeight = FontWeight.SemiBold,
                color = textPrimary(),
            )
            if (mode is TutorChatMode.Course && persistentEnabled) {
                IconButton(onClick = {
                    val token = accessToken ?: return@IconButton
                    scope.launch {
                        loading = true
                        try {
                            val created = LmsApi.createTutorSession(mode.course.courseCode, token)
                            activeSessionId = created.id
                            messages.clear()
                            sentContext = false
                        } catch (e: Exception) {
                            errorMessage = tutorMapError(context, localePrefs, e)
                        } finally {
                            loading = false
                        }
                    }
                }) {
                    Icon(Icons.Default.Add, contentDescription = tutorNewConversation())
                }
            }
            if (courseCode != null && !(mode is TutorChatMode.AskAi && ragEnabled && selectedCourse == null)) {
                IconButton(onClick = {
                    val token = accessToken ?: return@IconButton
                    scope.launch {
                        try {
                            if (persistentEnabled && activeSessionId != null) {
                                LmsApi.deleteTutorSession(courseCode!!, activeSessionId!!, token)
                                val created = LmsApi.createTutorSession(
                                    if (mode is TutorChatMode.Course) mode.course.courseCode else courseCode!!,
                                    token,
                                )
                                activeSessionId = created.id
                                messages.clear()
                            } else {
                                LmsApi.resetTutorConversation(courseCode!!, token)
                                messages.clear()
                            }
                            sentContext = false
                        } catch (e: Exception) {
                            errorMessage = tutorMapError(context, localePrefs, e)
                        }
                    }
                }) {
                    Icon(Icons.Default.Delete, contentDescription = tutorReset())
                }
            }
        }
        HorizontalDivider()
        if (!isOnline) OfflineBanner(modifier = Modifier.fillMaxWidth())
        if (showDisclosure) {
            LmsCard(modifier = Modifier.padding(16.dp)) {
                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    Text(tutorDisclosureTitle(), fontWeight = FontWeight.SemiBold, color = textPrimary())
                    Text(tutorDisclosureBody(), color = textSecondary())
                    TextButton(onClick = {
                        context.getSharedPreferences("lextures.tutor", android.content.Context.MODE_PRIVATE)
                            .edit()
                            .putBoolean(TutorLogic.disclosureStorageKey(courseCode), true)
                            .apply()
                        showDisclosure = false
                    }) { Text(tutorDisclosureAccept()) }
                }
            }
        }
        if (tokenLimit > 0) {
            Text(
                text = tutorTokenBudget(tokensUsed, tokenLimit),
                modifier = Modifier.padding(horizontal = 16.dp, vertical = 8.dp),
                color = textSecondary(),
            )
        }
        errorMessage?.let { LmsErrorBanner(message = it, modifier = Modifier.padding(16.dp)) }

        if (mode is TutorChatMode.AskAi && selectedCourse == null && !loading) {
            LazyColumn(
                modifier = Modifier.weight(1f),
                contentPadding = androidx.compose.foundation.layout.PaddingValues(16.dp),
                verticalArrangement = Arrangement.spacedBy(10.dp),
            ) {
                item {
                    Text(tutorAskAiCourseHint(), color = textSecondary())
                    if (ragEnabled) Text(tutorAskAiNotebookHint(), color = textSecondary())
                }
                items(courses.filter { it.isAiTutorEnabled || studyBuddyEnabled }) { course ->
                    TextButton(onClick = {
                        selectedCourse = course
                        scope.launch {
                            loading = true
                            accessToken?.let { loadCourseTutor(course, it) }
                            loading = false
                        }
                    }) {
                        LmsCard(modifier = Modifier.fillMaxWidth()) {
                            Text(course.displayTitle, color = textPrimary(), fontWeight = FontWeight.SemiBold)
                        }
                    }
                }
            }
        } else {
            LazyColumn(
                state = listState,
                modifier = Modifier.weight(1f),
                contentPadding = androidx.compose.foundation.layout.PaddingValues(16.dp),
                verticalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                if (loading && messages.isEmpty()) {
                    item {
                        Box(Modifier.fillMaxWidth(), contentAlignment = Alignment.Center) {
                            CircularProgressIndicator()
                        }
                    }
                }
                items(messages, key = { it.id }) { message ->
                    TutorMessageBubble(message)
                }
                if (streamingText.isNotBlank()) {
                    item {
                        TutorMessageBubble(
                            TutorDisplayMessage(role = "assistant", content = streamingText, isStreaming = true),
                        )
                    }
                }
            }
        }

        Row(
            modifier = Modifier
                .fillMaxWidth()
                .background(sceneBackground())
                .padding(16.dp),
            horizontalArrangement = Arrangement.spacedBy(10.dp),
            verticalAlignment = Alignment.Bottom,
        ) {
            OutlinedTextField(
                value = input,
                onValueChange = { input = it },
                modifier = Modifier.weight(1f),
                placeholder = { Text(tutorPlaceholder()) },
                enabled = isOnline && !loading,
                maxLines = 5,
            )
            if (streaming) {
                OutlinedButton(onClick = { stopStreaming() }) { Text(tutorStop()) }
            } else {
                Button(
                    onClick = { sendMessage() },
                    enabled = input.isNotBlank() && isOnline && !loading,
                ) { Text(tutorSend()) }
            }
        }
    }
}

@Composable
private fun TutorMessageBubble(message: TutorDisplayMessage) {
    val isUser = message.role == "user"
    Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = if (isUser) Arrangement.End else Arrangement.Start) {
        Column(
            modifier = Modifier
                .fillMaxWidth(0.85f)
                .background(
                    if (isUser) LexturesColors.Primary else cardBackground(),
                    shape = androidx.compose.foundation.shape.RoundedCornerShape(14.dp),
                )
                .padding(12.dp),
        ) {
            if (isUser) {
                Text(message.content, color = androidx.compose.ui.graphics.Color.White)
            } else {
                NotebookContentView(
                    markdown = message.content,
                    onToggleTask = {},
                    onEditTaskDue = {},
                )
                message.citations.forEach { citation ->
                    Text(
                        text = citation.title ?: tutorSource(),
                        color = LexturesColors.Primary,
                        modifier = Modifier.padding(top = 4.dp),
                    )
                }
            }
        }
    }
}