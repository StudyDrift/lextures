package com.lextures.android.features.contenttools.governance

import android.content.Context
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ExposedDropdownMenuBox
import androidx.compose.material3.ExposedDropdownMenuDefaults
import androidx.compose.material3.MenuAnchorType
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

/** Student report sheet — category + optional note (CT.M9 FR-10). */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ReportSheet(
    categories: List<String>,
    onSubmit: suspend (category: String, note: String?) -> Boolean,
    onDismiss: () -> Unit,
) {
    val scope = rememberCoroutineScope()
    val context = LocalContext.current
    var category by remember { mutableStateOf(categories.firstOrNull() ?: "other") }
    var note by remember { mutableStateOf("") }
    var busy by remember { mutableStateOf(false) }
    var errorText by remember { mutableStateOf<String?>(null) }
    var done by remember { mutableStateOf(false) }
    var menuOpen by remember { mutableStateOf(false) }
    val reportError = L.text(R.string.mobile_contentTools_governance_reportError)

    LaunchedEffect(categories) {
        if (category.isEmpty() || category !in categories) {
            category = categories.firstOrNull() ?: "other"
        }
    }

    AlertDialog(
        onDismissRequest = { if (!busy) onDismiss() },
        title = { Text(L.text(R.string.mobile_contentTools_governance_reportTitle)) },
        text = {
            Column(
                Modifier.fillMaxWidth(),
                verticalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                ExposedDropdownMenuBox(
                    expanded = menuOpen && !done,
                    onExpandedChange = { if (!done) menuOpen = it },
                ) {
                    OutlinedTextField(
                        value = categoryLabel(context, category),
                        onValueChange = {},
                        readOnly = true,
                        enabled = !done,
                        label = { Text(L.text(R.string.mobile_contentTools_governance_reportCategory)) },
                        trailingIcon = { ExposedDropdownMenuDefaults.TrailingIcon(expanded = menuOpen) },
                        modifier = Modifier
                            .menuAnchor(MenuAnchorType.PrimaryNotEditable)
                            .fillMaxWidth(),
                    )
                    ExposedDropdownMenu(
                        expanded = menuOpen && !done,
                        onDismissRequest = { menuOpen = false },
                    ) {
                        categories.forEach { cat ->
                            DropdownMenuItem(
                                text = { Text(categoryLabel(context, cat)) },
                                onClick = {
                                    category = cat
                                    menuOpen = false
                                },
                            )
                        }
                    }
                }
                OutlinedTextField(
                    value = note,
                    onValueChange = { note = it },
                    enabled = !done,
                    label = { Text(L.text(R.string.mobile_contentTools_governance_reportNote)) },
                    placeholder = {
                        Text(L.text(R.string.mobile_contentTools_governance_reportNotePlaceholder))
                    },
                    modifier = Modifier.fillMaxWidth(),
                    minLines = 3,
                )
                errorText?.let {
                    Text(text = it, fontSize = 12.sp, color = LexturesColors.Coral)
                }
                if (done) {
                    Text(
                        text = L.text(R.string.mobile_contentTools_governance_reportThanks),
                        fontSize = 12.sp,
                        color = textSecondary(),
                    )
                }
            }
        },
        confirmButton = {
            TextButton(
                enabled = !busy && !done,
                onClick = {
                    scope.launch {
                        busy = true
                        errorText = null
                        val noteTrim = note.trim()
                        val ok = onSubmit(category, noteTrim.ifEmpty { null })
                        busy = false
                        if (ok) {
                            done = true
                            delay(600)
                            onDismiss()
                        } else {
                            errorText = reportError
                        }
                    }
                },
            ) {
                Text(L.text(R.string.mobile_contentTools_governance_reportSubmit))
            }
        },
        dismissButton = {
            TextButton(onClick = onDismiss, enabled = !busy) {
                Text(L.text(R.string.mobile_contentTools_runtime_cancel))
            }
        },
    )
}

@Composable
private fun categoryLabel(context: Context, raw: String): String {
    val resName = "mobile_contentTools_governance_reportCategory_$raw"
    val id = context.resources.getIdentifier(resName, "string", context.packageName)
    return if (id != 0) L.text(id) else raw
}
