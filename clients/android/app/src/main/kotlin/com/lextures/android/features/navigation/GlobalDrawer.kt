package com.lextures.android.features.navigation

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.foundation.layout.height
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Search
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.cardBackground
import com.lextures.android.core.design.fieldBorder
import com.lextures.android.core.design.isDarkTheme
import com.lextures.android.core.design.sceneBackground
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.resolvedInitials
import com.lextures.android.core.design.LocalUIModeStore
import com.lextures.android.core.design.UIMode
import com.lextures.android.core.design.UIModeLogic
import com.lextures.android.core.navigation.MobileDestinations
import com.lextures.android.features.home.CourseHeroImage
import com.lextures.android.features.home.HomeShellState
import com.lextures.android.features.home.LmsAvatarChip

/**
 * App-wide navigation drawer mirroring the Lextures web sidebar: brand header, search,
 * and grouped destinations. Selecting a row switches the top-level pane.
 */
@Composable
fun GlobalDrawer(shell: HomeShellState, accessToken: String?, modifier: Modifier = Modifier) {
    val uiModeStore = LocalUIModeStore.current
    val uiMode = uiModeStore.effectiveMode(shell.activeRoleContext)
    Column(
        modifier = modifier
            .fillMaxSize()
            .background(sceneBackground()),
    ) {
        // Header: brand + avatar.
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(start = 16.dp, end = 16.dp, top = 16.dp, bottom = 14.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            Icon(
                painter = painterResource(R.drawable.launch_logo),
                contentDescription = null,
                tint = androidx.compose.ui.graphics.Color.Unspecified,
                modifier = Modifier.size(32.dp),
            )
            Text("Lextures", fontSize = 20.sp, fontWeight = FontWeight.Bold, color = textPrimary())
            androidx.compose.foundation.layout.Spacer(Modifier.weight(1f))
            LmsAvatarChip(
                initials = shell.accountProfile?.resolvedInitials() ?: shell.profile?.initials ?: "··",
                onClick = { shell.select(com.lextures.android.core.navigation.RootDestination.Profile) },
                size = 34,
                avatarUrl = shell.accountProfile?.avatarUrl,
            )
        }

        if (MobileDestinations.showsUniversalSearch(uiMode)) {
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(horizontal = 14.dp)
                    .clip(RoundedCornerShape(12.dp))
                    .background(cardBackground())
                    .border(1.dp, fieldBorder(), RoundedCornerShape(12.dp))
                    .clickable {
                        shell.closeDrawer()
                        shell.showUniversalSearch = true
                    }
                    .padding(horizontal = 12.dp, vertical = 10.dp),
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(10.dp),
            ) {
                Icon(Icons.Default.Search, contentDescription = null, tint = textSecondary(), modifier = Modifier.size(18.dp))
                Text(L.text(R.string.mobile_ia_search), color = textSecondary(), fontSize = 14.sp)
            }
        }
        // (pinned courses rendered inside the grouped list below)

        Column(
            modifier = Modifier
                .fillMaxWidth()
                .verticalScroll(rememberScrollState())
                .padding(horizontal = 10.dp, vertical = 8.dp),
        ) {
            // Pinned course covers sit directly under search, above the nav (web-parity).
            if (shell.pinnedCourses.isNotEmpty()) {
                PinnedCourseTiles(
                    courses = shell.pinnedCourses,
                    activeCourseCode = shell.activeCourse?.courseCode,
                    accessToken = accessToken,
                    onOpen = { course ->
                        shell.closeDrawer()
                        shell.openDeepLink(
                            com.lextures.android.core.routing.DeepLinkDestination.Course(course.courseCode),
                        )
                    },
                )
            }
            shell.globalDrawerGroups.forEach { group ->
                group.titleRes?.let { DrawerGroupHeader(drawerString(it)) }
                group.items.forEach { item ->
                    DrawerRow(
                        label = UIModeLogic.drawerLabel(item, uiMode),
                        icon = rootDestinationIcon(item),
                        selected = shell.rootDestination == item,
                        onClick = { shell.select(item) },
                        badge = if (item.showsInboxBadge) shell.unreadInbox else 0,
                        uiMode = uiMode,
                    )
                }
            }
        }
    }
}

/** Pinned course covers as a wrapping row of rounded tiles — no labels (web-parity). */
@OptIn(androidx.compose.foundation.layout.ExperimentalLayoutApi::class)
@Composable
private fun PinnedCourseTiles(
    courses: List<CourseSummary>,
    activeCourseCode: String?,
    accessToken: String?,
    onOpen: (CourseSummary) -> Unit,
) {
    androidx.compose.foundation.layout.FlowRow(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 2.dp, vertical = 2.dp),
        horizontalArrangement = Arrangement.spacedBy(8.dp),
        verticalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        courses.forEach { course ->
            val active = course.courseCode == activeCourseCode
            Box(
                modifier = Modifier
                    .width(88.dp)
                    .height(52.dp)
                    .clip(RoundedCornerShape(14.dp))
                    .border(
                        width = if (active) 2.dp else 1.dp,
                        color = if (active) LexturesColors.BrandTeal else fieldBorder(),
                        shape = RoundedCornerShape(14.dp),
                    )
                    .clickable { onOpen(course) }
                    .semantics { contentDescription = course.displayTitle },
            ) {
                CourseHeroImage(
                    url = course.heroImageUrl,
                    fallbackKey = course.courseCode,
                    accessToken = accessToken,
                    height = null,
                )
            }
        }
    }
}
