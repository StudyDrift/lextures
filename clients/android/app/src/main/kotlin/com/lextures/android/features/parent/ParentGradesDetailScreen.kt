package com.lextures.android.features.parent

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
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
import com.lextures.android.core.i18n.LocalLocalePreferences
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.ParentCourseGradesRow
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsSkeletonList

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ParentGradesDetailScreen(
    session: AuthSession,
    studentId: String,
    childName: String,
    onBack: () -> Unit,
) {
    val accessToken by session.accessToken.collectAsState()
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    var courses by remember { mutableStateOf<List<ParentCourseGradesRow>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var errorMessage by remember { mutableStateOf<String?>(null) }

    LaunchedEffect(accessToken, studentId) {
        val token = accessToken ?: return@LaunchedEffect
        loading = true
        try {
            courses = LmsApi.fetchParentStudentGrades(studentId, token)
            errorMessage = null
        } catch (e: Exception) {
            errorMessage = e.message
        } finally {
            loading = false
        }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text(L.text(context, localePrefs, R.string.mobile_parent_section_grades)) },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = null)
                    }
                },
            )
        },
    ) { padding ->
        when {
            loading -> LmsSkeletonList(count = 3, modifier = Modifier.padding(padding).fillMaxSize())
            errorMessage != null && courses.isEmpty() -> LmsEmptyState(
                icon = Icons.AutoMirrored.Filled.ArrowBack,
                title = L.text(context, localePrefs, R.string.mobile_parent_section_grades),
                message = errorMessage.orEmpty(),
                modifier = Modifier.padding(padding).fillMaxSize(),
            )
            else -> Column(
                modifier = Modifier
                    .padding(padding)
                    .fillMaxSize()
                    .verticalScroll(rememberScrollState())
                    .padding(16.dp),
                verticalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                Text(
                    localePrefs.localizedContext(context).getString(R.string.mobile_parent_readOnly, childName),
                    fontSize = 12.sp,
                    color = textSecondary(),
                )
                if (courses.isEmpty()) {
                    Text(L.text(context, localePrefs, R.string.mobile_parent_grades_empty), color = textSecondary())
                } else {
                    courses.forEach { course ->
                        LmsCard {
                            Text(course.title, fontWeight = FontWeight.Bold)
                            Text(course.courseCode, fontSize = 12.sp, color = textSecondary())
                            if (course.grades.isEmpty()) {
                                Text(
                                    L.text(context, localePrefs, R.string.mobile_parent_grades_noScores),
                                    modifier = Modifier.padding(top = 8.dp),
                                )
                            } else {
                                course.grades.toSortedMap().forEach { (itemId, score) ->
                                    Row(Modifier.fillMaxWidth().padding(vertical = 4.dp)) {
                                        Text(itemId.take(8) + "…", fontSize = 12.sp, color = textSecondary())
                                        Text(score, modifier = Modifier.weight(1f), fontWeight = FontWeight.Medium)
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}
