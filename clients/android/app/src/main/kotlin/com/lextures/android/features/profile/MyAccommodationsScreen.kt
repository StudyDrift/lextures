package com.lextures.android.features.profile

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.LexturesType
import com.lextures.android.core.design.accentColor
import com.lextures.android.core.design.sceneBackground
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalLocalePreferences
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.MyAccommodation
import com.lextures.android.features.home.LmsCard
import kotlinx.coroutines.launch

/** Lists the student's currently active accommodations in plain language (FR-3). */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun MyAccommodationsScreen(
    session: AuthSession,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    val scope = rememberCoroutineScope()
    var loading by remember { mutableStateOf(true) }
    var loadFailed by remember { mutableStateOf(false) }
    var items by remember { mutableStateOf<List<MyAccommodation>>(emptyList()) }

    fun load() {
        val token = accessToken ?: run {
            loading = false
            loadFailed = true
            return
        }
        scope.launch {
            loading = true
            loadFailed = false
            try {
                items = LmsApi.fetchMyAccommodations(token).filter { !it.isEmpty }
            } catch (_: Exception) {
                loadFailed = true
            } finally {
                loading = false
            }
        }
    }

    LaunchedEffect(accessToken) { load() }

    Column(
        modifier = modifier
            .fillMaxSize()
            .background(sceneBackground()),
    ) {
        TopAppBar(
            title = { Text(L.text(R.string.mobile_accommodations_title)) },
            navigationIcon = {
                IconButton(onClick = onBack) {
                    Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = null)
                }
            },
        )

        when {
            loading -> Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                CircularProgressIndicator(color = accentColor())
            }

            loadFailed -> Column(
                modifier = Modifier.fillMaxSize().padding(32.dp),
                horizontalAlignment = Alignment.CenterHorizontally,
                verticalArrangement = Arrangement.Center,
            ) {
                Text(L.text(R.string.mobile_accommodations_loadError), color = textSecondary())
                TextButton(onClick = { load() }) { Text(L.text(R.string.mobile_common_retry)) }
            }

            items.isEmpty() -> Column(
                modifier = Modifier.fillMaxSize().padding(32.dp),
                horizontalAlignment = Alignment.CenterHorizontally,
                verticalArrangement = Arrangement.spacedBy(8.dp, Alignment.CenterVertically),
            ) {
                Text(
                    text = L.text(R.string.mobile_accommodations_emptyTitle),
                    style = LexturesType.display(18),
                    color = textPrimary(),
                )
                Text(
                    text = L.text(R.string.mobile_accommodations_emptyBody),
                    fontSize = 14.sp,
                    color = textSecondary(),
                )
            }

            else -> Column(
                modifier = Modifier
                    .fillMaxSize()
                    .verticalScroll(rememberScrollState())
                    .padding(16.dp),
                verticalArrangement = Arrangement.spacedBy(16.dp),
            ) {
                Text(
                    text = L.text(R.string.mobile_accommodations_intro),
                    fontSize = 14.sp,
                    color = textSecondary(),
                )
                items.forEach { AccommodationCard(it) }
            }
        }
    }
}

@Composable
private fun AccommodationCard(item: MyAccommodation) {
    val context = LocalContext.current
    val prefs = LocalLocalePreferences.current
    val localized = remember(prefs) { prefs.localizedContext(context) }

    val scopeTitle = item.courseCode?.takeIf { it.isNotEmpty() }
        ?.let { localized.getString(R.string.mobile_accommodations_scopeCourse, it) }
        ?: L.text(R.string.mobile_accommodations_scopeAll)

    val supports = buildList {
        if (item.hasExtendedTime) add(L.text(R.string.mobile_accommodations_extendedTime))
        if (item.hasExtraAttempts) add(L.text(R.string.mobile_accommodations_extraAttempts))
        if (item.hintsAlwaysAvailable) add(L.text(R.string.mobile_accommodations_hints))
        if (item.reducedDistractionRecommended) add(L.text(R.string.mobile_accommodations_reducedDistraction))
        if (item.speechToTextEnabled) add(L.text(R.string.mobile_accommodations_speechToText))
        if (item.ttsEnabled) add(L.text(R.string.mobile_accommodations_readAloud))
        if (item.dyslexiaDisplayEnabled) add(L.text(R.string.mobile_accommodations_dyslexiaDisplay))
        if (item.highContrastEnabled) add(L.text(R.string.mobile_accommodations_highContrast))
        if (item.reducedMotionEnabled) add(L.text(R.string.mobile_accommodations_reducedMotion))
        if (item.separateSetting) add(L.text(R.string.mobile_accommodations_separateSetting))
    }

    val from = item.effectiveFrom
    val until = item.effectiveUntil
    val window = when {
        from != null && until != null ->
            localized.getString(R.string.mobile_accommodations_windowBetween, from, until)
        from != null -> localized.getString(R.string.mobile_accommodations_windowFrom, from)
        until != null -> localized.getString(R.string.mobile_accommodations_windowUntil, until)
        else -> null
    }

    LmsCard {
        Row(
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            Icon(
                Icons.Default.CheckCircle,
                contentDescription = null,
                tint = accentColor(),
                modifier = Modifier.size(18.dp),
            )
            Text(text = scopeTitle, style = LexturesType.display(17), color = textPrimary())
        }
        supports.forEach { line ->
            Row(
                modifier = Modifier.fillMaxWidth().padding(top = 8.dp),
                horizontalArrangement = Arrangement.spacedBy(10.dp),
                verticalAlignment = Alignment.Top,
            ) {
                Text(text = "•", color = accentColor(), fontWeight = FontWeight.Bold)
                Text(text = line, fontSize = 14.sp, color = textPrimary())
            }
        }
        window?.let {
            HorizontalDivider(modifier = Modifier.padding(vertical = 8.dp))
            Text(text = it, fontSize = 12.sp, color = textSecondary())
        }
    }
}
