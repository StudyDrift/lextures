package com.lextures.android.features.auth

import android.content.Context
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.HorizontalDivider
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
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.auth.AuthApi
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.auth.SsoProvider
import com.lextures.android.core.design.AuthCard
import com.lextures.android.core.design.AuthFooterLink
import com.lextures.android.core.design.AuthOutlineButton
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

private enum class MagicLinkStatus {
    Idle,
    Sending,
    Sent,
    Error,
}

@Composable
fun LoginScreen(
    session: AuthSession,
    onCreateAccount: () -> Unit,
    onMfaRequired: () -> Unit,
    bannerMessage: String? = null,
    modifier: Modifier = Modifier,
) {
    var email by remember { mutableStateOf("") }
    var password by remember { mutableStateOf("") }
    var isLoading by remember { mutableStateOf(false) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var magicLinkStatus by remember { mutableStateOf(MagicLinkStatus.Idle) }
    var samlStatus by remember { mutableStateOf(com.lextures.android.core.auth.SamlStatusResponse()) }
    var oidcStatus by remember { mutableStateOf(com.lextures.android.core.auth.OidcStatusResponse()) }
    val scope = rememberCoroutineScope()
    val context = LocalContext.current

    LaunchedEffect(Unit) {
        samlStatus = withContext(Dispatchers.IO) { AuthApi.fetchSamlStatus() }
        oidcStatus = withContext(Dispatchers.IO) { AuthApi.fetchOidcStatus() }
    }

    val forceSaml = samlStatus.idp?.forceSaml == true
    val ssoProviders = remember(samlStatus, oidcStatus) { buildSsoProviders(samlStatus, oidcStatus) }

    AuthScreenContainer(modifier = modifier) {
        AuthHeader(
            title = L.text(R.string.auth_login_title),
            subtitle = L.text(R.string.auth_login_subtitle),
        )

        AuthCard {
            Column(verticalArrangement = Arrangement.spacedBy(20.dp)) {
                bannerMessage?.let { message ->
                    Text(
                        text = message,
                        color = LexturesColors.Error,
                        fontSize = 15.sp,
                        modifier = Modifier.fillMaxWidth(),
                    )
                }

                if (ssoProviders.isNotEmpty()) {
                    Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                        ssoProviders.forEach { provider ->
                            AuthOutlineButton(
                                text = ssoLabel(context, provider, samlStatus),
                                enabled = !isLoading,
                                onClick = { SsoAuth.start(context, provider) },
                            )
                        }
                    }
                }

                if (forceSaml) {
                    Text(
                        text = L.text(R.string.auth_login_ssoRequired),
                        color = textSecondary(),
                        fontSize = 15.sp,
                    )
                }

                if (!forceSaml) {
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
                        text = if (isLoading) {
                            L.text(R.string.auth_login_submitting)
                        } else {
                            L.text(R.string.auth_login_submit)
                        },
                        enabled = !isLoading && email.isNotBlank() && password.isNotBlank(),
                        onClick = {
                            scope.launch {
                                isLoading = true
                                errorMessage = null
                                try {
                                    val response = withContext(Dispatchers.IO) {
                                        AuthApi.login(email.trim(), password)
                                    }
                                    session.applyTokenResponse(response)
                                } catch (e: AuthSession.AuthSessionError.MfaRequired) {
                                    onMfaRequired()
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

                    HorizontalDivider(modifier = Modifier.padding(vertical = 4.dp))

                    Text(
                        text = L.text(R.string.auth_login_magicLinkTitle),
                        fontWeight = FontWeight.SemiBold,
                        color = textPrimary(),
                        fontSize = 15.sp,
                    )
                    Text(
                        text = L.text(R.string.auth_login_magicLinkHint),
                        color = textSecondary(),
                        fontSize = 13.sp,
                    )
                    when (magicLinkStatus) {
                        MagicLinkStatus.Sent -> Text(
                            text = L.text(R.string.auth_login_magicLinkSent),
                            color = textSecondary(),
                            fontSize = 15.sp,
                        )
                        else -> AuthOutlineButton(
                            text = when (magicLinkStatus) {
                                MagicLinkStatus.Sending -> L.text(R.string.auth_login_magicLinkSending)
                                else -> L.text(R.string.auth_login_sendMagicLink)
                            },
                            enabled = magicLinkStatus != MagicLinkStatus.Sending && email.isNotBlank(),
                            onClick = {
                                scope.launch {
                                    magicLinkStatus = MagicLinkStatus.Sending
                                    errorMessage = null
                                    try {
                                        withContext(Dispatchers.IO) {
                                            AuthApi.requestMagicLink(email.trim())
                                        }
                                        magicLinkStatus = MagicLinkStatus.Sent
                                    } catch (e: Exception) {
                                        magicLinkStatus = MagicLinkStatus.Error
                                        errorMessage = session.mapError(e)
                                    }
                                }
                            },
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
}

private fun buildSsoProviders(
    samlStatus: com.lextures.android.core.auth.SamlStatusResponse,
    oidcStatus: com.lextures.android.core.auth.OidcStatusResponse,
): List<SsoProvider> {
    val items = mutableListOf<SsoProvider>()
    if (samlStatus.enabled) {
        samlStatus.idp?.let { items += SsoProvider.Saml(it.id) }
    }
    if (oidcStatus.showsClever) items += SsoProvider.Oidc("/auth/clever/login", "Clever")
    if (oidcStatus.showsClassLink) items += SsoProvider.Oidc("/auth/oidc/classlink/login", "ClassLink")
    if (oidcStatus.enabled) {
        if (oidcStatus.google == true) items += SsoProvider.Oidc("/auth/oidc/google/login", "Google")
        if (oidcStatus.microsoft == true) items += SsoProvider.Oidc("/auth/oidc/microsoft/login", "Microsoft")
        if (oidcStatus.apple == true) items += SsoProvider.Oidc("/auth/oidc/apple/login", "Apple")
        oidcStatus.custom.orEmpty().forEach { provider ->
            items += SsoProvider.Oidc(
                "/auth/oidc/custom/login?configId=${provider.id}",
                provider.displayName,
            )
        }
    }
    return items
}

private fun ssoLabel(
    context: Context,
    provider: SsoProvider,
    samlStatus: com.lextures.android.core.auth.SamlStatusResponse,
): String {
    val label = when (provider) {
        is SsoProvider.Saml -> samlStatus.idp?.label ?: "SSO"
        is SsoProvider.Oidc -> provider.label
    }
    return context.getString(R.string.auth_login_ssoButton, label)
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
