package com.lextures.android.features.contenttools.governance

import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.lextures.android.R
import com.lextures.android.core.i18n.L
import com.lextures.android.core.lms.ContentToolGovernanceLogic
import com.lextures.android.features.contenttools.ToolPlaceholder
import com.lextures.android.features.contenttools.ToolPlaceholderReason

/** Neutral placeholder naming why a tool did not mount (CT.M9 policy-blocked states). */
@Composable
fun PolicyBlockedPlaceholder(
    decision: ContentToolGovernanceLogic.MountDecision,
    toolName: String? = null,
    onRefresh: (() -> Unit)? = null,
    modifier: Modifier = Modifier,
) {
    val message = reasonMessage(decision)
    Box(modifier = modifier.fillMaxWidth()) {
        ToolPlaceholder(
            reason = ToolPlaceholderReason.UNAVAILABLE,
            toolName = toolName,
            message = message,
            modifier = Modifier.fillMaxWidth(),
        )
        if (
            decision == ContentToolGovernanceLogic.MountDecision.BLOCK_STALE_POLICY &&
            onRefresh != null
        ) {
            OutlinedButton(
                onClick = onRefresh,
                modifier = Modifier
                    .align(Alignment.BottomEnd)
                    .padding(8.dp),
            ) {
                Text(L.text(R.string.mobile_contentTools_governance_refreshPolicy))
            }
        }
    }
}

@Composable
private fun reasonMessage(decision: ContentToolGovernanceLogic.MountDecision): String =
    when (decision) {
        ContentToolGovernanceLogic.MountDecision.MOUNT -> ""
        ContentToolGovernanceLogic.MountDecision.BLOCK_NOT_AVAILABLE,
        ContentToolGovernanceLogic.MountDecision.BLOCK_CAPABILITY,
        -> L.text(R.string.mobile_contentTools_governance_notAvailableInCourse)
        ContentToolGovernanceLogic.MountDecision.BLOCK_KILLED ->
            L.text(R.string.mobile_contentTools_governance_killed)
        ContentToolGovernanceLogic.MountDecision.BLOCK_BREAKER ->
            L.text(R.string.mobile_contentTools_governance_temporarilyUnavailable)
        ContentToolGovernanceLogic.MountDecision.BLOCK_TOMBSTONE,
        ContentToolGovernanceLogic.MountDecision.BLOCK_DEPRECATED,
        -> L.text(R.string.mobile_contentTools_governance_withdrawn)
        ContentToolGovernanceLogic.MountDecision.BLOCK_UNKNOWN ->
            L.text(R.string.mobile_contentTools_governance_unavailable)
        ContentToolGovernanceLogic.MountDecision.BLOCK_STALE_POLICY ->
            L.text(R.string.mobile_contentTools_governance_policyStale)
    }
