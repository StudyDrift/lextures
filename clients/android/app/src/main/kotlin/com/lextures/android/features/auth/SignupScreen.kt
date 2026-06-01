package com.lextures.android.features.auth

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Switch
import androidx.compose.material3.SwitchDefaults
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
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardCapitalization
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.core.auth.AuthApi
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.auth.PasswordPolicy
import com.lextures.android.core.design.AuthCard
import com.lextures.android.core.design.AuthFooterLink
import com.lextures.android.core.design.AuthPrimaryButton
import com.lextures.android.core.design.AuthScreenContainer
import com.lextures.android.core.design.AuthTextField
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.fieldBorder
import com.lextures.android.core.design.isDarkTheme
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import java.util.TimeZone

@Composable
fun SignupScreen(
    session: AuthSession,
    onSignIn: () -> Unit,
    modifier: Modifier = Modifier,
) {
    var displayName by remember { mutableStateOf("") }
    var email by remember { mutableStateOf("") }
    var password by remember { mutableStateOf("") }
    var registerAsParent by remember { mutableStateOf(false) }
    var policy by remember { mutableStateOf(PasswordPolicy.fallback) }
    var isLoading by remember { mutableStateOf(false) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()
    val timezone = remember { TimeZone.getDefault().id }

    LaunchedEffect(Unit) {
        policy = withContext(Dispatchers.IO) { AuthApi.fetchPasswordPolicy() }
    }

    AuthScreenContainer(modifier = modifier) {
        AuthHeader(
            title = "Create your account",
            subtitle = "One account for courses, assignments, and messages. If your school uses SSO, you can sign in that way later.",
        )

        AuthCard {
            Column(verticalArrangement = Arrangement.spacedBy(20.dp)) {
                AuthTextField(
                    title = "Display name (optional)",
                    value = displayName,
                    onValueChange = { displayName = it },
                    placeholder = "Alex",
                    capitalization = KeyboardCapitalization.Words,
                )
                AuthTextField(
                    title = "Email",
                    value = email,
                    onValueChange = { email = it },
                    placeholder = "you@school.edu",
                    keyboardType = KeyboardType.Email,
                )

                Column {
                    AuthTextField(
                        title = "Password",
                        value = password,
                        onValueChange = { password = it },
                        placeholder = "At least ${policy.minLength} characters",
                        isSecure = true,
                    )
                    Spacer(modifier = Modifier.height(6.dp))
                    PasswordRequirements(policy)
                    Spacer(modifier = Modifier.height(6.dp))
                    PasswordStrengthBar(password)
                }

                ParentToggle(
                    checked = registerAsParent,
                    onCheckedChange = { registerAsParent = it },
                )

                errorMessage?.let { message ->
                    Text(text = message, color = LexturesColors.Error, fontSize = 15.sp)
                }

                AuthPrimaryButton(
                    text = if (isLoading) "Creating account…" else "Create account",
                    enabled = !isLoading && email.isNotBlank() && password.length >= policy.minLength,
                    onClick = {
                        scope.launch {
                            isLoading = true
                            errorMessage = null
                            try {
                                val response = withContext(Dispatchers.IO) {
                                    AuthApi.signup(
                                        email = email.trim(),
                                        password = password,
                                        displayName = displayName,
                                        registerAsParent = registerAsParent,
                                        timezone = timezone,
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

                AuthFooterLink(
                    prompt = "Already have an account?",
                    actionLabel = "Sign in",
                    onAction = onSignIn,
                )
            }
        }
    }
}

@Composable
private fun PasswordRequirements(policy: PasswordPolicy) {
    Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
        Text("At least ${policy.minLength} characters", fontSize = 12.sp, color = textSecondary())
        if (policy.requireUpper) {
            Text("One uppercase letter", fontSize = 12.sp, color = textSecondary())
        }
        if (policy.requireLower) {
            Text("One lowercase letter", fontSize = 12.sp, color = textSecondary())
        }
        if (policy.requireDigit) {
            Text("One digit", fontSize = 12.sp, color = textSecondary())
        }
        if (policy.requireSpecial) {
            Text("One symbol or punctuation character", fontSize = 12.sp, color = textSecondary())
        }
        if (policy.checkHibp) {
            Text(
                "Must not appear in known public breach lists (checked securely)",
                fontSize = 12.sp,
                color = textSecondary(),
            )
        }
    }
}

@Composable
private fun PasswordStrengthBar(password: String) {
    val strength = remember(password) { PasswordStrength.evaluate(password) }
    Row(
        modifier = Modifier.fillMaxWidth(),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        Text("Strength:", fontSize = 12.sp, fontWeight = FontWeight.Medium, color = textSecondary())
        Text(
            strength.label,
            fontSize = 12.sp,
            fontWeight = FontWeight.SemiBold,
            color = textPrimary(),
        )
        Box(
            modifier = Modifier
                .weight(1f)
                .height(6.dp)
                .clip(CircleShape)
                .background(Color.Gray.copy(alpha = 0.25f)),
        ) {
            Box(
                modifier = Modifier
                    .fillMaxWidth(strength.fraction)
                    .height(6.dp)
                    .clip(CircleShape)
                    .background(strength.color),
            )
        }
    }
}

@Composable
private fun ParentToggle(
    checked: Boolean,
    onCheckedChange: (Boolean) -> Unit,
) {
    val shape = RoundedCornerShape(8.dp)
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clip(shape)
            .background(
                if (isDarkTheme()) Color.White.copy(alpha = 0.04f) else Color.Black.copy(alpha = 0.02f),
            )
            .border(1.dp, fieldBorder(), shape)
            .padding(12.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        Switch(
            checked = checked,
            onCheckedChange = onCheckedChange,
            colors = SwitchDefaults.colors(checkedTrackColor = LexturesColors.Primary),
        )
        Text(
            text = "I am registering as a parent or guardian for read-only access when my school links my account to a student.",
            fontSize = 15.sp,
            color = textPrimary(),
            lineHeight = 20.sp,
        )
    }
}

private enum class PasswordStrength(
    val label: String,
    val fraction: Float,
    val color: Color,
) {
    Weak("Weak", 0.33f, LexturesColors.Error),
    Fair("Fair", 0.66f, Color(0xFFFF9800)),
    Strong("Strong", 1f, LexturesColors.StrengthStrong),
    ;

    companion object {
        fun evaluate(password: String): PasswordStrength {
            if (password.length < 8) return Weak
            var score = 0
            if (password.any { it.isUpperCase() }) score++
            if (password.any { it.isLowerCase() }) score++
            if (password.any { it.isDigit() }) score++
            if (password.any { !it.isLetterOrDigit() }) score++
            if (password.length >= 12) score++
            return when (score) {
                in 0..2 -> Weak
                in 3..4 -> Fair
                else -> Strong
            }
        }
    }
}
