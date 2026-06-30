package com.lextures.android.features.profile

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
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
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.LexturesType
import com.lextures.android.core.design.ProfileAvatar
import com.lextures.android.core.design.accentColor
import com.lextures.android.core.design.sceneBackground
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.lms.AccountProfilePatch
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.nameFieldsFromProfile
import com.lextures.android.core.network.ApiError
import com.lextures.android.features.home.HomeShellState
import com.lextures.android.features.home.LmsCard
import kotlinx.coroutines.launch

/** Edits the student's server-backed profile: name, avatar URL, and phone (FR-1). */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun EditProfileScreen(
    session: AuthSession,
    shell: HomeShellState,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    val scope = rememberCoroutineScope()
    val saveErrorText = L.text(R.string.mobile_editProfile_saveError)

    var loading by remember { mutableStateOf(true) }
    var loadFailed by remember { mutableStateOf(false) }
    var saving by remember { mutableStateOf(false) }
    var saved by remember { mutableStateOf(false) }
    var errorMessage by remember { mutableStateOf<String?>(null) }

    var firstName by remember { mutableStateOf("") }
    var lastName by remember { mutableStateOf("") }
    var avatarUrl by remember { mutableStateOf("") }
    var phone by remember { mutableStateOf("") }
    var email by remember { mutableStateOf("") }

    fun load() {
        val token = accessToken ?: run {
            loading = false
            loadFailed = true
            return
        }
        scope.launch {
            loading = true
            loadFailed = false
            try {
                val profile = LmsApi.fetchAccountProfile(token)
                val (first, last) = nameFieldsFromProfile(profile)
                firstName = first
                lastName = last
                avatarUrl = profile.avatarUrl.orEmpty()
                phone = profile.phoneNumber.orEmpty()
                email = profile.email
            } catch (_: Exception) {
                loadFailed = true
            } finally {
                loading = false
            }
        }
    }

    LaunchedEffect(accessToken) { load() }

    fun save() {
        val token = accessToken ?: return
        scope.launch {
            saving = true
            saved = false
            errorMessage = null
            try {
                val updated = LmsApi.updateAccountProfile(
                    AccountProfilePatch(
                        firstName = firstName.trim(),
                        lastName = lastName.trim(),
                        avatarUrl = avatarUrl.trim(),
                        phoneNumber = phone.trim(),
                    ),
                    token,
                )
                val (savedFirst, savedLast) = nameFieldsFromProfile(updated)
                firstName = savedFirst
                lastName = savedLast
                avatarUrl = updated.avatarUrl.orEmpty()
                phone = updated.phoneNumber.orEmpty()
                shell.refresh(token)
                saved = true
            } catch (e: ApiError.HttpStatus) {
                errorMessage = e.apiMessage ?: saveErrorText
            } catch (_: Exception) {
                errorMessage = saveErrorText
            } finally {
                saving = false
            }
        }
    }

    Column(
        modifier = modifier
            .fillMaxSize()
            .background(sceneBackground()),
    ) {
        TopAppBar(
            title = { Text(L.text(R.string.mobile_editProfile_title)) },
            navigationIcon = {
                IconButton(onClick = onBack) {
                    Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = null)
                }
            },
        )

        when {
            loading -> Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                CircularProgressIndicator(color = accentColor())
            }

            loadFailed -> Column(
                modifier = Modifier.fillMaxSize().padding(32.dp),
                horizontalAlignment = Alignment.CenterHorizontally,
                verticalArrangement = Arrangement.Center,
            ) {
                Text(L.text(R.string.mobile_editProfile_loadError), color = textSecondary())
                TextButton(onClick = { load() }) { Text(L.text(R.string.mobile_common_retry)) }
            }

            else -> Column(
                modifier = Modifier
                    .fillMaxSize()
                    .verticalScroll(rememberScrollState())
                    .padding(16.dp),
                verticalArrangement = Arrangement.spacedBy(16.dp),
            ) {
                AvatarPreview(
                    firstName = firstName,
                    lastName = lastName,
                    email = email,
                    avatarUrl = avatarUrl,
                )

                LmsCard {
                    OutlinedTextField(
                        value = firstName,
                        onValueChange = { firstName = it },
                        label = { Text(L.text(R.string.mobile_editProfile_firstName)) },
                        singleLine = true,
                        modifier = Modifier.fillMaxWidth(),
                    )
                    OutlinedTextField(
                        value = lastName,
                        onValueChange = { lastName = it },
                        label = { Text(L.text(R.string.mobile_editProfile_lastName)) },
                        singleLine = true,
                        modifier = Modifier.fillMaxWidth().padding(top = 8.dp),
                    )
                }

                LmsCard {
                    OutlinedTextField(
                        value = avatarUrl,
                        onValueChange = { avatarUrl = it },
                        label = { Text(L.text(R.string.mobile_editProfile_avatarUrl)) },
                        singleLine = true,
                        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Uri),
                        modifier = Modifier.fillMaxWidth(),
                    )
                    Text(
                        text = L.text(R.string.mobile_editProfile_avatarHint),
                        fontSize = 12.sp,
                        color = textSecondary(),
                        modifier = Modifier.padding(top = 4.dp),
                    )
                }

                LmsCard {
                    OutlinedTextField(
                        value = phone,
                        onValueChange = { phone = it },
                        label = { Text(L.text(R.string.mobile_editProfile_phone)) },
                        singleLine = true,
                        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Phone),
                        modifier = Modifier.fillMaxWidth(),
                    )
                    Text(
                        text = L.text(R.string.mobile_profile_email),
                        fontSize = 11.sp,
                        color = textSecondary(),
                        modifier = Modifier.padding(top = 12.dp),
                    )
                    Text(
                        text = email.ifEmpty { "—" },
                        fontSize = 14.sp,
                        color = textSecondary(),
                    )
                }

                errorMessage?.let {
                    Text(text = it, color = LexturesColors.Error, fontSize = 13.sp)
                }

                Button(
                    onClick = { save() },
                    enabled = !saving,
                    modifier = Modifier.fillMaxWidth(),
                    colors = ButtonDefaults.buttonColors(
                        containerColor = if (saved) LexturesColors.BrandTeal else LexturesColors.PrimaryDeep,
                    ),
                ) {
                    if (saving) {
                        CircularProgressIndicator(color = Color.White, modifier = Modifier.size(20.dp))
                    } else {
                        Text(
                            text = if (saved) {
                                L.text(R.string.mobile_editProfile_saved)
                            } else {
                                L.text(R.string.mobile_common_save)
                            },
                            color = Color.White,
                            fontWeight = FontWeight.SemiBold,
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun AvatarPreview(
    firstName: String,
    lastName: String,
    email: String,
    avatarUrl: String,
) {
    val initials = remember(firstName, lastName, email) {
        val letters = "$firstName $lastName".split(" ").mapNotNull { it.firstOrNull() }
        when {
            letters.size >= 2 -> "${letters[0]}${letters[1]}".uppercase()
            letters.size == 1 -> letters[0].toString().uppercase()
            else -> email.take(1).uppercase()
        }
    }
    val name = "$firstName $lastName".trim().ifEmpty { email }
    Column(
        modifier = Modifier.fillMaxWidth(),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(10.dp),
    ) {
        ProfileAvatar(
            avatarUrl = avatarUrl,
            initials = initials,
            size = 84.dp,
        )
        if (name.isNotEmpty()) {
            Text(text = name, style = LexturesType.display(18), color = textPrimary())
        }
    }
}
