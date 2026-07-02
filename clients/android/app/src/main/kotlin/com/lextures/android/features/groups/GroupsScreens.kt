package com.lextures.android.features.groups

import android.content.Intent
import android.net.Uri
import android.webkit.WebView
import android.webkit.WebViewClient
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.ChevronRight
import androidx.compose.material.icons.filled.Description
import androidx.compose.material.icons.filled.Groups
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.FilterChip
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.viewinterop.AndroidView
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.i18n.L
import com.lextures.android.core.config.AppConfiguration
import com.lextures.android.core.design.OfflineBanner
import com.lextures.android.core.design.StalenessChip
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.LocalLocalePreferences
import com.lextures.android.core.lms.CollabDoc
import com.lextures.android.core.lms.CollabDocType
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.GroupFeedContext
import com.lextures.android.core.lms.GroupMemberRow
import com.lextures.android.core.lms.GroupPublic
import com.lextures.android.core.lms.GroupSpaceTab
import com.lextures.android.core.lms.GroupsLogic
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.offline.OfflineCacheKey
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.features.feed.FeedChannelsScreen
import com.lextures.android.core.lms.FilePreviewTarget
import com.lextures.android.features.files.CourseFilesScreen
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsSkeletonList
import kotlinx.serialization.builtins.ListSerializer

@Composable
fun CourseGroupsSection(session: AuthSession, course: CourseSummary, modifier: Modifier = Modifier) {
    val accessToken by session.accessToken.collectAsState()
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    val offline = remember { OfflineService.get(context) }
    val isOnline by offline.networkMonitor.isOnline.collectAsState()

    var groups by remember { mutableStateOf<List<GroupPublic>>(emptyList()) }
    var cacheLabel by remember { mutableStateOf<String?>(null) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }
    var openGroup by remember { mutableStateOf<GroupPublic?>(null) }

    suspend fun load() {
        val token = accessToken ?: return
        loading = true
        errorMessage = null
        try {
            val result = offline.cachedFetch(
                key = OfflineCacheKey.myGroups(course.courseCode),
                accessToken = token,
                serializer = ListSerializer(GroupPublic.serializer()),
            ) {
                if (course.viewerIsStaff) {
                    LmsApi.fetchAllGroups(course.courseCode, token)
                } else {
                    LmsApi.fetchMyGroups(course.courseCode, token)
                }
            }
            groups = result.first
            cacheLabel = result.second?.takeIf { it.isStale(isOnline) }?.lastUpdatedLabel()
        } catch (e: Exception) {
            errorMessage = session.mapError(e)
        } finally {
            loading = false
        }
    }

    LaunchedEffect(accessToken) { load() }

    openGroup?.let { group ->
        GroupSpaceScreen(session = session, course = course, group = group, onBack = { openGroup = null })
        return
    }

    Column(modifier = modifier, verticalArrangement = Arrangement.spacedBy(12.dp)) {
        if (!isOnline) OfflineBanner()
        cacheLabel?.let { StalenessChip(label = it) }
        errorMessage?.let { LmsErrorBanner(message = it) }

        when {
            loading && groups.isEmpty() -> LmsSkeletonList(count = 2)
            groups.isEmpty() -> LmsEmptyState(
                icon = Icons.Default.Groups,
                title = groupsEmptyTitle(context, localePrefs),
                message = groupsEmptyMessage(context, localePrefs),
            )
            else -> GroupsLogic.sortedGroups(groups).forEach { group ->
                LmsCard(modifier = Modifier.clickable { openGroup = group }) {
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(12.dp),
                    ) {
                        GroupAvatarBadge(userId = group.id, label = group.name)
                        Column(modifier = Modifier.weight(1f)) {
                            Text(group.name, fontWeight = FontWeight.SemiBold, color = textPrimary())
                            Text(
                                groupsMemberCount(context, localePrefs, group.memberCount),
                                color = textSecondary(),
                            )
                        }
                        Icon(Icons.Default.ChevronRight, contentDescription = null, tint = textSecondary())
                    }
                }
            }
        }
    }
}

@Composable
fun CourseCollabDocsSection(session: AuthSession, course: CourseSummary, modifier: Modifier = Modifier) {
    val accessToken by session.accessToken.collectAsState()
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    val offline = remember { OfflineService.get(context) }
    val isOnline by offline.networkMonitor.isOnline.collectAsState()

    var docs by remember { mutableStateOf<List<CollabDoc>>(emptyList()) }
    var cacheLabel by remember { mutableStateOf<String?>(null) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }
    var openDoc by remember { mutableStateOf<CollabDoc?>(null) }

    val courseDocs = remember(docs) { GroupsLogic.courseCollabDocs(docs) }

    suspend fun load() {
        val token = accessToken ?: return
        loading = true
        errorMessage = null
        try {
            val result = offline.cachedFetch(
                key = OfflineCacheKey.collabDocs(course.courseCode),
                accessToken = token,
                serializer = ListSerializer(CollabDoc.serializer()),
            ) { LmsApi.fetchCollabDocs(course.courseCode, token) }
            docs = result.first
            cacheLabel = result.second?.takeIf { it.isStale(isOnline) }?.lastUpdatedLabel()
        } catch (e: Exception) {
            errorMessage = session.mapError(e)
        } finally {
            loading = false
        }
    }

    LaunchedEffect(accessToken) { load() }

    openDoc?.let { doc ->
        CollabDocScreen(session = session, course = course, doc = doc, onBack = { openDoc = null })
        return
    }

    Column(modifier = modifier, verticalArrangement = Arrangement.spacedBy(12.dp)) {
        if (!isOnline) OfflineBanner()
        cacheLabel?.let { StalenessChip(label = it) }
        errorMessage?.let { LmsErrorBanner(message = it) }

        when {
            loading && courseDocs.isEmpty() -> LmsSkeletonList(count = 2)
            courseDocs.isEmpty() -> LmsEmptyState(
                icon = Icons.Default.Description,
                title = collabDocsEmptyTitle(context, localePrefs),
                message = collabDocsEmptyMessage(context, localePrefs),
            )
            else -> courseDocs.forEach { doc ->
                LmsCard(modifier = Modifier.clickable { openDoc = doc }) {
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Column(modifier = Modifier.weight(1f)) {
                            Text(doc.title, fontWeight = FontWeight.SemiBold, color = textPrimary())
                            Text(
                                if (doc.docType == CollabDocType.Whiteboard) "Whiteboard" else "Rich text",
                                color = textSecondary(),
                            )
                        }
                        Icon(Icons.Default.ChevronRight, contentDescription = null, tint = textSecondary())
                    }
                }
            }
        }
    }
}

@Composable
fun GroupSpaceScreen(
    session: AuthSession,
    course: CourseSummary,
    group: GroupPublic,
    onBack: () -> Unit,
) {
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    var tab by remember { mutableStateOf(GroupSpaceTab.Discussion) }
    var openDoc by remember { mutableStateOf<CollabDoc?>(null) }
    var previewTarget by remember { mutableStateOf<FilePreviewTarget?>(null) }

    previewTarget?.let { target ->
        com.lextures.android.features.files.FilePreviewScreen(
            session = session,
            target = target,
            onBack = { previewTarget = null },
        )
        return
    }

    openDoc?.let { doc ->
        CollabDocScreen(session = session, course = course, doc = doc, onBack = { openDoc = null })
        return
    }

    Column(modifier = Modifier.fillMaxSize()) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            TextButton(onClick = onBack) { Text("←") }
            Text(group.name, fontWeight = FontWeight.SemiBold, color = textPrimary())
        }

        Row(
            modifier = Modifier
                .fillMaxWidth()
                .horizontalScroll(rememberScrollState())
                .padding(horizontal = 16.dp),
            horizontalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            GroupSpaceTab.entries.forEach { item ->
                FilterChip(
                    selected = tab == item,
                    onClick = { tab = item },
                    label = {
                        Text(
                            when (item) {
                                GroupSpaceTab.Members -> "Members"
                                GroupSpaceTab.Discussion -> "Discussion"
                                GroupSpaceTab.Files -> "Files"
                                GroupSpaceTab.Docs -> "Docs"
                            },
                        )
                    },
                )
            }
        }

        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(16.dp),
        ) {
            when (tab) {
                GroupSpaceTab.Members -> GroupMembersTab(session = session, course = course, group = group)
                GroupSpaceTab.Discussion -> FeedChannelsScreen(
                    session = session,
                    course = course,
                    groupContext = GroupFeedContext(group.id, group.name),
                )
                GroupSpaceTab.Files -> Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    Text(
                        L.text(context, localePrefs, com.lextures.android.R.string.mobile_groups_filesHint),
                        color = textSecondary(),
                    )
                    CourseFilesScreen(
                        session = session,
                        course = course,
                        onOpenPreview = { previewTarget = it },
                    )
                }
                GroupSpaceTab.Docs -> GroupDocsTab(
                    session = session,
                    course = course,
                    group = group,
                    onOpenDoc = { openDoc = it },
                )
            }
        }
    }
}

@Composable
private fun GroupMembersTab(session: AuthSession, course: CourseSummary, group: GroupPublic) {
    val accessToken by session.accessToken.collectAsState()
    var members by remember { mutableStateOf<List<GroupMemberRow>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var errorMessage by remember { mutableStateOf<String?>(null) }

    LaunchedEffect(accessToken, group.id) {
        val token = accessToken ?: return@LaunchedEffect
        loading = true
        errorMessage = null
        try {
            val roster = LmsApi.fetchFeedRoster(course.courseCode, token)
            val channel = LmsApi.fetchGroupFeedChannels(course.courseCode, group.id, token).firstOrNull()
            val authorIds = if (channel != null) {
                GroupsLogic.collectMessageAuthorIds(
                    LmsApi.fetchGroupFeedMessages(course.courseCode, group.id, channel.id, token),
                )
            } else {
                emptySet()
            }
            members = GroupsLogic.memberRows(roster, authorIds)
        } catch (e: Exception) {
            errorMessage = session.mapError(e)
        } finally {
            loading = false
        }
    }

    Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
        errorMessage?.let { LmsErrorBanner(message = it) }
        when {
            loading && members.isEmpty() -> LmsSkeletonList(count = 3)
            members.isEmpty() -> LmsEmptyState(
                icon = Icons.Default.Groups,
                title = "No active members yet",
                message = "This group has ${group.memberCount} members.",
            )
            else -> members.forEach { member ->
                LmsCard {
                    Row(horizontalArrangement = Arrangement.spacedBy(12.dp), verticalAlignment = Alignment.CenterVertically) {
                        GroupAvatarBadge(userId = member.id, label = member.displayName)
                        Column {
                            Text(member.displayName, fontWeight = FontWeight.SemiBold, color = textPrimary())
                            Text(member.email, color = textSecondary())
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun GroupDocsTab(
    session: AuthSession,
    course: CourseSummary,
    group: GroupPublic,
    onOpenDoc: (CollabDoc) -> Unit,
) {
    val accessToken by session.accessToken.collectAsState()
    val context = LocalContext.current
    val offline = remember { OfflineService.get(context) }
    var docs by remember { mutableStateOf<List<CollabDoc>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }

    LaunchedEffect(accessToken) {
        val token = accessToken ?: return@LaunchedEffect
        loading = true
        try {
            val result = offline.cachedFetch(
                key = OfflineCacheKey.collabDocs(course.courseCode),
                accessToken = token,
                serializer = ListSerializer(CollabDoc.serializer()),
            ) { LmsApi.fetchCollabDocs(course.courseCode, token) }
            docs = result.first
        } finally {
            loading = false
        }
    }

    val groupDocs = remember(docs, group.id) { GroupsLogic.groupCollabDocs(docs, group.id) }
    when {
        loading && groupDocs.isEmpty() -> LmsSkeletonList(count = 2)
        groupDocs.isEmpty() -> LmsEmptyState(
            icon = Icons.Default.Description,
            title = collabDocsEmptyTitle(context, LocalLocalePreferences.current),
            message = "No collaborative documents for this group yet.",
        )
        else -> Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
            groupDocs.forEach { doc ->
                LmsCard(modifier = Modifier.clickable { onOpenDoc(doc) }) {
                    Text(doc.title, fontWeight = FontWeight.SemiBold, color = textPrimary())
                }
            }
        }
    }
}

@Composable
fun CollabDocScreen(
    session: AuthSession,
    course: CourseSummary,
    doc: CollabDoc,
    onBack: () -> Unit,
) {
    val accessToken by session.accessToken.collectAsState()
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    val webPath = GroupsLogic.collabDocWebPath(course.courseCode, doc.id)
    val webUrl = AppConfiguration.webUrl(webPath)

    Column(modifier = Modifier.fillMaxSize()) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            TextButton(onClick = onBack) { Text("←") }
            Text(doc.title, fontWeight = FontWeight.SemiBold, color = textPrimary())
        }

        if (doc.docType == CollabDocType.Whiteboard) {
            LmsEmptyState(
                icon = Icons.Default.Description,
                title = "Whiteboard",
                message = "Open \"${doc.title}\" on the web to edit the whiteboard.",
                modifier = Modifier.weight(1f),
            )
        } else {
            AndroidView(
                modifier = Modifier
                    .weight(1f)
                    .fillMaxWidth(),
                factory = { ctx ->
                    WebView(ctx).apply {
                        settings.javaScriptEnabled = true
                        webViewClient = WebViewClient()
                        val token = accessToken
                        if (token != null) {
                            loadUrl(webUrl, mapOf("Authorization" to "Bearer $token"))
                        } else {
                            loadUrl(webUrl)
                        }
                    }
                },
                update = { webView ->
                    val token = accessToken
                    if (token != null) {
                        webView.loadUrl(webUrl, mapOf("Authorization" to "Bearer $token"))
                    }
                },
            )
        }

        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(12.dp),
            horizontalArrangement = Arrangement.End,
        ) {
            TextButton(
                onClick = {
                    context.startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(webUrl)))
                },
            ) { Text(collabDocsOpenOnWeb(context, localePrefs)) }
        }
    }
}

@Composable
fun GroupAvatarBadge(userId: String, label: String) {
    val hue = GroupsLogic.avatarHue(userId)
    Box(
        modifier = Modifier
            .size(36.dp)
            .clip(CircleShape)
            .background(Color.hsv(hue, 0.58f, 0.48f)),
        contentAlignment = Alignment.Center,
    ) {
        Text(
            GroupsLogic.displayInitials(label),
            color = Color.White,
            fontWeight = FontWeight.Bold,
        )
    }
}

