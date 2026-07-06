package com.lextures.android.core.design

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.lextures.android.R
import com.lextures.android.core.i18n.L

/** Sticky unsaved-changes banner mirroring web (M13.1). */
@Composable
fun UnsavedChangesBanner(
    isSaving: Boolean,
    onSave: () -> Unit,
    onDiscard: () -> Unit,
    modifier: Modifier = Modifier,
) {
    Surface(
        modifier = modifier.fillMaxWidth(),
        tonalElevation = 4.dp,
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 16.dp, vertical = 12.dp),
            horizontalArrangement = Arrangement.spacedBy(12.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                text = L.text(R.string.mobile_courseSettings_unsavedChanges),
                style = MaterialTheme.typography.bodyMedium,
                modifier = Modifier.weight(1f),
            )
            TextButton(onClick = onDiscard) {
                Text(L.text(R.string.mobile_courseSettings_discard))
            }
            Button(onClick = onSave, enabled = !isSaving) {
                if (isSaving) {
                    CircularProgressIndicator(modifier = Modifier.padding(4.dp))
                } else {
                    Text(L.text(R.string.mobile_courseSettings_saveChanges))
                }
            }
        }
    }
}
