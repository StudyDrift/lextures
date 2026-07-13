package com.lextures.android.features.settings.admin

import android.content.Intent
import android.net.Uri
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Email
import androidx.compose.material.icons.filled.Lock
import androidx.compose.material.icons.filled.OpenInBrowser
import androidx.compose.material.icons.filled.Person
import androidx.compose.material.icons.filled.Search
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.config.AppConfiguration
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalePreferences
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.PaginatedPeople
import com.lextures.android.core.lms.PeopleAdminLogic
import com.lextures.android.core.lms.PersonRow
import com.lextures.android.features.home.HomeShellState
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsSkeletonList
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun PeopleAdminScreen(
    session: AuthSession,
    shell: HomeShellState,
    localePrefs: LocalePreferences,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val accessToken by session.accessToken.collectAsState()

    var searchText by remember { mutableStateOf("") }
    var submittedQuery by remember { mutableStateOf("") }
    var page by remember { mutableIntStateOf(1) }
    var results by remember { mutableStateOf<PaginatedPeople?>(null) }
    var loading by remember { mutableStateOf(false) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var showInviteSheet by remember { mutableStateOf(false) }
    var selectedUserId by remember { mutableStateOf<String?>(null) }

    val canView = PeopleAdminLogic.canView(shell.platformFeatures, shell.permissions)
    val genericError = L.text(context, localePrefs, R.string.mobile_admin_people_error)

    suspend fun search(token: String) {
        if (!PeopleAdminLogic.shouldSearch(submittedQuery)) {
            results = null
            return
        }
        loading = true
        errorMessage = null
        runCatching {
            results = LmsApi.searchPeople(
                query = submittedQuery,
                page = page,
                perPage = PeopleAdminLogic.DEFAULT_PER_PAGE,
                accessToken = token,
            )
        }.onFailure {
            errorMessage = PeopleAdminLogic.userFacingError(it, genericError)
            results = null
        }
        loading = false
    }

    LaunchedEffect(accessToken, submittedQuery, page) {
        val token = accessToken ?: return@LaunchedEffect
        if (canView && submittedQuery.isNotEmpty()) search(token)
    }

    if (selectedUserId != null) {
        UserDetailAdminScreen(
            session = session,
            shell = shell,
            localePrefs = localePrefs,
            userId = selectedUserId!!,
            onBack = { selectedUserId = null },
            modifier = modifier,
        )
        return
    }

    val inviteSheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)

    Scaffold(
        modifier = modifier.fillMaxSize(),
        topBar = {
            TopAppBar(
                title = { Text(L.text(context, localePrefs, R.string.mobile_admin_people_title)) },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = null)
                    }
                },
            )
        },
    ) { padding ->
        if (!canView) {
            LmsEmptyState(
                icon = Icons.Default.Lock,
                title = L.text(context, localePrefs, R.string.mobile_admin_people_accessDeniedTitle),
                message = L.text(context, localePrefs, R.string.mobile_admin_people_accessDeniedMessage),
                modifier = Modifier.padding(padding).padding(16.dp),
            )
            return@Scaffold
        }

        Column(
            modifier = Modifier
                .padding(padding)
                .fillMaxSize()
                .verticalScroll(rememberScrollState())
                .padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp),
        ) {
            Text(
                L.text(context, localePrefs, R.string.mobile_admin_people_description),
                color = textSecondary(),
            )

            LmsCard {
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .clickable {
                            val url = AppConfiguration.webUrl(
                                PeopleAdminLogic.webSettingsPath(),
                            )
                            context.startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(url)))
                        }
                        .padding(12.dp),
                    horizontalArrangement = Arrangement.spacedBy(10.dp),
                ) {
                    Icon(Icons.Default.OpenInBrowser, contentDescription = null)
                    Column(modifier = Modifier.weight(1f)) {
                        Text(
                            L.text(context, localePrefs, R.string.mobile_admin_people_webTitle),
                            fontWeight = FontWeight.SemiBold,
                            color = textPrimary(),
                        )
                        Text(
                            L.text(context, localePrefs, R.string.mobile_admin_people_webHint),
                            color = textSecondary(),
                        )
                    }
                }
            }

            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                OutlinedTextField(
                    value = searchText,
                    onValueChange = { searchText = it },
                    label = { Text(L.text(context, localePrefs, R.string.mobile_admin_people_search)) },
                    modifier = Modifier.weight(1f),
                    singleLine = true,
                )
                Button(
                    onClick = {
                        submittedQuery = PeopleAdminLogic.normalizedSearchQuery(searchText)
                        page = 1
                    },
                ) {
                    Icon(Icons.Default.Search, contentDescription = null)
                }
            }

            OutlinedButton(
                onClick = { showInviteSheet = true },
                modifier = Modifier.fillMaxWidth(),
            ) {
                Icon(Icons.Default.Email, contentDescription = null)
                Text(L.text(context, localePrefs, R.string.mobile_admin_people_invite))
            }

            errorMessage?.let { LmsErrorBanner(it) }

            when {
                loading && results == null -> LmsSkeletonList(count = 3)
                results != null -> resultsSection(
                    context = context,
                    localePrefs = localePrefs,
                    data = results!!,
                    loading = loading,
                    onSelectUser = { selectedUserId = it },
                    onPrevious = { if (page > 1) page -= 1 },
                    onNext = {
                        val totalPages = results?.totalPages ?: 1
                        if (page < totalPages) page += 1
                    },
                )
                !PeopleAdminLogic.shouldSearch(submittedQuery) -> {
                    LmsEmptyState(
                        icon = Icons.Default.Person,
                        title = L.text(context, localePrefs, R.string.mobile_admin_people_emptyTitle),
                        message = L.text(context, localePrefs, R.string.mobile_admin_people_emptyMessage),
                    )
                }
            }
        }
    }

    if (showInviteSheet) {
        ModalBottomSheet(
            onDismissRequest = { showInviteSheet = false },
            sheetState = inviteSheetState,
        ) {
            PeopleInviteSheet(
                session = session,
                localePrefs = localePrefs,
                onDismiss = { showInviteSheet = false },
                onInvited = { email ->
                    searchText = email
                    submittedQuery = email
                    page = 1
                    showInviteSheet = false
                    accessToken?.let { token ->
                        scope.launch { search(token) }
                    }
                },
            )
        }
    }
}

@Composable
private fun resultsSection(
    context: android.content.Context,
    localePrefs: LocalePreferences,
    data: PaginatedPeople,
    loading: Boolean,
    onSelectUser: (String) -> Unit,
    onPrevious: () -> Unit,
    onNext: () -> Unit,
) {
    if (data.items.isEmpty()) {
        LmsEmptyState(
            icon = Icons.Default.Search,
            title = L.text(context, localePrefs, R.string.mobile_admin_people_emptyTitle),
            message = L.text(context, localePrefs, R.string.mobile_admin_people_emptySearch),
        )
        return
    }

    Text(
        L.format(context, localePrefs, R.string.mobile_admin_people_resultsCount, data.total.toInt()),
        color = textSecondary(),
    )

    data.items.forEach { person ->
        LmsCard {
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .clickable { onSelectUser(person.id) }
                    .padding(12.dp),
            ) {
                Text(
                    PeopleAdminLogic.personDisplayName(person),
                    fontWeight = FontWeight.SemiBold,
                    color = textPrimary(),
                )
                Text(person.email, color = textSecondary())
                Text(
                    "${person.orgName} · ${
                        PeopleAdminLogic.statusLabel(
                            person.active,
                            L.text(context, localePrefs, R.string.mobile_admin_people_status_active),
                            L.text(context, localePrefs, R.string.mobile_admin_people_status_suspended),
                        )
                    }",
                    color = textSecondary(),
                )
            }
        }
    }

    if (data.totalPages > 1) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
        ) {
            TextButton(onClick = onPrevious, enabled = data.page > 1 && !loading) {
                Text(L.text(context, localePrefs, R.string.mobile_admin_people_previous))
            }
            Text(
                L.format(
                    context,
                    localePrefs,
                    R.string.mobile_admin_people_pageOf,
                    data.page,
                    data.totalPages,
                ),
                color = textSecondary(),
            )
            TextButton(onClick = onNext, enabled = data.page < data.totalPages && !loading) {
                Text(L.text(context, localePrefs, R.string.mobile_admin_people_next))
            }
        }
    }
}

@Composable
private fun PeopleInviteSheet(
    session: AuthSession,
    localePrefs: LocalePreferences,
    onDismiss: () -> Unit,
    onInvited: (String) -> Unit,
) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val accessToken by session.accessToken.collectAsState()
    val genericError = L.text(context, localePrefs, R.string.mobile_admin_people_error)

    var email by remember { mutableStateOf("") }
    var firstName by remember { mutableStateOf("") }
    var lastName by remember { mutableStateOf("") }
    var busy by remember { mutableStateOf(false) }
    var errorMessage by remember { mutableStateOf<String?>(null) }

    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        Text(
            L.text(context, localePrefs, R.string.mobile_admin_people_inviteTitle),
            fontWeight = FontWeight.SemiBold,
            color = textPrimary(),
        )
        errorMessage?.let { LmsErrorBanner(it) }
        OutlinedTextField(
            value = email,
            onValueChange = { email = it },
            label = { Text(L.text(context, localePrefs, R.string.mobile_admin_people_inviteEmail)) },
            modifier = Modifier.fillMaxWidth(),
            singleLine = true,
        )
        OutlinedTextField(
            value = firstName,
            onValueChange = { firstName = it },
            label = { Text(L.text(context, localePrefs, R.string.mobile_admin_people_inviteFirstName)) },
            modifier = Modifier.fillMaxWidth(),
            singleLine = true,
        )
        OutlinedTextField(
            value = lastName,
            onValueChange = { lastName = it },
            label = { Text(L.text(context, localePrefs, R.string.mobile_admin_people_inviteLastName)) },
            modifier = Modifier.fillMaxWidth(),
            singleLine = true,
        )
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            TextButton(onClick = onDismiss) {
                Text(L.text(context, localePrefs, R.string.mobile_common_cancel))
            }
            Button(
                onClick = {
                    val token = accessToken ?: return@Button
                    scope.launch {
                        busy = true
                        errorMessage = null
                        val request = PeopleAdminLogic.invitePersonRequest(email, firstName, lastName)
                        runCatching {
                            LmsApi.invitePerson(request, token)
                        }.onSuccess {
                            onInvited(request.email)
                        }.onFailure {
                            errorMessage = PeopleAdminLogic.userFacingError(it, genericError)
                        }
                        busy = false
                    }
                },
                enabled = !busy && email.trim().isNotEmpty(),
            ) {
                Text(
                    if (busy) {
                        L.text(context, localePrefs, R.string.mobile_admin_people_loading)
                    } else {
                        L.text(context, localePrefs, R.string.mobile_admin_people_inviteSend)
                    },
                )
            }
        }
    }
}
