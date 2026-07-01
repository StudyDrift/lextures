package com.lextures.android.features.peerreview

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
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateMapOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.OfflineBanner
import com.lextures.android.core.design.StalenessChip
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.lms.AssignmentLogic
import com.lextures.android.core.lms.AssignmentSubmission
import com.lextures.android.core.lms.FilePreviewTarget
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.LmsDates
import com.lextures.android.core.lms.PeerReviewAllocation
import com.lextures.android.core.lms.PeerReviewAllocationDetail
import com.lextures.android.core.lms.PeerReviewLogic
import com.lextures.android.core.lms.PeerReviewReceivedItem
import com.lextures.android.core.lms.PeerReviewSubmitRequest
import com.lextures.android.core.lms.RubricCriterion
import com.lextures.android.core.lms.RubricDefinition
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.features.files.FilePreviewScreen
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsSkeletonList
import kotlinx.coroutines.launch
import kotlinx.serialization.builtins.ListSerializer
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json

private val offlineJson = Json { ignoreUnknownKeys = true }

@Composable
fun PeerReviewListScreen(
    session: AuthSession,
    onOpenAllocation: (String) -> Unit,
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    val context = LocalContext.current
    val offline = remember { OfflineService.get(context) }
    val isOnline by offline.networkMonitor.isOnline.collectAsState()

    var allocations by remember { mutableStateOf<List<PeerReviewAllocation>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var cacheLabel by remember { mutableStateOf<String?>(null) }

    LaunchedEffect(accessToken) {
        val token = accessToken ?: return@LaunchedEffect
        loading = true
        errorMessage = null
        try {
            val result = offline.cachedFetch(
                key = PeerReviewLogic.cacheKeyAssigned(),
                accessToken = token,
                serializer = ListSerializer(PeerReviewAllocation.serializer()),
            ) { LmsApi.fetchPeerReviewAssigned(token) }
            allocations = result.first
            cacheLabel = result.second?.takeIf { it.isStale(isOnline) }?.lastUpdatedLabel()
        } catch (e: Exception) {
            errorMessage = session.mapError(e)
        } finally {
            loading = false
        }
    }

    Column(
        modifier = modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        if (!isOnline) OfflineBanner()
        cacheLabel?.let { StalenessChip(label = it) }
        errorMessage?.let { LmsErrorBanner(it) }

        LmsCard {
            Text(L.text(R.string.mobile_peerReview_progress), fontWeight = FontWeight.SemiBold, color = textPrimary())
            Text(
                L.format(
                    R.string.mobile_peerReview_progressSummary,
                    PeerReviewLogic.completedCount(allocations),
                    allocations.size,
                ),
                color = textSecondary(),
            )
        }

        when {
            loading && allocations.isEmpty() -> LmsSkeletonList(count = 3)
            PeerReviewLogic.pending(allocations).isEmpty() -> LmsEmptyState(
                icon = Icons.Default.CheckCircle,
                title = L.text(R.string.mobile_peerReview_allDoneTitle),
                message = L.text(R.string.mobile_peerReview_allDoneMessage),
            )
            else -> {
                for (allocation in PeerReviewLogic.pending(allocations)) {
                    val label = PeerReviewLogic.targetLabel(allocation)
                        ?: L.text(R.string.mobile_peerReview_anonymousPeer)
                    LmsCard(
                        modifier = Modifier
                            .fillMaxWidth()
                            .clickable { onOpenAllocation(allocation.id) },
                    ) {
                        Text(
                            L.format(R.string.mobile_peerReview_reviewTarget, label),
                            fontWeight = FontWeight.SemiBold,
                            color = textPrimary(),
                        )
                        Text(allocation.courseCode, color = textSecondary())
                        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                            Text(L.text(PeerReviewLogic.statusLabelRes(allocation.status)), color = textSecondary())
                            allocation.closesAt?.let { raw ->
                                Text(
                                    L.format(R.string.mobile_peerReview_due, LmsDates.shortDateTime(raw)),
                                    color = textSecondary(),
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
fun PeerReviewDetailScreen(
    session: AuthSession,
    allocationId: String,
    onSubmitted: () -> Unit = {},
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    val context = LocalContext.current
    val offline = remember { OfflineService.get(context) }
    val isOnline by offline.networkMonitor.isOnline.collectAsState()
    val scope = rememberCoroutineScope()

    var detail by remember { mutableStateOf<PeerReviewAllocationDetail?>(null) }
    val rubricScores = remember { mutableStateMapOf<String, Double>() }
    var scoreText by remember { mutableStateOf("") }
    var comments by remember { mutableStateOf("") }
    var loading by remember { mutableStateOf(true) }
    var saving by remember { mutableStateOf(false) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var successMessage by remember { mutableStateOf<String?>(null) }
    var previewTarget by remember { mutableStateOf<FilePreviewTarget?>(null) }
    var reloadKey by remember { mutableStateOf(0) }
    val queueLabel = L.text(R.string.mobile_peerReview_queueLabel)
    val submitSuccessLabel = L.text(R.string.mobile_peerReview_submitSuccess)

    LaunchedEffect(allocationId, accessToken, reloadKey) {
        val token = accessToken ?: return@LaunchedEffect
        loading = true
        errorMessage = null
        try {
            val result = offline.cachedFetch(
                key = PeerReviewLogic.cacheKeyAllocation(allocationId),
                accessToken = token,
                serializer = PeerReviewAllocationDetail.serializer(),
            ) { LmsApi.fetchPeerReviewAllocation(allocationId, token) }
            detail = result.first
            result.first.review?.let { review ->
                review.score?.let { scoreText = it.toString() }
                rubricScores.clear()
                review.rubricScores?.forEach { (k, v) -> rubricScores[k] = v }
                comments = review.comments.orEmpty()
            }
        } catch (e: Exception) {
            errorMessage = session.mapError(e)
        } finally {
            loading = false
        }
    }

    previewTarget?.let { target ->
        FilePreviewScreen(
            session = session,
            target = target,
            onBack = { previewTarget = null },
        )
        return
    }

    Column(
        modifier = modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        if (loading && detail == null) {
            CircularProgressIndicator(Modifier.align(Alignment.CenterHorizontally))
            return@Column
        }
        errorMessage?.let { LmsErrorBanner(it) }
        successMessage?.let {
            LmsCard { Text(it, fontWeight = FontWeight.SemiBold, color = textPrimary()) }
        }

        detail?.let { value ->
            val targetLabel = PeerReviewLogic.targetLabel(value.allocation)
                ?: L.text(R.string.mobile_peerReview_anonymousPeer)
            LmsCard {
                Text(targetLabel, fontWeight = FontWeight.Bold, color = textPrimary())
                Text(value.allocation.courseCode, color = textSecondary())
            }
            SubmissionCard(
                submission = value.submission,
                courseCode = value.allocation.courseCode,
                onPreview = { previewTarget = it },
            )
            value.rubric?.takeIf { it.criteria.isNotEmpty() }?.let { rubric ->
                RubricScorerSection(rubric = rubric, scores = rubricScores, disabled = saving)
            } ?: run {
                LmsCard {
                    Text(L.text(R.string.mobile_peerReview_score), fontWeight = FontWeight.SemiBold)
                    OutlinedTextField(
                        value = scoreText,
                        onValueChange = { scoreText = it },
                        modifier = Modifier.fillMaxWidth(),
                        label = { Text(L.text(R.string.mobile_peerReview_scorePlaceholder)) },
                    )
                }
            }
            LmsCard {
                Text(L.text(R.string.mobile_peerReview_comments), fontWeight = FontWeight.SemiBold)
                OutlinedTextField(
                    value = comments,
                    onValueChange = { comments = it },
                    modifier = Modifier.fillMaxWidth(),
                    minLines = 4,
                    label = { Text(L.text(R.string.mobile_peerReview_commentsPlaceholder)) },
                )
            }
            Button(
                onClick = {
                    val token = accessToken ?: return@Button
                    scope.launch {
                        saving = true
                        errorMessage = null
                        try {
                            val trimmed = comments.trim()
                            val body = if (value.rubric?.criteria?.isNotEmpty() == true) {
                                PeerReviewSubmitRequest(
                                    score = PeerReviewLogic.rubricTotal(value.rubric!!, rubricScores),
                                    rubricScores = rubricScores.toMap(),
                                    comments = trimmed.takeIf { it.isNotEmpty() },
                                )
                            } else {
                                PeerReviewSubmitRequest(
                                    score = scoreText.toDoubleOrNull(),
                                    comments = trimmed.takeIf { it.isNotEmpty() },
                                )
                            }
                            if (isOnline) {
                                LmsApi.submitPeerReview(allocationId, body, token)
                            } else {
                                offline.enqueueMutation(
                                    method = "POST",
                                    path = "/api/v1/peer-review/allocations/$allocationId",
                                    bodyJson = offlineJson.encodeToString(body),
                                    label = queueLabel,
                                    accessToken = token,
                                    preferQueue = true,
                                )
                            }
                            successMessage = submitSuccessLabel
                            onSubmitted()
                            reloadKey++
                        } catch (e: Exception) {
                            errorMessage = session.mapError(e)
                        } finally {
                            saving = false
                        }
                    }
                },
                enabled = !saving,
                modifier = Modifier.fillMaxWidth(),
            ) {
                if (saving) {
                    CircularProgressIndicator()
                } else {
                    Text(
                        if (PeerReviewLogic.isComplete(value.allocation)) {
                            L.text(R.string.mobile_peerReview_updateReview)
                        } else {
                            L.text(R.string.mobile_peerReview_submitReview)
                        },
                    )
                }
            }
        }
    }
}

@Composable
fun ReviewsReceivedScreen(
    session: AuthSession,
    courseCode: String,
    assignmentId: String,
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    val context = LocalContext.current
    val offline = remember { OfflineService.get(context) }
    val isOnline by offline.networkMonitor.isOnline.collectAsState()

    var reviews by remember { mutableStateOf<List<PeerReviewReceivedItem>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var errorMessage by remember { mutableStateOf<String?>(null) }

    LaunchedEffect(courseCode, assignmentId, accessToken) {
        val token = accessToken ?: return@LaunchedEffect
        loading = true
        try {
            val result = offline.cachedFetch(
                key = PeerReviewLogic.cacheKeyReceived(courseCode, assignmentId),
                accessToken = token,
                serializer = ListSerializer(PeerReviewReceivedItem.serializer()),
            ) { LmsApi.fetchPeerReviewReceived(courseCode, assignmentId, token) }
            reviews = result.first
        } catch (e: Exception) {
            errorMessage = session.mapError(e)
        } finally {
            loading = false
        }
    }

    Column(
        modifier = modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        if (!isOnline) OfflineBanner()
        errorMessage?.let { LmsErrorBanner(it) }
        when {
            loading && reviews.isEmpty() -> LmsSkeletonList(count = 2)
            reviews.isEmpty() -> LmsEmptyState(
                icon = Icons.Default.CheckCircle,
                title = L.text(R.string.mobile_peerReview_receivedEmptyTitle),
                message = L.text(R.string.mobile_peerReview_receivedEmptyMessage),
            )
            else -> reviews.forEach { review ->
                LmsCard {
                    Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                        Text(
                            review.reviewerLabel ?: L.text(R.string.mobile_peerReview_anonymousReviewer),
                            fontWeight = FontWeight.SemiBold,
                        )
                        review.score?.let {
                            Text(L.format(R.string.mobile_peerReview_scoreValue, it.toString()))
                        }
                    }
                    review.comments?.takeIf { it.isNotBlank() }?.let { Text(it) }
                }
            }
        }
    }
}

@Composable
private fun SubmissionCard(
    submission: AssignmentSubmission,
    courseCode: String,
    onPreview: (FilePreviewTarget) -> Unit,
) {
    LmsCard {
        Text(L.text(R.string.mobile_peerReview_theirWork), fontWeight = FontWeight.SemiBold, color = textPrimary())
        submission.bodyText?.takeIf { it.isNotBlank() }?.let { Text(it, color = textPrimary()) }
        if (AssignmentLogic.hasAttachment(submission)) {
            val path = submission.attachmentContentPath
            val name = submission.attachmentFilename
            if (path != null && name != null) {
                Text(
                    name,
                    modifier = Modifier
                        .padding(top = 8.dp)
                        .clickable {
                            onPreview(
                                FilePreviewTarget.submissionContentPath(
                                    courseCode = courseCode,
                                    contentPath = path,
                                    fileName = name,
                                    mimeType = submission.attachmentMimeType,
                                ),
                            )
                        },
                    color = textSecondary(),
                )
            }
        }
    }
}

@Composable
private fun RubricScorerSection(
    rubric: RubricDefinition,
    scores: MutableMap<String, Double>,
    disabled: Boolean,
) {
    LmsCard {
        Text(L.text(R.string.mobile_peerReview_rubricScore), fontWeight = FontWeight.SemiBold)
        Text(PeerReviewLogic.rubricTotal(rubric, scores).toString(), fontWeight = FontWeight.Bold)
        Text(
            L.format(
                R.string.mobile_peerReview_criteriaProgress,
                PeerReviewLogic.rubricGradedCount(rubric, scores),
                rubric.criteria.size,
            ),
        )
    }
    rubric.criteria.forEachIndexed { index, criterion ->
        RubricCriterionPicker(
            index = index,
            criterion = criterion,
            selected = scores[criterion.id],
            disabled = disabled,
            onSelect = { scores[criterion.id] = it },
        )
    }
}

@Composable
private fun RubricCriterionPicker(
    index: Int,
    criterion: RubricCriterion,
    selected: Double?,
    disabled: Boolean,
    onSelect: (Double) -> Unit,
) {
    LmsCard {
        Text("${index + 1}. ${criterion.title}", fontWeight = FontWeight.SemiBold)
        criterion.description?.takeIf { it.isNotBlank() }?.let { Text(it, color = textSecondary()) }
        for (level in criterion.levels) {
            val active = selected == level.points
            Text(
                "${level.label} (${level.points})",
                modifier = Modifier
                    .fillMaxWidth()
                    .clickable(enabled = !disabled) { onSelect(level.points) }
                    .padding(vertical = 6.dp),
                fontWeight = if (active) FontWeight.Bold else FontWeight.Normal,
                color = if (active) textPrimary() else textSecondary(),
            )
        }
    }
}
