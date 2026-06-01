package com.lextures.android.core.design

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.defaultMinSize
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.WindowInsetsSides
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.only
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.safeDrawing
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.layout.windowInsetsPadding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.focus.onFocusChanged
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardCapitalization
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.input.VisualTransformation
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import coil.compose.AsyncImage
import coil.decode.SvgDecoder
import coil.request.ImageRequest

@Composable
fun PublicAuthBackground(modifier: Modifier = Modifier, content: @Composable () -> Unit) {
    val dark = isDarkTheme()
    val tealGlow = LexturesColors.Primary.copy(alpha = if (dark) 0.07f else 0.055f)
    val grayGlow = Color.Gray.copy(alpha = if (dark) 0.04f else 0.05f)

    Box(
        modifier = modifier
            .fillMaxSize()
            .background(sceneBackground()),
    ) {
        Box(
            modifier = Modifier
                .fillMaxSize()
                .background(
                    Brush.radialGradient(
                        colors = listOf(tealGlow, Color.Transparent),
                        radius = 900f,
                    ),
                ),
        )
        Box(
            modifier = Modifier
                .fillMaxSize()
                .background(
                    Brush.radialGradient(
                        colors = listOf(grayGlow, Color.Transparent),
                        radius = 800f,
                    ),
                ),
        )
        content()
    }
}

/** Scrollable auth column centered on wide screens (parity with web/iOS auth shell). */
@Composable
fun AuthScreenContainer(
    modifier: Modifier = Modifier,
    content: @Composable () -> Unit,
) {
    Box(
        modifier = modifier.fillMaxSize(),
        contentAlignment = Alignment.TopCenter,
    ) {
        Column(
            modifier = Modifier
                .widthIn(max = 440.dp)
                .fillMaxWidth()
                .verticalScroll(rememberScrollState())
                .windowInsetsPadding(WindowInsets.safeDrawing.only(WindowInsetsSides.Top))
                .padding(horizontal = 20.dp)
                .padding(top = 16.dp, bottom = 28.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            content()
        }
    }
}

@Composable
fun AuthCard(
    modifier: Modifier = Modifier,
    content: @Composable () -> Unit,
) {
    val shape = RoundedCornerShape(12.dp)
    Column(
        modifier = modifier
            .fillMaxWidth()
            .shadow(
                elevation = if (isDarkTheme()) 0.dp else 2.dp,
                shape = shape,
                clip = false,
            )
            .clip(shape)
            .background(cardBackground())
            .border(1.dp, fieldBorder().copy(alpha = 0.9f), shape)
            .padding(28.dp),
    ) {
        content()
    }
}

@Composable
fun BrandLogo(
    modifier: Modifier = Modifier,
    maxHeight: Int = 56,
    maxWidth: Int = 200,
) {
    val context = LocalContext.current
    AsyncImage(
        model = ImageRequest.Builder(context)
            .data("file:///android_asset/logo-trimmed.svg")
            .decoderFactory(SvgDecoder.Factory())
            .build(),
        contentDescription = "Lextures",
        contentScale = ContentScale.Fit,
        modifier = modifier
            .height(maxHeight.dp)
            .widthIn(max = maxWidth.dp),
    )
}

@Composable
fun AuthTextField(
    title: String,
    value: String,
    onValueChange: (String) -> Unit,
    modifier: Modifier = Modifier,
    placeholder: String = "",
    isSecure: Boolean = false,
    keyboardType: KeyboardType = KeyboardType.Text,
    capitalization: KeyboardCapitalization = KeyboardCapitalization.None,
) {
    var focused by remember { mutableStateOf(false) }
    val shape = RoundedCornerShape(8.dp)
    val borderColor = if (focused) LexturesColors.Primary else fieldBorder()
    val fieldBackground = if (isDarkTheme()) Color(0xFF0D0D0D) else Color.White

    Column(modifier = modifier.fillMaxWidth()) {
        Text(
            text = title,
            fontSize = 15.sp,
            fontWeight = FontWeight.Medium,
            color = textPrimary(),
            modifier = Modifier.padding(bottom = 6.dp),
        )
        BasicTextField(
            value = value,
            onValueChange = onValueChange,
            modifier = Modifier
                .fillMaxWidth()
                .onFocusChanged { focused = it.isFocused }
                .defaultMinSize(minHeight = 44.dp)
                .clip(shape)
                .background(fieldBackground)
                .border(
                    width = if (focused) 2.dp else 1.dp,
                    color = borderColor,
                    shape = shape,
                )
                .padding(horizontal = 12.dp, vertical = 12.dp),
            textStyle = TextStyle(
                fontSize = 16.sp,
                color = textPrimary(),
            ),
            cursorBrush = SolidColor(LexturesColors.Primary),
            singleLine = true,
            visualTransformation = if (isSecure) {
                PasswordVisualTransformation()
            } else {
                VisualTransformation.None
            },
            keyboardOptions = KeyboardOptions(
                keyboardType = keyboardType,
                capitalization = capitalization,
            ),
            decorationBox = { inner ->
                Box(contentAlignment = Alignment.CenterStart) {
                    if (value.isEmpty()) {
                        Text(
                            text = placeholder,
                            color = textSecondary(),
                            fontSize = 16.sp,
                        )
                    }
                    inner()
                }
            },
        )
    }
}

@Composable
fun AuthPrimaryButton(
    text: String,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    enabled: Boolean = true,
) {
    Button(
        onClick = onClick,
        enabled = enabled,
        modifier = modifier
            .fillMaxWidth()
            .height(44.dp),
        shape = RoundedCornerShape(8.dp),
        contentPadding = PaddingValues(vertical = 11.dp),
        colors = ButtonDefaults.buttonColors(
            containerColor = LexturesColors.Primary,
            contentColor = Color.White,
            disabledContainerColor = LexturesColors.Primary.copy(alpha = 0.55f),
            disabledContentColor = Color.White.copy(alpha = 0.92f),
        ),
    ) {
        Text(
            text = text,
            fontWeight = FontWeight.SemiBold,
            fontSize = 15.sp,
            color = Color.White,
        )
    }
}

@Composable
fun AuthFooterLink(
    prompt: String,
    actionLabel: String,
    onAction: () -> Unit,
    modifier: Modifier = Modifier,
) {
    Row(
        modifier = modifier
            .fillMaxWidth()
            .padding(top = 4.dp),
        horizontalArrangement = Arrangement.Center,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(
            text = prompt,
            fontSize = 15.sp,
            color = textSecondary(),
        )
        Text(
            text = actionLabel,
            modifier = Modifier
                .padding(start = 4.dp)
                .clickable(onClick = onAction),
            fontSize = 15.sp,
            fontWeight = FontWeight.Medium,
            color = LexturesColors.PrimaryMuted,
        )
    }
}
