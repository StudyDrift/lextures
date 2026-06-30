package com.lextures.android.features.courses

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CalendarMonth
import androidx.compose.material.icons.filled.Person
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.core.design.LexturesType
import com.lextures.android.core.design.isDarkTheme
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.LmsDates
import com.lextures.android.features.home.CourseHeroImage

/** Course detail banner: hero image (when set) or cover gradient, with course metadata overlay. */
@Composable
fun CourseBanner(
    course: CourseSummary,
    accessToken: String?,
    modifier: Modifier = Modifier,
) {
    val hasHeroImage = !course.heroImageUrl.isNullOrBlank()
    val shape = RoundedCornerShape(24.dp)
    val dark = isDarkTheme()

    Box(
        modifier = modifier
            .fillMaxWidth()
            .shadow(
                elevation = if (dark) 0.dp else 7.dp,
                shape = shape,
                clip = false,
                ambientColor = Color(0xFF3A2E18).copy(alpha = 0.12f),
                spotColor = Color(0xFF3A2E18).copy(alpha = 0.12f),
            )
            .clip(shape),
    ) {
        CourseHeroImage(
            url = course.heroImageUrl,
            fallbackKey = course.courseCode,
            accessToken = accessToken,
            modifier = Modifier.matchParentSize(),
            height = null,
        )

        if (hasHeroImage) {
            Box(
                modifier = Modifier
                    .matchParentSize()
                    .background(
                        Brush.linearGradient(
                            colors = listOf(
                                Color.Black.copy(alpha = 0.55f),
                                Color.Black.copy(alpha = 0.18f),
                                Color.Black.copy(alpha = 0.05f),
                            ),
                        ),
                    ),
            )
        }

        Box(
            modifier = Modifier
                .size(140.dp)
                .align(Alignment.TopEnd)
                .offset(x = 44.dp, y = (-52).dp)
                .clip(CircleShape)
                .background(Color.White.copy(alpha = 0.08f)),
        )

        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(20.dp),
            verticalArrangement = Arrangement.spacedBy(7.dp),
        ) {
            Text(
                text = course.courseCode.uppercase(),
                fontSize = 11.sp,
                fontWeight = FontWeight.SemiBold,
                letterSpacing = 1.2.sp,
                color = Color.White.copy(alpha = 0.8f),
            )
            Text(
                text = course.title,
                style = LexturesType.display(22),
                color = Color.White,
            )
            if (course.description.isNotEmpty()) {
                Text(
                    text = course.description,
                    fontSize = 13.sp,
                    color = Color.White.copy(alpha = 0.85f),
                    maxLines = 3,
                    overflow = TextOverflow.Ellipsis,
                )
            }
            Row(
                modifier = Modifier.padding(top = 4.dp),
                horizontalArrangement = Arrangement.spacedBy(6.dp),
            ) {
                LmsDates.parse(course.startsAt)?.let {
                    CourseBannerChip(
                        text = LmsDates.shortDate(course.startsAt),
                        icon = Icons.Default.CalendarMonth,
                    )
                }
                course.viewerEnrollmentRoles.orEmpty().forEach { role ->
                    CourseBannerChip(
                        text = if (role.length <= 2) role.uppercase() else role.replaceFirstChar { it.uppercase() },
                        icon = Icons.Default.Person,
                    )
                }
            }
        }
    }
}

@Composable
private fun CourseBannerChip(
    text: String,
    icon: ImageVector,
) {
    Row(
        modifier = Modifier
            .clip(RoundedCornerShape(50))
            .background(Color.White.copy(alpha = 0.16f))
            .padding(horizontal = 9.dp, vertical = 4.dp),
        horizontalArrangement = Arrangement.spacedBy(4.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(icon, contentDescription = null, tint = Color.White, modifier = Modifier.size(12.dp))
        Text(
            text = text,
            fontSize = 12.sp,
            fontWeight = FontWeight.Medium,
            color = Color.White,
        )
    }
}
