package com.lextures.android.features.auth

import androidx.compose.animation.AnimatedContent
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.slideInHorizontally
import androidx.compose.animation.slideOutHorizontally
import androidx.compose.animation.togetherWith
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.config.AppConfiguration
import com.lextures.android.core.config.EnvironmentStore
import com.lextures.android.core.design.PublicAuthBackground

private enum class AuthScreen {
    GetStarted,
    Login,
    Signup,
    Mfa,
}

@Composable
fun AuthFlowScreen(
    session: AuthSession,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val environmentStore = remember { EnvironmentStore.get(context) }
    val mfaRequired by session.mfaRequired.collectAsState()
    var screen by remember {
        mutableStateOf(
            if (environmentStore.hasSelection) AuthScreen.Login else AuthScreen.GetStarted,
        )
    }
    var signOutBanner by remember { mutableStateOf<String?>(null) }

    LaunchedEffect(Unit) {
        AppConfiguration.bindEnvironment(environmentStore)
        signOutBanner = session.consumeSignOutMessage()
    }

    if (mfaRequired != null) {
        screen = AuthScreen.Mfa
    }

    PublicAuthBackground(modifier = modifier.fillMaxSize()) {
        AnimatedContent(
            targetState = screen,
            modifier = Modifier.fillMaxSize(),
            transitionSpec = {
                val forward = when {
                    targetState == AuthScreen.Login && initialState == AuthScreen.GetStarted -> true
                    targetState == AuthScreen.Signup || targetState == AuthScreen.Mfa -> true
                    targetState == AuthScreen.GetStarted -> false
                    else -> false
                }
                val enter = slideInHorizontally { if (forward) it else -it } + fadeIn()
                val exit = slideOutHorizontally { if (forward) -it else it } + fadeOut()
                enter togetherWith exit
            },
            label = "authFlow",
        ) { current ->
            when (current) {
                AuthScreen.GetStarted -> GetStartedScreen(
                    onComplete = { screen = AuthScreen.Login },
                )
                AuthScreen.Login -> LoginScreen(
                    session = session,
                    bannerMessage = signOutBanner,
                    onCreateAccount = { screen = AuthScreen.Signup },
                    onMfaRequired = { screen = AuthScreen.Mfa },
                    onChangeEnvironment = {
                        environmentStore.clearSelection()
                        AppConfiguration.bindEnvironment(environmentStore)
                        screen = AuthScreen.GetStarted
                    },
                )
                AuthScreen.Signup -> SignupScreen(
                    session = session,
                    onSignIn = { screen = AuthScreen.Login },
                    onMfaRequired = { screen = AuthScreen.Mfa },
                )
                AuthScreen.Mfa -> MfaChallengeScreen(
                    session = session,
                    onCancel = { screen = AuthScreen.Login },
                )
            }
        }
    }
}
