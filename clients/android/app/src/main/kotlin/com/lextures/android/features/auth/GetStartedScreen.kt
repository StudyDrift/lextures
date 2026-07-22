package com.lextures.android.features.auth

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Home
import androidx.compose.material.icons.filled.School
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardCapitalization
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.config.AppConfiguration
import com.lextures.android.core.config.EnvironmentStore
import com.lextures.android.core.config.SchoolCodeLogic
import com.lextures.android.core.design.AuthCard
import com.lextures.android.core.design.AuthPrimaryButton
import com.lextures.android.core.design.AuthScreenContainer
import com.lextures.android.core.design.AuthTextField
import com.lextures.android.core.design.BrandLogo
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.cardBackground
import com.lextures.android.core.design.fieldBorder
import com.lextures.android.core.design.isDarkTheme
import com.lextures.android.core.design.sceneBackground
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L

private enum class GetStartedStep {
    Choose,
    SchoolCode,
}

@Composable
fun GetStartedScreen(
    onComplete: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val store = remember { EnvironmentStore.get(context) }
    var step by remember { mutableStateOf(GetStartedStep.Choose) }
    var schoolCode by remember { mutableStateOf("") }

    AuthScreenContainer(modifier = modifier) {
        when (step) {
            GetStartedStep.Choose -> ChooseStep(
                onHomeschool = {
                    store.selectHomeschool()
                    AppConfiguration.bindEnvironment(store)
                    onComplete()
                },
                onSchool = { step = GetStartedStep.SchoolCode },
            )
            GetStartedStep.SchoolCode -> SchoolCodeStep(
                schoolCode = schoolCode,
                onSchoolCodeChange = { schoolCode = it },
                onBack = {
                    step = GetStartedStep.Choose
                    schoolCode = ""
                },
                onContinue = {
                    if (!SchoolCodeLogic.isValid(schoolCode)) return@SchoolCodeStep
                    store.selectSchool(schoolCode)
                    AppConfiguration.bindEnvironment(store)
                    onComplete()
                },
            )
        }
    }
}

@Composable
private fun ChooseStep(
    onHomeschool: () -> Unit,
    onSchool: () -> Unit,
) {
    BrandLogo(maxHeight = 56)
    Spacer(modifier = Modifier.height(28.dp))

    Text(
        text = L.text(R.string.auth_getStarted_title),
        fontFamily = FontFamily.Serif,
        fontWeight = FontWeight.SemiBold,
        fontSize = 28.sp,
        color = textPrimary(),
        textAlign = TextAlign.Center,
        modifier = Modifier.fillMaxWidth(),
    )
    Spacer(modifier = Modifier.height(8.dp))
    Text(
        text = L.text(R.string.auth_getStarted_subtitle),
        fontSize = 15.sp,
        color = textSecondary(),
        textAlign = TextAlign.Center,
        modifier = Modifier.fillMaxWidth(),
    )
    Spacer(modifier = Modifier.height(28.dp))

    Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
        PathCard(
            icon = Icons.Default.Home,
            title = L.text(R.string.auth_getStarted_homeschoolTitle),
            description = L.text(R.string.auth_getStarted_homeschoolDescription),
            onClick = onHomeschool,
        )
        PathCard(
            icon = Icons.Default.School,
            title = L.text(R.string.auth_getStarted_schoolTitle),
            description = L.text(R.string.auth_getStarted_schoolDescription),
            onClick = onSchool,
        )
    }
}

@Composable
private fun PathCard(
    icon: ImageVector,
    title: String,
    description: String,
    onClick: () -> Unit,
) {
    val shape = RoundedCornerShape(18.dp)
    val accent = if (isDarkTheme()) LexturesColors.BrandTeal else LexturesColors.Primary
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clip(shape)
            .background(cardBackground())
            .border(1.dp, fieldBorder().copy(alpha = 0.7f), shape)
            .clickable(onClick = onClick)
            .padding(18.dp),
        verticalAlignment = Alignment.Top,
    ) {
        Box(
            modifier = Modifier
                .size(44.dp)
                .clip(RoundedCornerShape(12.dp))
                .background(accent.copy(alpha = if (isDarkTheme()) 0.18f else 0.12f)),
            contentAlignment = Alignment.Center,
        ) {
            Icon(
                imageVector = icon,
                contentDescription = null,
                tint = accent,
                modifier = Modifier.size(22.dp),
            )
        }
        Spacer(modifier = Modifier.width(14.dp))
        Column(modifier = Modifier.weight(1f)) {
            Text(
                text = title,
                fontWeight = FontWeight.SemiBold,
                fontSize = 15.sp,
                color = textPrimary(),
            )
            Spacer(modifier = Modifier.height(4.dp))
            Text(
                text = description,
                fontSize = 13.sp,
                color = textSecondary(),
                lineHeight = 18.sp,
            )
        }
    }
}

@Composable
private fun SchoolCodeStep(
    schoolCode: String,
    onSchoolCodeChange: (String) -> Unit,
    onBack: () -> Unit,
    onContinue: () -> Unit,
) {
    val errorKey = if (schoolCode.isEmpty()) null else SchoolCodeLogic.errorKey(schoolCode)
    val errorText = errorKey?.let { key ->
        val resId = when (key) {
            "auth_getStarted_schoolCodeErrorEmpty" -> R.string.auth_getStarted_schoolCodeErrorEmpty
            "auth_getStarted_schoolCodeErrorLengthMin" -> R.string.auth_getStarted_schoolCodeErrorLengthMin
            "auth_getStarted_schoolCodeErrorLengthMax" -> R.string.auth_getStarted_schoolCodeErrorLengthMax
            "auth_getStarted_schoolCodeErrorFormat" -> R.string.auth_getStarted_schoolCodeErrorFormat
            "auth_getStarted_schoolCodeErrorReserved" -> R.string.auth_getStarted_schoolCodeErrorReserved
            else -> null
        }
        resId?.let { L.text(it) }
    }
    val preview = SchoolCodeLogic.previewHost(schoolCode)

    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onBack)
            .padding(bottom = 24.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(
            imageVector = Icons.AutoMirrored.Filled.ArrowBack,
            contentDescription = null,
            tint = LexturesColors.PrimaryMuted,
            modifier = Modifier.size(18.dp),
        )
        Spacer(modifier = Modifier.width(6.dp))
        Text(
            text = L.text(R.string.auth_getStarted_back),
            fontSize = 15.sp,
            fontWeight = FontWeight.Medium,
            color = LexturesColors.PrimaryMuted,
        )
    }

    Text(
        text = L.text(R.string.auth_getStarted_schoolCodeTitle),
        fontFamily = FontFamily.Serif,
        fontWeight = FontWeight.SemiBold,
        fontSize = 28.sp,
        color = textPrimary(),
        modifier = Modifier.fillMaxWidth(),
    )
    Spacer(modifier = Modifier.height(8.dp))
    Text(
        text = L.text(R.string.auth_getStarted_schoolCodeSubtitle),
        fontSize = 15.sp,
        color = textSecondary(),
        modifier = Modifier.fillMaxWidth(),
    )
    Spacer(modifier = Modifier.height(24.dp))

    AuthCard {
        Column(verticalArrangement = Arrangement.spacedBy(16.dp)) {
            AuthTextField(
                title = L.text(R.string.auth_getStarted_schoolCodeLabel),
                value = schoolCode,
                onValueChange = onSchoolCodeChange,
                placeholder = L.text(R.string.auth_getStarted_schoolCodePlaceholder),
                capitalization = KeyboardCapitalization.None,
            )
            Text(
                text = L.text(R.string.auth_getStarted_schoolCodeHelp),
                fontSize = 13.sp,
                color = textSecondary(),
            )
            if (errorText != null) {
                Text(
                    text = errorText,
                    fontSize = 15.sp,
                    color = LexturesColors.Error,
                    modifier = Modifier.fillMaxWidth(),
                )
            }
            Text(
                text = L.format(R.string.auth_getStarted_schoolCodePreview, preview),
                fontSize = 15.sp,
                color = textSecondary(),
                modifier = Modifier
                    .fillMaxWidth()
                    .clip(RoundedCornerShape(12.dp))
                    .background(sceneBackground().copy(alpha = 0.7f))
                    .padding(12.dp),
            )
            AuthPrimaryButton(
                text = L.text(R.string.auth_getStarted_continue),
                onClick = onContinue,
                enabled = SchoolCodeLogic.isValid(schoolCode),
            )
        }
    }
}
