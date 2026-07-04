package com.lextures.android.features.reader

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ExposedDropdownMenuBox
import androidx.compose.material3.ExposedDropdownMenuDefaults
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.TranslationCoverageLocale
import kotlinx.coroutines.launch

sealed class ContentTranslationMode {
    data class Ugc(val target: UgcTranslationTarget, val accessToken: String) : ContentTranslationMode()
    data class CourseContent(
        val courseCode: String,
        val accessToken: String,
        val onReload: (suspend () -> Unit)?,
    ) : ContentTranslationMode()
}

@Composable
fun ContentTranslationControls(mode: ContentTranslationMode, modifier: Modifier = Modifier) {
    when (mode) {
        is ContentTranslationMode.Ugc -> UgcTranslationControls(mode.target, mode.accessToken, modifier)
        is ContentTranslationMode.CourseContent -> CourseLocalePicker(
            courseCode = mode.courseCode,
            accessToken = mode.accessToken,
            onReload = mode.onReload,
            modifier = modifier,
        )
    }
}

@Composable
private fun UgcTranslationControls(
    target: UgcTranslationTarget,
    accessToken: String,
    modifier: Modifier = Modifier,
) {
    var state by remember { mutableStateOf(TranslateUiState.IDLE) }
    var translated by remember { mutableStateOf<String?>(null) }
    var sourceLang by remember { mutableStateOf<String?>(null) }
    var showOriginal by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()

    Column(modifier = modifier, verticalArrangement = Arrangement.spacedBy(6.dp)) {
        if (state == TranslateUiState.DONE && translated != null && !showOriginal) {
            Text(translated!!)
            Text("Machine translated", color = textSecondary())
            sourceLang?.let { Text("From ${ReaderLogic.localeLabel(it)}", color = textSecondary()) }
        }
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp), verticalAlignment = Alignment.CenterVertically) {
            when (state) {
                TranslateUiState.IDLE, TranslateUiState.ERROR -> TextButton(onClick = {
                    if (target.text.isBlank()) return@TextButton
                    state = TranslateUiState.LOADING
                    scope.launch {
                        try {
                            val result = LmsApi.translateContent(
                                contentType = target.contentType,
                                contentId = target.contentId,
                                targetLang = target.targetLang,
                                text = target.text,
                                accessToken = accessToken,
                            )
                            translated = result.translated
                            sourceLang = result.sourceLang
                            showOriginal = false
                            state = TranslateUiState.DONE
                        } catch (_: Exception) {
                            state = TranslateUiState.ERROR
                        }
                    }
                }) { Text("Translate") }
                TranslateUiState.LOADING -> {
                    CircularProgressIndicator()
                    Text("Translating…", color = textSecondary())
                }
                TranslateUiState.DONE -> TextButton(onClick = { showOriginal = !showOriginal }) {
                    Text(if (showOriginal) "Show translation" else "Show original")
                }
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun CourseLocalePicker(
    courseCode: String,
    accessToken: String,
    onReload: (suspend () -> Unit)?,
    modifier: Modifier = Modifier,
) {
    var locales by remember { mutableStateOf<List<TranslationCoverageLocale>>(emptyList()) }
    var selected by remember { mutableStateOf("") }
    var expanded by remember { mutableStateOf(false) }
    var saving by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()

    LaunchedEffect(courseCode, accessToken) {
        locales = runCatching {
            LmsApi.fetchTranslationCoverage(courseCode, accessToken).filter { it.percent > 0 }
        }.getOrDefault(emptyList())
    }

    if (locales.isEmpty()) return

    ExposedDropdownMenuBox(expanded = expanded, onExpandedChange = { expanded = it }, modifier = modifier) {
        TextButton(
            onClick = { expanded = true },
            enabled = !saving,
            modifier = Modifier.menuAnchor(),
        ) {
            Text(
                if (selected.isEmpty()) "Content language" else ReaderLogic.localeLabel(selected),
            )
            ExposedDropdownMenuDefaults.TrailingIcon(expanded)
        }
        ExposedDropdownMenu(expanded = expanded, onDismissRequest = { expanded = false }) {
            DropdownMenuItem(
                text = { Text("Original") },
                onClick = {
                    expanded = false
                    selected = ""
                    saving = true
                    scope.launch {
                        runCatching { LmsApi.patchMyContentLocale(courseCode, null, accessToken) }
                        onReload?.invoke()
                        saving = false
                    }
                },
            )
            locales.forEach { locale ->
                DropdownMenuItem(
                    text = { Text("${ReaderLogic.localeLabel(locale.targetLocale)} (${locale.percent.toInt()}%)") },
                    onClick = {
                        expanded = false
                        selected = locale.targetLocale
                        saving = true
                        scope.launch {
                            runCatching {
                                LmsApi.patchMyContentLocale(courseCode, locale.targetLocale, accessToken)
                            }
                            onReload?.invoke()
                            saving = false
                        }
                    },
                )
            }
        }
    }
}

private enum class TranslateUiState { IDLE, LOADING, DONE, ERROR }