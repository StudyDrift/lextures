package com.lextures.android.features.contenttools.governance

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.material3.Button
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L

/** Blocked AI composer state with reachable consent action (CT.M9 FR-7/FR-8). */
@Composable
fun ConsentGateView(
    busy: Boolean,
    modifier: Modifier = Modifier,
    onGrant: () -> Unit,
) {
    val required = L.text(R.string.mobile_contentTools_governance_consentRequired)
    Column(
        modifier = modifier
            .fillMaxWidth()
            .semantics { contentDescription = required },
        verticalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        Text(text = required, fontSize = 12.sp, color = textSecondary())
        Button(
            onClick = onGrant,
            enabled = !busy,
            modifier = Modifier.heightIn(min = 44.dp),
        ) {
            Text(L.text(R.string.mobile_contentTools_governance_consentGrant))
        }
    }
}
