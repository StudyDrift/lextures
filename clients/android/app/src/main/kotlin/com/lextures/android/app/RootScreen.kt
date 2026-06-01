package com.lextures.android.app

import androidx.compose.animation.AnimatedContent
import androidx.compose.animation.core.tween
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.togetherWith
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import com.lextures.android.core.auth.AuthPhase
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.features.auth.AuthFlowScreen
import com.lextures.android.features.auth.PlaceholderHomeScreen
import com.lextures.android.features.splash.SplashScreen

@Composable
fun RootScreen(session: AuthSession, modifier: Modifier = Modifier) {
    val phase by session.phase.collectAsState()

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
            AuthPhase.Authenticated -> PlaceholderHomeScreen(session = session)
        }
    }
}
