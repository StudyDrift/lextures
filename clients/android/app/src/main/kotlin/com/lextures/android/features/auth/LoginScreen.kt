package com.lextures.android.features.auth

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.auth.AuthApi
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.AuthCard
import com.lextures.android.core.design.AuthFooterLink
import com.lextures.android.core.design.AuthPrimaryButton
import com.lextures.android.core.design.AuthScreenContainer
import com.lextures.android.core.design.AuthTextField
import com.lextures.android.core.design.BrandLogo
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

@Composable
fun LoginScreen(
    session: AuthSession,
    onCreateAccount: () -> Unit,
    modifier: Modifier = Modifier,
) {
    var email by remember { mutableStateOf("") }
    var password by remember { mutableStateOf("") }
    var isLoading by remember { mutableStateOf(false) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    AuthScreenContainer(modifier = modifier) {
        AuthHeader(
            title = L.text(R.string.auth_login_title),
            subtitle = L.text(R.string.auth_login_subtitle),
        )

        AuthCard {
            Column(verticalArrangement = Arrangement.spacedBy(20.dp)) {
                AuthTextField(
                    title = L.text(R.string.auth_login_email),
                    value = email,
                    onValueChange = { email = it },
                    placeholder = L.text(R.string.auth_login_emailPlaceholder),
                    keyboardType = KeyboardType.Email,
                )
                AuthTextField(
                    title = L.text(R.string.auth_login_password),
                    value = password,
                    onValueChange = { password = it },
                    placeholder = "••••••••",
                    isSecure = true,
                )

                errorMessage?.let { message ->
                    Text(
                        text = message,
                        color = LexturesColors.Error,
                        fontSize = 15.sp,
                        modifier = Modifier.fillMaxWidth(),
                    )
                }

                AuthPrimaryButton(
                    text = if (isLoading) L.text(R.string.auth_login_submitting) else L.text(R.string.auth_login_submit),
                    enabled = !isLoading && email.isNotBlank() && password.isNotBlank(),
                    onClick = {
                        scope.launch {
                            isLoading = true
                            errorMessage = null
                            try {
                                val response = withContext(Dispatchers.IO) {
                                    AuthApi.login(
                                        email = email.trim(),
                                        password = password,
                                    )
                                }
                                session.applyTokenResponse(response)
                            } catch (e: Exception) {
                                errorMessage = session.mapError(e)
                            } finally {
                                isLoading = false
                            }
                        }
                    },
                )

                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.End,
                ) {
                    Text(
                        text = L.text(R.string.auth_login_forgotPassword),
                        color = LexturesColors.PrimaryMuted.copy(alpha = 0.7f),
                        fontSize = 15.sp,
                    )
                }

                AuthFooterLink(
                    prompt = L.text(R.string.auth_login_newHere),
                    actionLabel = L.text(R.string.auth_login_createAccount),
                    onAction = onCreateAccount,
                )
            }
        }
    }
}

@Composable
internal fun AuthHeader(
    title: String,
    subtitle: String,
    modifier: Modifier = Modifier,
) {
    Column(
        modifier = modifier
            .fillMaxWidth()
            .padding(bottom = 24.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        BrandLogo(maxHeight = 56, maxWidth = 180)
        Text(
            text = title,
            fontSize = 28.sp,
            fontFamily = FontFamily.Serif,
            fontWeight = FontWeight.SemiBold,
            color = textPrimary(),
            textAlign = TextAlign.Center,
            modifier = Modifier.padding(top = 20.dp),
        )
        Text(
            text = subtitle,
            fontSize = 15.sp,
            color = textSecondary(),
            textAlign = TextAlign.Center,
            lineHeight = 22.sp,
            modifier = Modifier.padding(top = 8.dp, start = 4.dp, end = 4.dp),
        )
    }
}
