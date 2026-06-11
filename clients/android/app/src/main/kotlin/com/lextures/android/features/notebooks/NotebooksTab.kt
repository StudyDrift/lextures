package com.lextures.android.features.notebooks

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Description
import androidx.compose.material.icons.filled.EditNote
import androidx.compose.material.icons.filled.Public
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.LexturesType
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.LmsDates
import com.lextures.android.core.notebook.CourseNotebook
import com.lextures.android.core.notebook.NotebookStore
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsCoverTile
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsSectionHeader

private data class NotebookRow(
    val courseCode: String,
    val title: String,
    val notebook: CourseNotebook?,
)

/** My Notebooks: the global notebook plus one notebook per enrolled course (device-local). */
@Composable
fun NotebooksTab(session: AuthSession, modifier: Modifier = Modifier) {
    val context = LocalContext.current
    val accessToken by session.accessToken.collectAsState()
    val store = remember(accessToken) { NotebookStore(context, accessToken) }

    var courses by remember { mutableStateOf<List<CourseSummary>>(emptyList()) }
    var openNotebook by remember { mutableStateOf<NotebookRow?>(null) }
    // Bumped after the editor closes so previews re-read from the store.
    var revision by remember { mutableIntStateOf(0) }

    LaunchedEffect(accessToken) {
        val token = accessToken ?: return@LaunchedEffect
        courses = runCatching { LmsApi.fetchCourses(token) }.getOrDefault(emptyList())
    }

    openNotebook?.let { row ->
        NotebookEditorScreen(
            store = store,
            courseCode = row.courseCode,
            title = row.title,
            onBack = {
                openNotebook = null
                revision++
            },
            modifier = modifier,
        )
        return
    }

    val saved = remember(revision, store) { store.listCourseNotebooks() }
    val globalNotebook = remember(revision, store) { store.load(NotebookStore.GLOBAL_KEY) }

    val courseRows = remember(courses, saved) {
        val rows = mutableListOf<NotebookRow>()
        val seen = mutableSetOf<String>()
        for (course in courses) {
            if (course.notebookEnabled == false) continue
            rows.add(NotebookRow(course.courseCode, course.displayTitle, saved[course.courseCode]))
            seen.add(course.courseCode)
        }
        for ((code, notebook) in saved) {
            if (code !in seen) rows.add(NotebookRow(code, notebook.courseTitle ?: code, notebook))
        }
        rows
    }

    Column(modifier = modifier) {
        Text(
            text = "Notebooks",
            style = LexturesType.display(24),
            color = textPrimary(),
            modifier = Modifier.padding(start = 16.dp, top = 12.dp),
        )

        LazyColumn(
            modifier = Modifier.fillMaxSize(),
            contentPadding = PaddingValues(16.dp),
            verticalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            item { LmsSectionHeader("Global notebook", Icons.Default.Public) }
            item {
                NotebookCard(
                    title = NotebookStore.GLOBAL_TITLE,
                    subtitle = "Notes that follow you across courses",
                    notebook = globalNotebook,
                    onClick = {
                        openNotebook = NotebookRow(NotebookStore.GLOBAL_KEY, NotebookStore.GLOBAL_TITLE, globalNotebook)
                    },
                )
            }

            item { LmsSectionHeader("Course notebooks", Icons.Default.Description) }
            if (courseRows.isEmpty()) {
                item {
                    LmsEmptyState(
                        icon = Icons.Default.Description,
                        title = "No course notebooks",
                        message = "Enroll in a course to start a notebook for it.",
                    )
                }
            } else {
                items(courseRows, key = { it.courseCode }) { row ->
                    NotebookCard(
                        title = row.title,
                        subtitle = row.courseCode,
                        notebook = row.notebook,
                        onClick = { openNotebook = row },
                    )
                }
            }
        }
    }
}

@Composable
private fun NotebookCard(
    title: String,
    subtitle: String,
    notebook: CourseNotebook?,
    onClick: () -> Unit,
) {
    LmsCard(onClick = onClick) {
        Row(horizontalArrangement = Arrangement.spacedBy(14.dp)) {
            LmsCoverTile(key = subtitle, icon = Icons.Default.EditNote, size = 48)
            Column(verticalArrangement = Arrangement.spacedBy(2.dp)) {
                Text(
                    text = title,
                    fontSize = 15.sp,
                    fontWeight = FontWeight.SemiBold,
                    color = textPrimary(),
                )
                Text(text = subtitle, fontSize = 12.sp, color = textSecondary())
                val preview = notebook?.previewText.orEmpty()
                if (preview.isNotEmpty()) {
                    Text(
                        text = preview,
                        fontSize = 12.sp,
                        color = textSecondary(),
                        maxLines = 2,
                        overflow = TextOverflow.Ellipsis,
                    )
                    Text(
                        text = "Updated ${LmsDates.shortDateTime(notebook?.updatedAt)}",
                        fontSize = 11.sp,
                        color = textSecondary(),
                    )
                } else {
                    Text(
                        text = "No notes yet",
                        fontSize = 12.sp,
                        fontStyle = FontStyle.Italic,
                        color = textSecondary(),
                    )
                }
            }
        }
    }
}
