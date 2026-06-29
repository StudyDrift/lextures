package com.lextures.android.app

import androidx.compose.animation.AnimatedContent
import androidx.compose.animation.core.tween
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.togetherWith
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import com.lextures.android.core.auth.AuthPhase
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.features.auth.AuthFlowScreen
import com.lextures.android.features.home.HomeScreen
import com.lextures.android.features.splash.SplashScreen

@Composable
fun RootScreen(session: AuthSession, modifier: Modifier = Modifier) {
    val phase by session.phase.collectAsState()
    val accessToken by session.accessToken.collectAsState()
    val context = LocalContext.current
    val offline = OfflineService.get(context)

    LaunchedEffect(accessToken, phase) {
        if (phase == AuthPhase.Authenticated) {
            offline.configure(accessToken)
            offline.syncNow(accessToken)
        }
    }

    val isOnline by offline.networkMonitor.isOnline.collectAsState()
    LaunchedEffect(isOnline, accessToken, phase) {
        if (phase == AuthPhase.Authenticated && isOnline) {
            offline.syncNow(accessToken)
        }
    }

    AnimatedContent(
        targetState = phase,
        modifier = modifier.fillMaxSize(),
        transitionSpec = {
            fadeIn(animationSpec = tween(350)) togetherWith fadeOut(animationSpec = tween(350))
        },
        label = "rootPhase",
    ) { current ->
        when (current) {
            AuthPhase.Splash -> SplashScreen()
            AuthPhase.Unauthenticated -> AuthFlowScreen(session = session)
            AuthPhase.Authenticated -> HomeScreen(session = session)
        }
    }
}
