package com.lextures.android.app

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.activity.viewModels
import androidx.core.splashscreen.SplashScreen.Companion.installSplashScreen
import androidx.lifecycle.viewmodel.compose.viewModel
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.LexturesTheme

class MainActivity : ComponentActivity() {
    private val session: AuthSession by viewModels()

    override fun onCreate(savedInstanceState: Bundle?) {
        installSplashScreen()
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()

        setContent {
            val session: AuthSession = viewModel()
            LexturesTheme {
                RootScreen(session = session)
            }
        }
    }

    override fun onResume() {
        super.onResume()
        // Access tokens last 15 minutes; refresh when returning from background.
        session.onAppResumed()
    }
}
