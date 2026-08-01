package com.lextures.android.features.contenttools.governance

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Button
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L

/** Native AI disclosure chrome above tool content (CT.M9 FR-6) — cannot be covered by sandboxed tools. */
@Composable
fun AIDisclosureBanner(
    mode: String,
    busy: Boolean,
    onAcknowledge: () -> Unit,
    onOptOut: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val title = L.text(R.string.mobile_contentTools_governance_aiDisclosureTitle)
    val ackLabel = if (mode.equals("acknowledge", ignoreCase = true)) {
        L.text(R.string.mobile_contentTools_governance_consentGrant)
    } else {
        L.text(R.string.mobile_contentTools_governance_continueWithAI)
    }
    Column(
        modifier = modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(8.dp))
            .background(textSecondary().copy(alpha = 0.08f))
            .padding(10.dp)
            .semantics { contentDescription = title },
        verticalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        Text(text = title, fontSize = 13.sp, color = textPrimary())
        Text(
            text = L.text(R.string.mobile_contentTools_governance_aiDisclosureBody),
            fontSize = 12.sp,
            color = textSecondary(),
        )
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            Button(onClick = onAcknowledge, enabled = !busy) {
                Text(ackLabel)
            }
            OutlinedButton(onClick = onOptOut, enabled = !busy) {
                Text(L.text(R.string.mobile_contentTools_governance_consentOptOut))
            }
        }
    }
}
