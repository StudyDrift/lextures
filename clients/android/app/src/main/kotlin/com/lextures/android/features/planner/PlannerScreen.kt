package com.lextures.android.features.planner

import android.content.ClipData
import android.content.ClipboardManager
import android.content.Context
import android.content.Intent
import android.provider.CalendarContract
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.material3.Icon
import androidx.compose.ui.Alignment
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.IconButton
import androidx.compose.material3.Button
import androidx.activity.compose.BackHandler
import com.lextures.android.features.evaluations.EvaluationFormScreen
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Tab
import androidx.compose.material3.TabRow
import androidx.compose.material3.Text
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
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.lms.CalendarTokenCreated
import com.lextures.android.core.lms.CalendarTokenInfo
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.PlannerCalendarEvent
import com.lextures.android.core.lms.PlannerCourseFilter
import com.lextures.android.core.lms.StudentTodoItem
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.features.courses.CourseDetailScreen
import com.lextures.android.features.courses.ItemDetailScreen
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsErrorBanner
import kotlinx.coroutines.launch
import java.time.ZoneId

enum class PlannerTab { Todos, Calendar }

@Composable
fun PlannerScreen(
    session: AuthSession,
    offline: OfflineService,
    isOnline: Boolean,
    initialTab: PlannerTab = PlannerTab.Todos,
    onBack: (() -> Unit)? = null,
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    val context = LocalContext.current
    var tab by remember { mutableIntStateOf(initialTab.ordinal) }
    var courses by remember { mutableStateOf<List<CourseSummary>>(emptyList()) }
    var courseFilters by remember { mutableStateOf<List<PlannerCourseFilter>>(emptyList()) }
    var todos by remember { mutableStateOf(emptyList<StudentTodoItem>()) }
    var events by remember { mutableStateOf(emptyList<PlannerCalendarEvent>()) }
    var selectedCourseCode by remember { mutableStateOf<String?>(null) }
    var showCompleted by remember { mutableStateOf(false) }
    var loading by remember { mutableStateOf(true) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var staleLabel by remember { mutableStateOf<String?>(null) }
    var showSubscribe by remember { mutableStateOf(false) }
    var openTodo by remember { mutableStateOf<Pair<StudentTodoItem, CourseSummary?>?>(null) }

    LaunchedEffect(accessToken) {
        val token = accessToken ?: return@LaunchedEffect
        loading = true
        errorMessage = null
        try {
            val result = PlannerLoader.load(token, offline, isOnline)
            courses = result.courses
            courseFilters = result.courseFilters
            todos = result.todos
            events = result.events
            staleLabel = result.staleLabel
        } catch (_: Exception) {
            if (todos.isEmpty()) {
                errorMessage = context.getString(R.string.mobile_planner_error_load)
            } else {
                staleLabel = staleLabel ?: context.getString(R.string.mobile_planner_stale_offline)
            }
        } finally {
            loading = false
        }
    }

    openTodo?.let { (item, course) ->
        if (item.kind == com.lextures.android.core.lms.StudentTodoKind.Evaluation && course != null) {
            BackHandler { openTodo = null }
            EvaluationFormScreen(session = session, course = course)
            return
        }
        val structure = plannerStructureItem(item)
        if (course != null && structure != null) {
            ItemDetailScreen(session = session, course = course, item = structure, onBack = { openTodo = null })
            return
        }
        course?.let {
            CourseDetailScreen(session = session, course = it, onBack = { openTodo = null })
            return
        }
        openTodo = null
    }

    Column(modifier = modifier.fillMaxSize()) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 8.dp, vertical = 4.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            if (onBack != null) {
                IconButton(onClick = onBack) {
                    Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                }
            }
            Text(
                text = plannerTitle(),
                modifier = Modifier.weight(1f),
            )
        }
        staleLabel?.let { label ->
            Text(label, modifier = Modifier.padding(horizontal = 16.dp, vertical = 8.dp))
        }
        errorMessage?.let { LmsErrorBanner(it) }
        TabRow(selectedTabIndex = tab) {
            Tab(
                selected = tab == 0,
                onClick = { tab = 0 },
                text = { Text(plannerTabTodos()) },
            )
            Tab(
                selected = tab == 1,
                onClick = { tab = 1 },
                text = { Text(plannerTabCalendar()) },
            )
        }
        when (tab) {
            0 -> TodosScreen(
                todos = todos,
                courseFilters = courseFilters,
                selectedCourseCode = selectedCourseCode,
                showCompleted = showCompleted,
                loading = loading,
                onCourseSelected = { selectedCourseCode = it },
                onShowCompletedChange = { showCompleted = it },
                onOpenItem = { item, course -> openTodo = item to course },
                courses = courses,
            )
            else -> CalendarScreen(
                events = events,
                courseFilters = courseFilters,
                selectedCourseCode = selectedCourseCode,
                onCourseSelected = { selectedCourseCode = it },
                onEventSelected = { event -> addEventToDeviceCalendar(context, event) },
            )
        }
        OutlinedButton(
            onClick = { showSubscribe = true },
            modifier = Modifier
                .fillMaxWidth()
                .padding(16.dp),
        ) {
            Text(plannerSubscribeTitle())
        }
    }

    if (showSubscribe) {
        CalendarSubscribeSheet(
            session = session,
            onDismiss = { showSubscribe = false },
        )
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun CalendarSubscribeSheet(
    session: AuthSession,
    onDismiss: () -> Unit,
) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
    var loading by remember { mutableStateOf(true) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var tokenInfo by remember { mutableStateOf<CalendarTokenInfo?>(null) }
    var createdToken by remember { mutableStateOf<CalendarTokenCreated?>(null) }
    var copiedMessage by remember { mutableStateOf<String?>(null) }

    LaunchedEffect(Unit) {
        val token = session.accessToken.value ?: return@LaunchedEffect
        loading = true
        runCatching {
            tokenInfo = LmsApi.fetchCalendarTokenInfo(token)
            if (tokenInfo?.hasToken != true) {
                createdToken = LmsApi.createCalendarToken(token)
                tokenInfo = LmsApi.fetchCalendarTokenInfo(token)
            }
        }.onFailure {
            errorMessage = context.getString(R.string.mobile_planner_subscribe_error)
        }
        loading = false
    }

    ModalBottomSheet(onDismissRequest = onDismiss, sheetState = sheetState) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .verticalScroll(rememberScrollState())
                .padding(20.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Text(plannerSubscribeTitle())
            Text(plannerSubscribeMessage())
            if (loading) CircularProgressIndicator()
            errorMessage?.let { LmsErrorBanner(it) }
            val feedUrl = createdToken?.feedUrl
                ?: tokenInfo?.personalFeedUrl?.replace("<token>", createdToken?.token.orEmpty())
            if (!feedUrl.isNullOrBlank() && !feedUrl.contains("<token>")) {
                LmsCard {
                    Text(feedUrl)
                    Button(onClick = {
                        val clipboard = context.getSystemService(Context.CLIPBOARD_SERVICE) as ClipboardManager
                        clipboard.setPrimaryClip(ClipData.newPlainText("calendar", feedUrl))
                        copiedMessage = context.getString(R.string.mobile_planner_subscribe_copied)
                    }) { Text(plannerSubscribeCopy()) }
                    OutlinedButton(onClick = {
                        val intent = Intent(Intent.ACTION_VIEW).apply {
                            data = android.net.Uri.parse(
                                feedUrl.replace("https://", "webcal://").replace("http://", "webcal://"),
                            )
                        }
                        context.startActivity(intent)
                    }) { Text(plannerSubscribeOpen()) }
                }
            }
            Button(onClick = {
                scope.launch {
                    val token = session.accessToken.value ?: return@launch
                    runCatching {
                        createdToken = LmsApi.createCalendarToken(token)
                        tokenInfo = LmsApi.fetchCalendarTokenInfo(token)
                    }.onFailure { errorMessage = context.getString(R.string.mobile_planner_subscribe_error) }
                }
            }) { Text(plannerSubscribeGenerate()) }
            copiedMessage?.let { Text(it) }
        }
    }
}

fun addEventToDeviceCalendar(context: Context, event: PlannerCalendarEvent) {
    val startMillis = event.startsAt.toEpochMilli()
    val endMillis = (event.endsAt ?: event.startsAt.plusSeconds(3600)).toEpochMilli()
    val intent = Intent(Intent.ACTION_INSERT).apply {
        data = CalendarContract.Events.CONTENT_URI
        putExtra(CalendarContract.Events.TITLE, event.title)
        putExtra(CalendarContract.EXTRA_EVENT_BEGIN_TIME, startMillis)
        putExtra(CalendarContract.EXTRA_EVENT_END_TIME, endMillis)
    }
    context.startActivity(intent)
}
