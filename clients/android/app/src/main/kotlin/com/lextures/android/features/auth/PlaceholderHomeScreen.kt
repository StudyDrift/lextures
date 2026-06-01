package com.lextures.android.features.auth

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.AuthPrimaryButton
import com.lextures.android.core.design.BrandLogo
import com.lextures.android.core.design.PublicAuthBackground
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun PlaceholderHomeScreen(
    session: AuthSession,
    modifier: Modifier = Modifier,
) {
    val userEmail by session.userEmail.collectAsState()

    Scaffold(
        modifier = modifier,
        topBar = {
            TopAppBar(title = { Text("Lextures") })
        },
    ) { padding ->
        PublicAuthBackground(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding),
        ) {
            Column(
                modifier = Modifier
                    .fillMaxSize()
                    .padding(24.dp),
                horizontalAlignment = Alignment.CenterHorizontally,
                verticalArrangement = Arrangement.Center,
            ) {
                BrandLogo(maxHeight = 72)
                Text(
                    text = "You're signed in",
                    fontSize = 22.sp,
                    fontWeight = FontWeight.SemiBold,
                    color = textPrimary(),
                    modifier = Modifier.padding(top = 20.dp),
                )
                userEmail?.let { email ->
                    Text(
                        text = email,
                        fontSize = 15.sp,
                        color = textSecondary(),
                        modifier = Modifier.padding(top = 8.dp),
                    )
                }
                Text(
                    text = "Course and dashboard features are coming soon to the Android app.",
                    fontSize = 15.sp,
                    color = textSecondary(),
                    textAlign = TextAlign.Center,
                    modifier = Modifier.padding(top = 16.dp, start = 8.dp, end = 8.dp),
                )
                AuthPrimaryButton(
                    text = "Sign out",
                    onClick = { session.signOut() },
                    modifier = Modifier
                        .padding(top = 24.dp, start = 40.dp, end = 40.dp),
                )
            }
        }
    }
}
