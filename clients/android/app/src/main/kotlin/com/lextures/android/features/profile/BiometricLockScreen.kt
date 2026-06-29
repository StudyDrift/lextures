package com.lextures.android.features.profile

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Fingerprint
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.fragment.app.FragmentActivity
import com.lextures.android.R
import com.lextures.android.core.auth.BiometricGate
import com.lextures.android.core.design.AuthPrimaryButton
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.LexturesType
import com.lextures.android.core.design.accentColor
import com.lextures.android.core.design.sceneBackground
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalLocalePreferences
import kotlinx.coroutines.launch

@Composable
fun BiometricLockScreen(
    biometricGate: BiometricGate,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val localePreferences = LocalLocalePreferences.current
    val activity = context as? FragmentActivity
    val scope = rememberCoroutineScope()
    var isUnlocking by remember { mutableStateOf(false) }

    fun attemptUnlock() {
        val host = activity ?: return
        if (isUnlocking) return
        scope.launch {
            isUnlocking = true
            try {
                biometricGate.unlock(host)
            } finally {
                isUnlocking = false
            }
        }
    }

    LaunchedEffect(activity) {
        attemptUnlock()
    }

    Column(
        modifier = modifier
            .fillMaxSize()
            .background(sceneBackground())
            .padding(32.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(24.dp, Alignment.CenterVertically),
    ) {
        Icon(
            imageVector = Icons.Default.Fingerprint,
            contentDescription = null,
            tint = accentColor(),
            modifier = Modifier.padding(bottom = 8.dp),
        )
        Text(
            text = L.text(context, localePreferences, R.string.mobile_biometric_lockedTitle),
            style = LexturesType.display(24, FontWeight.SemiBold),
            color = textPrimary(),
            textAlign = TextAlign.Center,
        )
        Text(
            text = localePreferences.localizedContext(context).getString(
                R.string.mobile_biometric_lockedSubtitle,
                biometricGate.biometryLabel(context),
            ),
            fontSize = 14.sp,
            color = textSecondary(),
            textAlign = TextAlign.Center,
        )
        if (isUnlocking) {
            CircularProgressIndicator(color = LexturesColors.Primary, strokeWidth = 2.dp)
        } else {
            AuthPrimaryButton(
                text = L.text(context, localePreferences, R.string.mobile_biometric_unlock),
                onClick = { attemptUnlock() },
                enabled = activity != null,
                modifier = Modifier.fillMaxWidth(),
            )
        }
    }
}
