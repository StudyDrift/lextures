package com.lextures.android.features.auth

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.input.KeyboardCapitalization
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.auth.AuthApi
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.auth.MfaRequired
import com.lextures.android.core.design.AuthCard
import com.lextures.android.core.design.AuthOutlineButton
import com.lextures.android.core.design.AuthPrimaryButton
import com.lextures.android.core.design.AuthScreenContainer
import com.lextures.android.core.design.AuthTextField
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.i18n.L
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

@Composable
fun MfaChallengeScreen(
    session: AuthSession,
    onCancel: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val mfaRequired by session.mfaRequired.collectAsState()
    val mfaPendingToken by session.mfaPendingToken.collectAsState()
    val mode = mfaRequired ?: MfaRequired.Challenge

    var code by remember { mutableStateOf("") }
    var backup by remember { mutableStateOf("") }
    var showBackup by remember { mutableStateOf(false) }
    var isLoading by remember { mutableStateOf(false) }
    var passkeyBusy by remember { mutableStateOf(false) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var totpCredentialId by remember { mutableStateOf<String?>(null) }
    var totpSetupStarted by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()
    val context = LocalContext.current
    val passkeyUnsupportedMessage = context.getString(R.string.auth_mfa_passkeyUnsupported)

    AuthScreenContainer(modifier = modifier) {
        AuthHeader(
            title = if (mode == MfaRequired.Setup) {
                L.text(R.string.auth_mfa_setupTitle)
            } else {
                L.text(R.string.auth_mfa_title)
            },
            subtitle = if (mode == MfaRequired.Setup) {
                L.text(R.string.auth_mfa_setupSubtitle)
            } else {
                L.text(R.string.auth_mfa_subtitle)
            },
        )

        AuthCard {
            Column(verticalArrangement = Arrangement.spacedBy(16.dp)) {
                if (mode == MfaRequired.Setup && !totpSetupStarted) {
                    AuthOutlineButton(
                        text = L.text(R.string.auth_mfa_setupTotp),
                        enabled = !isLoading,
                        onClick = {
                            scope.launch {
                                val token = mfaPendingToken ?: return@launch
                                isLoading = true
                                errorMessage = null
                                try {
                                    val response = withContext(Dispatchers.IO) {
                                        AuthApi.mfaTotpEnrol(token)
                                    }
                                    totpCredentialId = response.credentialId
                                    totpSetupStarted = true
                                } catch (e: Exception) {
                                    errorMessage = session.mapError(e)
                                } finally {
                                    isLoading = false
                                }
                            }
                        },
                    )
                    AuthPrimaryButton(
                        text = if (passkeyBusy) {
                            L.text(R.string.auth_mfa_passkeyWaiting)
                        } else {
                            L.text(R.string.auth_mfa_setupPasskey)
                        },
                        enabled = !passkeyBusy && !isLoading,
                        onClick = {
                            passkeyBusy = true
                            errorMessage = passkeyUnsupportedMessage
                            passkeyBusy = false
                        },
                    )
                }

                if (totpSetupStarted || mode == MfaRequired.Challenge) {
                    if (mode == MfaRequired.Setup && totpSetupStarted) {
                        Text(
                            text = L.text(R.string.auth_mfa_scanInstructions),
                            color = LexturesColors.TextSecondary,
                            fontSize = 15.sp,
                        )
                    }

                    AuthTextField(
                        title = L.text(R.string.auth_mfa_code),
                        value = code,
                        onValueChange = { code = it.filter(Char::isDigit).take(6) },
                        placeholder = L.text(R.string.auth_mfa_codePlaceholder),
                        keyboardType = KeyboardType.Number,
                    )

                    errorMessage?.let { message ->
                        Text(text = message, color = LexturesColors.Error, fontSize = 15.sp)
                    }

                    AuthPrimaryButton(
                        text = if (isLoading) {
                            L.text(R.string.auth_mfa_verifying)
                        } else if (mode == MfaRequired.Setup) {
                            L.text(R.string.auth_mfa_confirmEnrolment)
                        } else {
                            L.text(R.string.auth_mfa_verify)
                        },
                        enabled = !isLoading && code.length == 6,
                        onClick = {
                            scope.launch {
                                val token = mfaPendingToken ?: return@launch
                                isLoading = true
                                errorMessage = null
                                try {
                                    if (mode == MfaRequired.Setup) {
                                        val credentialId = totpCredentialId ?: return@launch
                                        withContext(Dispatchers.IO) {
                                            AuthApi.mfaTotpVerifyEnrol(credentialId, code, token)
                                        }
                                        val response = withContext(Dispatchers.IO) {
                                            AuthApi.mfaSetupComplete(token)
                                        }
                                        session.applyTokenResponse(response)
                                    } else {
                                        val response = withContext(Dispatchers.IO) {
                                            AuthApi.mfaTotpChallenge(code, token)
                                        }
                                        session.applyTokenResponse(response)
                                    }
                                } catch (e: Exception) {
                                    errorMessage = session.mapError(e)
                                } finally {
                                    isLoading = false
                                }
                            }
                        },
                    )
                }

                if (mode == MfaRequired.Challenge) {
                    HorizontalDivider(modifier = Modifier.padding(vertical = 4.dp))
                    Text(
                        text = L.text(R.string.auth_login_orDivider).uppercase(),
                        color = LexturesColors.TextSecondary,
                        fontSize = 12.sp,
                        modifier = Modifier.fillMaxWidth(),
                    )
                    AuthOutlineButton(
                        text = if (passkeyBusy) {
                            L.text(R.string.auth_mfa_passkeyWaiting)
                        } else {
                            L.text(R.string.auth_mfa_usePasskey)
                        },
                        enabled = !passkeyBusy && !isLoading,
                        onClick = {
                            errorMessage = passkeyUnsupportedMessage
                        },
                    )
                    if (!showBackup) {
                        AuthOutlineButton(
                            text = L.text(R.string.auth_mfa_useBackup),
                            enabled = !isLoading,
                            onClick = { showBackup = true },
                        )
                    } else {
                        AuthTextField(
                            title = L.text(R.string.auth_mfa_backupCode),
                            value = backup,
                            onValueChange = { backup = it.uppercase() },
                            capitalization = KeyboardCapitalization.Characters,
                        )
                        AuthPrimaryButton(
                            text = L.text(R.string.auth_mfa_verify),
                            enabled = !isLoading && backup.length >= 8,
                            onClick = {
                                scope.launch {
                                    val token = mfaPendingToken ?: return@launch
                                    isLoading = true
                                    errorMessage = null
                                    try {
                                        val response = withContext(Dispatchers.IO) {
                                            AuthApi.mfaBackupChallenge(backup, token)
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
                    }
                }

                AuthOutlineButton(
                    text = L.text(R.string.auth_mfa_cancel),
                    enabled = !isLoading,
                    onClick = {
                        session.clearMfaFlow()
                        onCancel()
                    },
                )
            }
        }
    }
}
