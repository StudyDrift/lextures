package com.lextures.android.features.contenttools.governance

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L

/** Prominent crisis support resources — never treated as a generic retryable error (CT.M9 FR-13). */
@Composable
fun CrisisResourcesView(
    resources: List<String> = emptyList(),
    modifier: Modifier = Modifier,
) {
    val title = L.text(R.string.mobile_contentTools_governance_crisisTitle)
    val body = L.text(R.string.mobile_contentTools_governance_crisisBody)
    Column(
        modifier = modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(8.dp))
            .background(LexturesColors.Coral.copy(alpha = 0.12f))
            .padding(12.dp)
            .semantics { contentDescription = "$title. $body" },
        verticalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        Text(text = title, fontSize = 13.sp, color = textPrimary())
        Text(text = body, fontSize = 12.sp, color = textSecondary())
        if (resources.isEmpty()) {
            Text(
                text = L.text(R.string.mobile_contentTools_governance_crisisDefaultResource),
                fontSize = 12.sp,
                color = textPrimary(),
            )
        } else {
            resources.forEach { line ->
                Text(text = line, fontSize = 12.sp, color = textPrimary())
            }
        }
    }
}
