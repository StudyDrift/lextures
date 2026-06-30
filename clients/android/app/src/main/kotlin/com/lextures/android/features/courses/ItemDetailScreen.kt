package com.lextures.android.features.courses

import android.content.Intent
import androidx.activity.compose.BackHandler
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.automirrored.filled.Assignment
import androidx.compose.material.icons.automirrored.filled.MenuBook
import androidx.compose.material.icons.filled.AutoAwesome
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.Description
import androidx.compose.material.icons.filled.Inbox
import androidx.compose.material.icons.filled.Layers
import androidx.compose.material.icons.filled.Link
import androidx.compose.material.icons.filled.Schedule
import androidx.compose.material.icons.filled.Star
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
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
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.platform.LocalContext
import androidx.core.net.toUri
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.accessibility.ReadAloudControls
import com.lextures.android.core.design.LexturesType
import com.lextures.android.core.design.accentColor
import com.lextures.android.core.design.AuthPrimaryButton
import com.lextures.android.core.design.isDarkTheme
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.lms.AssignmentSubmission
import com.lextures.android.core.lms.CourseStructureItem
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.LmsDates
import com.lextures.android.core.lms.ModuleContentLogic
import com.lextures.android.core.lms.ModuleItemDetail
import com.lextures.android.core.lms.SubmissionGrade
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsCoverTile
import com.lextures.android.features.home.LmsErrorBanner

/** Shared icon/label mapping for course structure item kinds. */
object ItemKind {
    fun icon(kind: String): ImageVector = when (kind) {
        "assignment" -> Icons.AutoMirrored.Filled.Assignment
        "quiz" -> Icons.Default.CheckCircle
        "content_page" -> Icons.Default.Description
        "external_link", "lti_link" -> Icons.Default.Link
        "h5p", "vibe_activity" -> Icons.Default.AutoAwesome
        "library_resource", "textbook_resource" -> Icons.AutoMirrored.Filled.MenuBook
        else -> Icons.Default.Layers
    }

    fun label(kind: String): String = when (kind) {
        "assignment" -> "Assignment"
        "quiz" -> "Quiz"
        "content_page" -> "Page"
        "external_link" -> "External link"
        "lti_link" -> "External tool"
        "h5p" -> "Interactive"
        "vibe_activity" -> "Activity"
        "library_resource" -> "Library"
        "textbook_resource" -> "Textbook"
        else -> "Item"
    }

    /** Kinds the module list can navigate to (including placeholders for upcoming epics). */
    fun isOpenable(kind: String): Boolean = ModuleContentLogic.isNavigable(kind)
}

/** Activity detail: content body plus the settings "preview box" (parity with web). */
@Composable
fun ItemDetailScreen(
    session: AuthSession,
    course: CourseSummary,
    item: CourseStructureItem,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    val context = LocalContext.current
    val courseCode = course.courseCode

    var detail by remember { mutableStateOf<ModuleItemDetail?>(null) }
    var mySubmission by remember { mutableStateOf<AssignmentSubmission?>(null) }
    var myGrade by remember { mutableStateOf<SubmissionGrade?>(null) }
    var submissionLoaded by remember { mutableStateOf(false) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }

    BackHandler(onBack = onBack)

    LaunchedEffect(accessToken, item.id) {
        val token = accessToken ?: return@LaunchedEffect
        loading = true
        errorMessage = null
        try {
            detail = LmsApi.fetchItemDetail(courseCode, item, token)
        } catch (e: Exception) {
            errorMessage = session.mapError(e)
        } finally {
            loading = false
        }
        // Student view of their own submission + released grade (assignments only).
        if (item.kind == "assignment" && course.viewerIsStudent) {
            submissionLoaded = false
            mySubmission = runCatching { LmsApi.fetchMySubmission(courseCode, item.id, token) }.getOrNull()
            myGrade = mySubmission?.let { submission ->
                runCatching { LmsApi.fetchSubmissionGrade(courseCode, item.id, submission.id, token) }.getOrNull()
            }
            submissionLoaded = true
        }
    }

    Column(modifier = modifier) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(top = 8.dp, end = 16.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            IconButton(onClick = onBack) {
                Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back", tint = textPrimary())
            }
            Text(
                text = item.title,
                fontSize = 17.sp,
                fontWeight = FontWeight.SemiBold,
                color = textPrimary(),
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
        }

        if (loading) {
            Box(modifier = Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                CircularProgressIndicator(color = LexturesColors.Primary)
            }
            return
        }

        LazyColumn(
            modifier = Modifier.fillMaxSize(),
            contentPadding = PaddingValues(16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            item {
                Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        DetailChip(ItemKind.label(item.kind), ItemKind.icon(item.kind), accentColor())
                        val due = detail?.dueAt ?: item.dueAt
                        if (LmsDates.parse(due) != null) {
                            DetailChip("Due ${LmsDates.shortDateTime(due)}", Icons.Default.Schedule, LexturesColors.Coral)
                        }
                    }
                    Text(
                        text = detail?.title ?: item.title,
                        style = LexturesType.display(24),
                        color = textPrimary(),
                    )
                }
            }

            errorMessage?.let { message ->
                item { LmsErrorBanner(message) }
            }

            val url = detail?.url
            if (!url.isNullOrEmpty()) {
                item {
                    LmsCard {
                        Row(
                            horizontalArrangement = Arrangement.spacedBy(12.dp),
                            verticalAlignment = Alignment.CenterVertically,
                        ) {
                            LmsCoverTile(key = url, icon = Icons.Default.Link, size = 40)
                            Column {
                                detail?.provider?.takeIf { it.isNotEmpty() }?.let {
                                    Text(
                                        text = it,
                                        fontSize = 14.sp,
                                        fontWeight = FontWeight.SemiBold,
                                        color = textPrimary(),
                                    )
                                }
                                Text(
                                    text = url,
                                    fontSize = 12.sp,
                                    color = textSecondary(),
                                    maxLines = 1,
                                    overflow = TextOverflow.Ellipsis,
                                )
                            }
                        }
                        AuthPrimaryButton(
                            text = "Open link",
                            onClick = {
                                runCatching {
                                    context.startActivity(Intent(Intent.ACTION_VIEW, url.toUri()))
                                }
                            },
                        )
                    }
                }
            }

            val markdown = detail?.markdown
            if (!markdown.isNullOrBlank()) {
                item {
                    LmsCard {
                        ReadAloudControls(text = markdown)
                        MarkdownText(markdown)
                    }
                }
            }

            if (item.kind == "assignment" && course.viewerIsStudent && submissionLoaded) {
                item {
                    MySubmissionCard(submission = mySubmission, grade = myGrade)
                }
            }

            val rows = detailRows(item, detail)
            if (rows.isNotEmpty() || item.kind == "quiz") {
                item {
                    LmsCard {
                        if (item.kind == "quiz") {
                            Text(
                                text = "${detail?.questionCount ?: 0} questions",
                                style = LexturesType.display(18),
                                color = textPrimary(),
                            )
                            Text(
                                text = "A quick look at how this quiz is set up.",
                                fontSize = 12.sp,
                                color = textSecondary(),
                            )
                        } else {
                            Text(text = "Details", style = LexturesType.display(18), color = textPrimary())
                        }

                        HorizontalDivider(modifier = Modifier.padding(vertical = 4.dp))

                        rows.forEach { (label, value) ->
                            Row(
                                modifier = Modifier
                                    .fillMaxWidth()
                                    .padding(vertical = 3.dp),
                            ) {
                                Text(
                                    text = label,
                                    fontSize = 14.sp,
                                    color = textSecondary(),
                                    modifier = Modifier.weight(1f),
                                )
                                Text(
                                    text = value,
                                    fontSize = 14.sp,
                                    fontWeight = FontWeight.SemiBold,
                                    color = textPrimary(),
                                    textAlign = TextAlign.End,
                                )
                            }
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun DetailChip(text: String, icon: ImageVector, tint: Color) {
    Row(
        modifier = Modifier
            .clip(RoundedCornerShape(50))
            .background(tint.copy(alpha = 0.13f))
            .padding(horizontal = 9.dp, vertical = 4.dp),
        horizontalArrangement = Arrangement.spacedBy(4.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(icon, contentDescription = null, tint = tint, modifier = Modifier.size(13.dp))
        Text(text = text, fontSize = 12.sp, fontWeight = FontWeight.SemiBold, color = tint)
    }
}

private fun detailRows(item: CourseStructureItem, detail: ModuleItemDetail?): List<Pair<String, String>> {
    val rows = mutableListOf<Pair<String, String>>()
    val due = detail?.dueAt ?: item.dueAt
    if (LmsDates.parse(due) != null) {
        rows.add("Due date" to LmsDates.shortDateTime(due))
    }
    val points = detail?.pointsWorth ?: item.pointsWorth?.toInt() ?: item.pointsPossible?.toInt()

    when (item.kind) {
        "quiz" -> detail?.let { d ->
            rows.add("Unlimited attempts" to yesNo(d.unlimitedAttempts ?: false))
            rows.add("One question at a time" to yesNo(d.oneQuestionAtATime ?: false))
            rows.add("Course lockdown feature" to lockdownLabel(d.lockdownMode))
            rows.add("Delivery mode" to titlecase(d.adaptiveDeliveryMode ?: "standard"))
            if (d.unlimitedAttempts != true) {
                rows.add("Max attempts" to "${d.maxAttempts ?: 1}")
            }
            rows.add("Grade uses" to gradePolicyLabel(d.gradeAttemptPolicy))
            d.timeLimitMinutes?.let { rows.add("Time limit" to "$it min") }
            d.passingScorePercent?.let { rows.add("Passing score" to "$it%") }
            points?.let { rows.add("Points" to "$it") }
            rows.add("Shuffle questions" to yesNo(d.shuffleQuestions ?: false))
        }
        "assignment" -> {
            points?.let { rows.add("Points" to "$it") }
            detail?.let { d ->
                val types = listOfNotNull(
                    "Text".takeIf { d.submissionAllowText == true },
                    "File upload".takeIf { d.submissionAllowFileUpload == true },
                    "URL".takeIf { d.submissionAllowUrl == true },
                )
                if (types.isNotEmpty()) rows.add("Submission types" to types.joinToString(", "))
                d.lateSubmissionPolicy?.let { rows.add("Late submissions" to lateLabel(it, d.latePenaltyPercent)) }
                if (LmsDates.parse(d.availableFrom) != null) {
                    rows.add("Available from" to LmsDates.shortDateTime(d.availableFrom))
                }
                if (LmsDates.parse(d.availableUntil) != null) {
                    rows.add("Available until" to LmsDates.shortDateTime(d.availableUntil))
                }
            }
        }
        "external_link" -> detail?.provider?.takeIf { it.isNotEmpty() }?.let {
            rows.add("Provider" to titlecase(it))
        }
        else -> points?.let { rows.add("Points" to "$it") }
    }

    if (LmsDates.parse(detail?.updatedAt) != null) {
        rows.add("Updated" to LmsDates.shortDate(detail?.updatedAt))
    }
    return rows
}

private fun yesNo(value: Boolean) = if (value) "Yes" else "No"

private fun lockdownLabel(mode: String?): String =
    if (mode.isNullOrEmpty() || mode == "off" || mode == "none") "Off" else titlecase(mode)

private fun gradePolicyLabel(policy: String?): String = when (policy) {
    "highest" -> "Highest attempt"
    "latest" -> "Latest attempt"
    "first" -> "First attempt"
    "average" -> "Average of attempts"
    else -> titlecase(policy ?: "Latest attempt")
}

private fun lateLabel(policy: String, penalty: Int?): String = when (policy) {
    "allow" -> "Allowed"
    "block", "reject" -> "Not allowed"
    "penalty" -> if (penalty != null) "Allowed, −$penalty%" else "Allowed with penalty"
    else -> titlecase(policy)
}

private fun titlecase(raw: String): String =
    raw.replace('_', ' ').replaceFirstChar { it.uppercase() }

/** Lightweight block-level markdown renderer (headings, bullets, paragraphs). */
@Composable
fun MarkdownText(markdown: String, modifier: Modifier = Modifier) {
    val lines = remember(markdown) { parseMarkdownBlocks(markdown) }
    Column(modifier = modifier, verticalArrangement = Arrangement.spacedBy(8.dp)) {
        lines.forEach { block ->
            when (block) {
                is MdBlock.Heading -> Text(
                    text = block.text,
                    style = LexturesType.display(if (block.level == 1) 21 else if (block.level == 2) 18 else 16),
                    color = textPrimary(),
                    modifier = Modifier.padding(top = 4.dp),
                )
                is MdBlock.Bullet -> Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    Box(
                        modifier = Modifier
                            .padding(top = 7.dp)
                            .size(5.dp)
                            .clip(CircleShape)
                            .background(accentColor()),
                    )
                    Text(text = block.text, fontSize = 14.sp, color = textPrimary())
                }
                is MdBlock.Numbered -> Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    Text(
                        text = block.index,
                        fontSize = 14.sp,
                        fontWeight = FontWeight.SemiBold,
                        color = accentColor(),
                    )
                    Text(text = block.text, fontSize = 14.sp, color = textPrimary())
                }
                is MdBlock.Paragraph -> Text(
                    text = block.text,
                    fontSize = 14.sp,
                    lineHeight = 21.sp,
                    color = textPrimary(),
                )
            }
        }
    }
}

private sealed interface MdBlock {
    data class Heading(val level: Int, val text: String) : MdBlock
    data class Bullet(val text: String) : MdBlock
    data class Numbered(val index: String, val text: String) : MdBlock
    data class Paragraph(val text: String) : MdBlock
}

private val numberedRegex = Regex("""^(\d+)\.\s+(.*)$""")

private fun stripInline(text: String): String =
    text.replace(Regex("""\*\*(.+?)\*\*"""), "$1")
        .replace(Regex("""\*(.+?)\*"""), "$1")
        .replace(Regex("""`(.+?)`"""), "$1")
        .replace(Regex("""\[(.+?)]\(.+?\)"""), "$1")

private fun parseMarkdownBlocks(markdown: String): List<MdBlock> {
    val out = mutableListOf<MdBlock>()
    val paragraph = mutableListOf<String>()

    fun flush() {
        if (paragraph.isNotEmpty()) {
            out.add(MdBlock.Paragraph(stripInline(paragraph.joinToString(" "))))
            paragraph.clear()
        }
    }

    markdown.lineSequence().forEach { raw ->
        val line = raw.trim()
        when {
            line.isEmpty() -> flush()
            line.startsWith("#") -> {
                flush()
                val level = line.takeWhile { it == '#' }.length.coerceAtMost(3)
                out.add(MdBlock.Heading(level, stripInline(line.dropWhile { it == '#' }.trim())))
            }
            line.startsWith("- ") || line.startsWith("* ") -> {
                flush()
                out.add(MdBlock.Bullet(stripInline(line.drop(2))))
            }
            numberedRegex.matches(line) -> {
                flush()
                val match = numberedRegex.find(line)!!
                out.add(MdBlock.Numbered("${match.groupValues[1]}.", stripInline(match.groupValues[2])))
            }
            else -> paragraph.add(line)
        }
    }
    flush()
    return out
}

/** Student view of their own submission status and released grade. */
@Composable
private fun MySubmissionCard(
    submission: AssignmentSubmission?,
    grade: SubmissionGrade?,
) {
    if (submission == null) {
        LmsCard {
            Row(
                horizontalArrangement = Arrangement.spacedBy(10.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Icon(
                    Icons.Default.Inbox,
                    contentDescription = null,
                    tint = textSecondary(),
                    modifier = Modifier.size(18.dp),
                )
                Column(verticalArrangement = Arrangement.spacedBy(2.dp)) {
                    Text(
                        text = "Not submitted yet",
                        fontSize = 14.sp,
                        fontWeight = FontWeight.Medium,
                        color = textPrimary(),
                    )
                    Text(
                        text = "Submit this assignment from the web app.",
                        fontSize = 12.sp,
                        color = textSecondary(),
                    )
                }
            }
        }
        return
    }

    val revisionRequested = submission.resubmissionRequested == true
    LmsCard(accent = if (revisionRequested) LexturesColors.Coral else LexturesColors.BrandTeal) {
        Text(text = "Your submission", style = LexturesType.display(18), color = textPrimary())

        Row(
            horizontalArrangement = Arrangement.spacedBy(10.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Icon(
                Icons.Default.CheckCircle,
                contentDescription = null,
                tint = LexturesColors.Primary,
                modifier = Modifier.size(18.dp),
            )
            Column(verticalArrangement = Arrangement.spacedBy(2.dp)) {
                Text(
                    text = "Submitted ${LmsDates.shortDateTime(submission.submittedAt)}",
                    fontSize = 14.sp,
                    fontWeight = FontWeight.Medium,
                    color = textPrimary(),
                )
                submission.versionNumber?.takeIf { it > 1 }?.let {
                    Text(text = "Version $it", fontSize = 12.sp, color = textSecondary())
                }
            }
        }

        submission.attachmentFilename?.takeIf { it.isNotEmpty() }?.let { filename ->
            Text(text = filename, fontSize = 12.sp, color = textSecondary())
        }

        if (revisionRequested) {
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .clip(RoundedCornerShape(12.dp))
                    .background(LexturesColors.Coral.copy(alpha = 0.08f))
                    .padding(10.dp),
                verticalArrangement = Arrangement.spacedBy(4.dp),
            ) {
                Text(
                    text = "Revision requested",
                    fontSize = 12.sp,
                    fontWeight = FontWeight.SemiBold,
                    color = LexturesColors.Coral,
                )
                submission.revisionFeedback?.takeIf { it.isNotEmpty() }?.let {
                    Text(text = it, fontSize = 12.sp, color = textSecondary())
                }
                LmsDates.parse(submission.revisionDueAt)?.let {
                    Text(
                        text = "Revise by ${LmsDates.shortDateTime(submission.revisionDueAt)}",
                        fontSize = 12.sp,
                        fontWeight = FontWeight.SemiBold,
                        color = LexturesColors.Coral,
                    )
                }
            }
        }

        if (grade?.posted == true && grade.pointsEarned != null) {
            HorizontalDivider()
            Row(
                modifier = Modifier.fillMaxWidth(),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text(
                    text = "Grade",
                    fontSize = 14.sp,
                    color = textSecondary(),
                    modifier = Modifier.weight(1f),
                )
                val earned = grade.pointsEarned
                val max = grade.maxPoints
                Text(
                    text = if (max != null) "${fmtPts(earned)} / ${fmtPts(max)}" else fmtPts(earned),
                    style = LexturesType.display(18, FontWeight.Bold),
                    color = LexturesColors.Primary,
                )
            }
            grade.instructorComment?.takeIf { it.isNotEmpty() }?.let {
                Text(
                    text = "“$it”",
                    fontSize = 13.sp,
                    fontStyle = androidx.compose.ui.text.font.FontStyle.Italic,
                    color = textSecondary(),
                )
            }
        }
    }
}

private fun fmtPts(points: Double): String =
    if (points % 1.0 == 0.0) points.toLong().toString() else points.toString()
