package com.lextures.android.app

import android.content.Intent
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.activity.viewModels
import androidx.core.splashscreen.SplashScreen.Companion.installSplashScreen
import androidx.lifecycle.viewmodel.compose.viewModel
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.LexturesTheme
import com.lextures.android.core.push.PushManager

class MainActivity : ComponentActivity() {
    private val session: AuthSession by viewModels()

    override fun onCreate(savedInstanceState: Bundle?) {
        installSplashScreen()
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()

        PushManager.getInstance(this).configure { session.accessToken.value }
        handleDeepLinkIntent(intent)

        setContent {
            val session: AuthSession = viewModel()
            LexturesTheme {
                RootScreen(session = session)
            }
        }
    }

    override fun onNewIntent(intent: Intent) {
        super.onNewIntent(intent)
        handleDeepLinkIntent(intent)
    }

    override fun onResume() {
        super.onResume()
        session.onAppResumed()
        PushManager.getInstance(this).requestTokenSync()
    }

    private fun handleDeepLinkIntent(intent: Intent?) {
        val actionUrl = intent?.data?.toString()
            ?: intent?.getStringExtra(PushManager.EXTRA_ACTION_URL)
        if (!actionUrl.isNullOrBlank()) {
            PushManager.getInstance(this).onDeepLinkFromPayload(actionUrl)
        }
    }
}
