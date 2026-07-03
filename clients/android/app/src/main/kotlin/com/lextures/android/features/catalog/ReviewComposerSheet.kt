package com.lextures.android.features.catalog

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Star
import androidx.compose.material.icons.outlined.Star
import androidx.compose.material3.Button
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalePreferences
import com.lextures.android.core.lms.CourseReviewLogic
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.features.home.LmsErrorBanner
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ReviewComposerSheet(
    session: AuthSession,
    localePrefs: LocalePreferences,
    courseCode: String,
    courseTitle: String,
    initialRating: Int = 0,
    initialText: String = "",
    hasReview: Boolean = false,
    canEdit: Boolean = true,
    onDismiss: () -> Unit,
    onSubmitted: () -> Unit,
) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
    val accessToken = session.accessToken.value

    var rating by remember { mutableIntStateOf(initialRating) }
    var reviewText by remember { mutableStateOf(initialText) }
    var submitting by remember { mutableStateOf(false) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var thanks by remember { mutableStateOf(false) }

    ModalBottomSheet(onDismissRequest = onDismiss, sheetState = sheetState) {
        Column(Modifier.padding(16.dp)) {
            Text(
                if (hasReview && canEdit) {
                    L.text(context, localePrefs, R.string.mobile_reviews_editTitle)
                } else {
                    L.text(context, localePrefs, R.string.mobile_reviews_writeTitle)
                },
            )
            Text(courseTitle, modifier = Modifier.padding(bottom = 12.dp))

            if (thanks) {
                Text(L.text(context, localePrefs, R.string.mobile_reviews_thanks))
            } else {
                errorMessage?.let { LmsErrorBanner(message = it, modifier = Modifier.padding(bottom = 8.dp)) }
                Row(Modifier.fillMaxWidth()) {
                    (1..5).forEach { star ->
                        IconButton(onClick = { rating = star }) {
                            Icon(
                                imageVector = if (star <= rating) Icons.Filled.Star else Icons.Outlined.Star,
                                contentDescription = CourseReviewLogic.starLabel(star),
                            )
                        }
                    }
                }
                OutlinedTextField(
                    value = reviewText,
                    onValueChange = { reviewText = it },
                    modifier = Modifier.fillMaxWidth().padding(vertical = 8.dp),
                    label = { Text(L.text(context, localePrefs, R.string.mobile_reviews_textLabel)) },
                    minLines = 3,
                )
                Button(
                    onClick = {
                        CourseReviewLogic.validateRating(rating)?.let {
                            errorMessage = it
                            return@Button
                        }
                        CourseReviewLogic.validateReviewText(reviewText)?.let {
                            errorMessage = it
                            return@Button
                        }
                        val token = accessToken ?: return@Button
                        submitting = true
                        errorMessage = null
                        scope.launch {
                            try {
                                LmsApi.submitCourseReview(courseCode, rating, reviewText, token)
                                thanks = true
                                onSubmitted()
                            } catch (_: Exception) {
                                errorMessage = L.text(context, localePrefs, R.string.mobile_reviews_submitError)
                            } finally {
                                submitting = false
                            }
                        }
                    },
                    enabled = !submitting,
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    Text(
                        if (submitting) {
                            L.text(context, localePrefs, R.string.mobile_reviews_submitting)
                        } else {
                            L.text(context, localePrefs, R.string.mobile_reviews_submit)
                        },
                    )
                }
            }
        }
    }
}