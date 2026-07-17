package com.lextures.android.features.boards.publicboard

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.input.VisualTransformation
import androidx.compose.ui.unit.dp
import com.lextures.android.R
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.lms.Board
import com.lextures.android.core.lms.BoardAccessApi
import com.lextures.android.core.lms.BoardLinkAccessState
import com.lextures.android.core.lms.BoardPost
import com.lextures.android.core.lms.BoardShareCapability
import com.lextures.android.core.lms.BoardsLogic
import com.lextures.android.core.network.ApiError
import kotlinx.coroutines.launch

/** Minimal public board view for share links (VC.M6). No course nav / roster PII. */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun BoardPublicScreen(
    token: String,
    onClose: () -> Unit,
) {
    val scope = rememberCoroutineScope()
    var state by remember { mutableStateOf(BoardLinkAccessState.Loading) }
    var board by remember { mutableStateOf<Board?>(null) }
    var posts by remember { mutableStateOf<List<BoardPost>>(emptyList()) }
    var capability by remember { mutableStateOf(BoardShareCapability.View) }
    var password by remember { mutableStateOf("") }
    var showPassword by remember { mutableStateOf(false) }
    var displayName by remember { mutableStateOf("") }
    var draft by remember { mutableStateOf("") }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var busy by remember { mutableStateOf(false) }

    val passwordRequired = L.text(R.string.mobile_boards_share_passwordRequired)
    val externalDisabled = L.text(R.string.mobile_boards_share_externalDisabled)
    val linkInvalid = L.text(R.string.mobile_boards_share_linkInvalid)
    val saveError = L.text(R.string.mobile_boards_share_saveError)

    fun load(pw: String? = null) {
        scope.launch {
            busy = true
            errorMessage = null
            if (pw == null) state = BoardLinkAccessState.Loading
            try {
                val data = BoardAccessApi.resolveBoardLink(
                    token,
                    pw ?: password.ifBlank { null },
                )
                board = data.board
                posts = data.posts
                capability = BoardShareCapability.fromApi(data.capability)
                state = BoardLinkAccessState.Ready
            } catch (e: ApiError.HttpStatus) {
                board = null
                posts = emptyList()
                val classified = BoardsLogic.classifyBoardLinkError(e.code, e.message)
                state = classified
                errorMessage = when (classified) {
                    BoardLinkAccessState.NeedsPassword -> passwordRequired
                    BoardLinkAccessState.Denied -> if (e.code == 403) externalDisabled else linkInvalid
                    else -> linkInvalid
                }
            } catch (_: Exception) {
                state = BoardLinkAccessState.Denied
                errorMessage = linkInvalid
            } finally {
                busy = false
            }
        }
    }

    LaunchedEffect(token) { load() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text(L.text(R.string.mobile_boards_share_publicLabel)) },
                navigationIcon = {
                    TextButton(onClick = onClose) { Text(L.text(R.string.mobile_common_close)) }
                },
            )
        },
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            when (state) {
                BoardLinkAccessState.Loading -> CircularProgressIndicator()
                BoardLinkAccessState.NeedsPassword -> {
                    Text(
                        L.text(R.string.mobile_boards_share_passwordPrompt),
                        fontWeight = FontWeight.SemiBold,
                        color = textPrimary(),
                    )
                    errorMessage?.let { msg ->
                        Text(msg, color = androidx.compose.ui.graphics.Color.Red)
                    }
                    OutlinedTextField(
                        value = password,
                        onValueChange = { password = it },
                        modifier = Modifier.fillMaxWidth(),
                        label = { Text(L.text(R.string.mobile_boards_share_passwordOptional)) },
                        visualTransformation = if (showPassword) {
                            VisualTransformation.None
                        } else {
                            PasswordVisualTransformation()
                        },
                        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Password),
                        trailingIcon = {
                            TextButton(onClick = { showPassword = !showPassword }) {
                                Text(
                                    if (showPassword) L.text(R.string.mobile_boards_share_hidePassword)
                                    else L.text(R.string.mobile_boards_share_showPassword),
                                )
                            }
                        },
                        singleLine = true,
                    )
                    Button(onClick = { load(password) }, enabled = !busy) {
                        Text(L.text(R.string.mobile_boards_share_unlock))
                    }
                }
                BoardLinkAccessState.Denied -> {
                    Text(
                        errorMessage ?: linkInvalid,
                        color = androidx.compose.ui.graphics.Color.Red,
                    )
                }
                BoardLinkAccessState.Ready -> {
                    val b = board
                    if (b != null) {
                        Text(b.title, fontWeight = FontWeight.SemiBold, color = textPrimary())
                        if (b.description.isNotBlank()) {
                            Text(b.description, color = textSecondary())
                        }
                        if (capability == BoardShareCapability.View) {
                            Text(L.text(R.string.mobile_boards_share_readOnly), color = textSecondary())
                        }
                        if (capability == BoardShareCapability.Contribute) {
                            OutlinedTextField(
                                value = displayName,
                                onValueChange = { displayName = it },
                                modifier = Modifier.fillMaxWidth(),
                                label = { Text(L.text(R.string.mobile_boards_share_displayName)) },
                                singleLine = true,
                            )
                            OutlinedTextField(
                                value = draft,
                                onValueChange = { draft = it },
                                modifier = Modifier.fillMaxWidth(),
                                label = { Text(L.text(R.string.mobile_boards_compose_bodyLabel)) },
                                minLines = 3,
                            )
                            Button(
                                onClick = {
                                    val name = displayName.trim()
                                    val text = draft.trim()
                                    if (name.isEmpty() || text.isEmpty()) return@Button
                                    scope.launch {
                                        busy = true
                                        errorMessage = null
                                        try {
                                            val post = BoardAccessApi.createBoardLinkPost(
                                                token,
                                                name,
                                                text,
                                                password.ifBlank { null },
                                            )
                                            posts = listOf(post) + posts
                                            draft = ""
                                        } catch (e: Exception) {
                                            errorMessage = e.message ?: saveError
                                        } finally {
                                            busy = false
                                        }
                                    }
                                },
                                enabled = !busy && displayName.isNotBlank() && draft.isNotBlank(),
                            ) {
                                Text(L.text(R.string.mobile_boards_share_postAsGuest))
                            }
                        }
                        errorMessage?.let { msg ->
                            Text(msg, color = androidx.compose.ui.graphics.Color.Red)
                        }
                        LazyColumn(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                            items(posts, key = { it.id }) { post ->
                                Card(modifier = Modifier.fillMaxWidth()) {
                                    Column(
                                        modifier = Modifier.padding(12.dp),
                                        verticalArrangement = Arrangement.spacedBy(4.dp),
                                    ) {
                                        if (post.title.isNotBlank()) {
                                            Text(
                                                post.title,
                                                fontWeight = FontWeight.Medium,
                                                color = textPrimary(),
                                            )
                                        }
                                        val body = BoardsLogic.bodyPlainText(post)
                                        if (body.isNotBlank()) {
                                            Text(body, color = textPrimary())
                                        }
                                        BoardsLogic.attributionLabel(post)?.let { label ->
                                            Text(label, color = textSecondary())
                                        }
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}
