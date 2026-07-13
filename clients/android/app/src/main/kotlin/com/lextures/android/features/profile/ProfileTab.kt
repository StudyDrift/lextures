package com.lextures.android.features.profile

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.KeyboardArrowRight
import androidx.compose.material.icons.automirrored.filled.Logout
import androidx.compose.material.icons.filled.Apps
import androidx.compose.material.icons.filled.CreditCard
import androidx.compose.material.icons.filled.Dns
import androidx.compose.material.icons.filled.Email
import androidx.compose.material.icons.filled.Link
import androidx.compose.material.icons.filled.Notifications
import androidx.compose.material.icons.filled.OpenInNew
import androidx.compose.material.icons.filled.Person
import androidx.compose.material.icons.filled.Search
import androidx.compose.material.icons.filled.Settings
import androidx.compose.material.icons.automirrored.filled.FormatListBulleted
import androidx.compose.material.icons.filled.Security
import androidx.compose.material.icons.filled.VerifiedUser
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExposedDropdownMenuBox
import androidx.compose.material3.ExposedDropdownMenuDefaults
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material.icons.filled.Archive
import androidx.compose.material.icons.filled.Apartment
import androidx.compose.material.icons.filled.Brush
import androidx.compose.material.icons.filled.Chat
import androidx.compose.material.icons.filled.Delete
import androidx.compose.material.icons.filled.Storage
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.platform.LocalContext
import com.lextures.android.core.design.OutboxStatusChip
import com.lextures.android.core.offline.OutboxStatus
import com.lextures.android.core.offline.OfflineService
import kotlinx.coroutines.launch
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.BuildConfig
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.config.AppConfiguration
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalLocalePreferences
import com.lextures.android.core.i18n.LocaleApi
import com.lextures.android.core.i18n.LocalePreferences
import com.lextures.android.core.design.HeroBrush
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.LexturesType
import com.lextures.android.core.design.accentColor
import com.lextures.android.core.design.ThemeAppearance
import com.lextures.android.core.design.ThemePreference
import com.lextures.android.core.design.isDarkTheme
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.features.home.HomeShellState
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.MoreDestinationPlaceholder
import com.lextures.android.features.home.MoreHubScreen
import com.lextures.android.core.search.SearchRecentsStore

import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.ProfileDepthLogic
import com.lextures.android.core.lms.resolvedDisplayName
import com.lextures.android.core.lms.resolvedInitials
import com.lextures.android.core.design.ProfileAvatar
import androidx.compose.material3.Switch
import androidx.compose.material3.SwitchDefaults
import com.lextures.android.core.accessibility.rememberAccessibilityPreferencesState
import com.lextures.android.core.auth.BiometricGate

/** Profile tab: identity hero, notifications, app info, and sign-out. */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ProfileTab(
    session: AuthSession,
    shell: HomeShellState,
    modifier: Modifier = Modifier,
) {
    var confirmingSignOut by remember { mutableStateOf(false) }
    var confirmingClearCache by remember { mutableStateOf(false) }
    var showNotifications by remember { mutableStateOf(false) }
    var showNotificationPreferences by remember { mutableStateOf(false) }
    var showDeviceSessions by remember { mutableStateOf(false) }
    var showEditProfile by remember { mutableStateOf(false) }
    var showAccommodations by remember { mutableStateOf(false) }
    var showPersonalDetails by remember { mutableStateOf(false) }
    var showResearchStudies by remember { mutableStateOf(false) }
    var showLearnerProfile by remember { mutableStateOf(false) }
    var showIntegrations by remember { mutableStateOf(false) }
    var showShareFeedback by remember { mutableStateOf(false) }
    var feedbackSuccessMessage by remember { mutableStateOf<String?>(null) }
    var showArchivedCoursesAdmin by remember { mutableStateOf(false) }
    var showRolesPermissionsAdmin by remember { mutableStateOf(false) }
    var showPeopleAdmin by remember { mutableStateOf(false) }
    var showOrgStructureAdmin by remember { mutableStateOf(false) }
    var showOrgBrandingAdmin by remember { mutableStateOf(false) }
    var showPlatformSettingsAdmin by remember { mutableStateOf(false) }
    var personalDetailsVisible by remember { mutableStateOf(false) }
    var researchVisible by remember { mutableStateOf(false) }
    var showMoreHub by remember { mutableStateOf(false) }
    var confirmingClearSearchHistory by remember { mutableStateOf(false) }
    var openMoreDestination by remember { mutableStateOf<com.lextures.android.core.navigation.MoreDestination?>(null) }
    var openPeerReviewAllocationId by remember { mutableStateOf<String?>(null) }
    var openPathProgress by remember { mutableStateOf<com.lextures.android.core.lms.PathProgress?>(null) }
    var showPathsCatalog by remember { mutableStateOf(false) }
    var openPathLandingSlug by remember { mutableStateOf<String?>(null) }
    var openPathCourse by remember { mutableStateOf<com.lextures.android.core.lms.CourseSummary?>(null) }
    var openCatalogCourseSlug by remember { mutableStateOf<String?>(null) }
    var openMarketplaceSlug by remember { mutableStateOf<String?>(null) }
    var showBilling by remember { mutableStateOf(false) }
    var openCredential by remember { mutableStateOf<com.lextures.android.core.lms.IssuedCredentialSummary?>(null) }
    var openPortfolioId by remember { mutableStateOf<String?>(null) }
    var openPortfolioTitle by remember { mutableStateOf<String?>(null) }
    var openPortfolioArtifact by remember { mutableStateOf<com.lextures.android.core.lms.PortfolioArtifact?>(null) }
    var openCatalogPathSlug by remember { mutableStateOf<String?>(null) }
    var openReadingLibraryOrgId by remember { mutableStateOf<String?>(null) }
    var readingLogBook by remember { mutableStateOf<com.lextures.android.core.lms.LibraryBook?>(null) }
    val accessToken by session.accessToken.collectAsState()
    val context = LocalContext.current
    val themePreference = remember { ThemePreference.get(context) }
    val biometricGate = remember { BiometricGate.get(context) }
    val offline = remember { OfflineService.get(context) }
    val pendingCount by offline.pendingCount.collectAsState()
    val isOnline by offline.networkMonitor.isOnline.collectAsState()
    val storageBytes by offline.storageBytes.collectAsState()
    val outboxItems by offline.outboxItems.collectAsState()
    val scope = rememberCoroutineScope()
    val accessibilityState = rememberAccessibilityPreferencesState()
    val uiModeStore = com.lextures.android.core.design.LocalUIModeStore.current
    val localePreferences = LocalLocalePreferences.current
    var localeExpanded by remember { mutableStateOf(false) }
    var localeError by remember { mutableStateOf<String?>(null) }

    LaunchedEffect(shell.pendingProfileSettingsRoute) {
        when (shell.consumePendingProfileSettingsRoute()) {
            com.lextures.android.core.routing.SettingsDeepLinkSection.Account -> showEditProfile = true
            com.lextures.android.core.routing.SettingsDeepLinkSection.Notifications -> showNotificationPreferences = true
            com.lextures.android.core.routing.SettingsDeepLinkSection.LearnerProfile -> {
                if (com.lextures.android.core.lms.LearnerProfileLogic.learnerProfileEnabled(shell.platformFeatures)) {
                    showLearnerProfile = true
                }
            }
            null -> Unit
        }
    }

    LaunchedEffect(shell.profileDepthEnabled, accessToken, shell.platformFeatures) {
        if (!shell.profileDepthEnabled || accessToken == null) {
            personalDetailsVisible = false
            researchVisible = false
            return@LaunchedEffect
        }
        val token = accessToken!!
        var fields = 0
        if (shell.platformFeatures.customFieldsEnabled) {
            fields = try {
                LmsApi.fetchMyProfileFields(token).fields.size
            } catch (_: Exception) {
                0
            }
        }
        personalDetailsVisible = ProfileDepthLogic.shouldShowPersonalDetails(
            demographicsEnabled = shell.platformFeatures.ffDemographics,
            fieldCount = fields,
        )
        if (shell.platformFeatures.ffResearchConsent) {
            val pending = try {
                LmsApi.fetchPendingConsentStudies(token).size
            } catch (_: Exception) {
                0
            }
            val history = try {
                LmsApi.fetchConsentHistory(token).size
            } catch (_: Exception) {
                0
            }
            researchVisible = ProfileDepthLogic.shouldShowResearchStudies(
                researchConsentEnabled = true,
                pendingCount = pending,
                historyCount = history,
            )
        } else {
            researchVisible = false
        }
    }

    if (showNotificationPreferences) {
        NotificationPreferencesScreen(
            session = session,
            onBack = { showNotificationPreferences = false },
            modifier = modifier,
        )
        return
    }

    if (showNotifications) {
        NotificationsScreen(
            session = session,
            shell = shell,
            onBack = { showNotifications = false },
            onOpenPreferences = { showNotificationPreferences = true },
            modifier = modifier,
        )
        return
    }

    if (showBilling) {
        Column(modifier = modifier.fillMaxSize()) {
            TextButton(onClick = { showBilling = false }) {
                Text(L.text(context, localePreferences, R.string.mobile_ia_close))
            }
            com.lextures.android.features.billing.BillingScreen(
                session = session,
                shell = shell,
                localePrefs = localePreferences,
                modifier = Modifier.fillMaxSize(),
            )
        }
        return
    }

    if (showDeviceSessions) {
        DeviceSessionsScreen(
            session = session,
            onBack = { showDeviceSessions = false },
            modifier = modifier,
        )
        return
    }

    if (showEditProfile) {
        EditProfileScreen(
            session = session,
            shell = shell,
            onBack = { showEditProfile = false },
            modifier = modifier,
        )
        return
    }

    if (showAccommodations) {
        MyAccommodationsScreen(
            session = session,
            onBack = { showAccommodations = false },
            modifier = modifier,
        )
        return
    }

    if (showPersonalDetails) {
        ProfilePersonalDetailsScreen(
            session = session,
            shell = shell,
            onBack = { showPersonalDetails = false },
            modifier = modifier,
        )
        return
    }

    if (showResearchStudies) {
        ResearchStudiesScreen(
            session = session,
            onBack = { showResearchStudies = false },
            modifier = modifier,
        )
        return
    }

    if (showIntegrations) {
        com.lextures.android.features.settings.IntegrationsScreen(
            session = session,
            shell = shell,
            localePrefs = localePreferences,
            onBack = { showIntegrations = false },
            modifier = modifier,
        )
        return
    }

    if (showArchivedCoursesAdmin) {
        com.lextures.android.features.settings.admin.ArchivedCoursesAdminScreen(
            session = session,
            shell = shell,
            localePrefs = localePreferences,
            onBack = { showArchivedCoursesAdmin = false },
            modifier = modifier,
        )
        return
    }

    if (showRolesPermissionsAdmin) {
        com.lextures.android.features.settings.admin.RolesPermissionsAdminScreen(
            session = session,
            shell = shell,
            localePrefs = localePreferences,
            onBack = { showRolesPermissionsAdmin = false },
            modifier = modifier,
        )
        return
    }

    if (showPeopleAdmin) {
        com.lextures.android.features.settings.admin.PeopleAdminScreen(
            session = session,
            shell = shell,
            localePrefs = localePreferences,
            onBack = { showPeopleAdmin = false },
            modifier = modifier,
        )
        return
    }

    if (showOrgStructureAdmin) {
        com.lextures.android.features.settings.admin.OrgStructureAdminScreen(
            session = session,
            shell = shell,
            localePrefs = localePreferences,
            onBack = { showOrgStructureAdmin = false },
            modifier = modifier,
        )
        return
    }

    if (showOrgBrandingAdmin) {
        com.lextures.android.features.settings.admin.OrgBrandingAdminScreen(
            session = session,
            shell = shell,
            localePrefs = localePreferences,
            onBack = { showOrgBrandingAdmin = false },
            modifier = modifier,
        )
        return
    }

    if (showPlatformSettingsAdmin) {
        com.lextures.android.features.settings.admin.PlatformSettingsScreen(
            session = session,
            shell = shell,
            localePrefs = localePreferences,
            onBack = { showPlatformSettingsAdmin = false },
            modifier = modifier,
        )
        return
    }

    if (showLearnerProfile) {
        com.lextures.android.features.settings.learnerprofile.LearnerProfileScreen(
            session = session,
            localePrefs = localePreferences,
            onBack = { showLearnerProfile = false },
            modifier = modifier,
        )
        return
    }

    openMoreDestination?.let { destination ->
        Column(modifier = modifier.fillMaxSize().padding(16.dp)) {
            TextButton(onClick = { openMoreDestination = null }) {
                Text(L.text(context, localePreferences, R.string.mobile_ia_close))
            }
            if (destination == com.lextures.android.core.navigation.MoreDestination.Library &&
                shell.platformFeatures.libraryBrowseEnabled
            ) {
                com.lextures.android.features.library.LibraryBrowseScreen(
                    session = session,
                    shell = shell,
                    modifier = Modifier.fillMaxSize(),
                )
            } else if (destination == com.lextures.android.core.navigation.MoreDestination.AskAi) {
                com.lextures.android.features.tutor.TutorChatScreen(
                    session = session,
                    mode = com.lextures.android.features.tutor.TutorChatMode.AskAi,
                    shell = shell,
                    onClose = { openMoreDestination = null },
                    modifier = Modifier.fillMaxSize(),
                )
            } else if (destination == com.lextures.android.core.navigation.MoreDestination.PeerReviews) {
                if (openPeerReviewAllocationId != null) {
                    com.lextures.android.features.peerreview.PeerReviewDetailScreen(
                        session = session,
                        allocationId = openPeerReviewAllocationId!!,
                        onSubmitted = { openPeerReviewAllocationId = null },
                        modifier = Modifier.fillMaxSize(),
                    )
                } else {
                    com.lextures.android.features.peerreview.PeerReviewListScreen(
                        session = session,
                        onOpenAllocation = { openPeerReviewAllocationId = it },
                        modifier = Modifier.fillMaxSize(),
                    )
                }
            } else if (destination == com.lextures.android.core.navigation.MoreDestination.ReportCards) {
                com.lextures.android.features.mastery.ReportCardListScreen(
                    session = session,
                    modifier = Modifier.fillMaxSize(),
                )
            } else if (destination == com.lextures.android.core.navigation.MoreDestination.Insights) {
                if (shell.platformFeatures.selfReflectionEnabled) {
                    com.lextures.android.features.insights.InsightsScreen(
                        session = session,
                        onOpenCourse = { course ->
                            openMoreDestination = null
                            shell.select(com.lextures.android.core.navigation.RootDestination.Courses)
                            shell.activeCourse = course
                            shell.activeCourseSection = com.lextures.android.core.navigation.CourseWorkspaceSection.Modules
                        },
                        onOpenReview = {
                            openMoreDestination = null
                            shell.select(com.lextures.android.core.navigation.RootDestination.Review)
                        },
                        modifier = Modifier.fillMaxSize(),
                    )
                } else {
                    MoreDestinationPlaceholder(destination = destination, modifier = Modifier.fillMaxSize())
                }
            } else if (destination == com.lextures.android.core.navigation.MoreDestination.Reading) {
                if (shell.platformFeatures.ffLibrary) {
                    when {
                        openReadingLibraryOrgId != null -> Column(modifier = Modifier.fillMaxSize().padding(16.dp)) {
                            TextButton(onClick = { openReadingLibraryOrgId = null }) {
                                Text(L.text(context, localePreferences, R.string.mobile_ia_close))
                            }
                            com.lextures.android.features.reading.LeveledLibraryScreen(
                                session = session,
                                orgId = openReadingLibraryOrgId!!,
                                onLogBook = { book ->
                                    readingLogBook = book
                                    openReadingLibraryOrgId = null
                                },
                                modifier = Modifier.fillMaxSize(),
                            )
                        }
                        else -> com.lextures.android.features.reading.ReadingDashboardScreen(
                            session = session,
                            onOpenBookClub = { course ->
                                openMoreDestination = null
                                shell.select(com.lextures.android.core.navigation.RootDestination.Courses)
                                shell.activeCourse = course
                                shell.activeCourseSection = com.lextures.android.core.navigation.CourseWorkspaceSection.Groups
                            },
                            onOpenLibrary = { openReadingLibraryOrgId = it },
                            initialLogBook = readingLogBook,
                            onConsumeInitialLogBook = { readingLogBook = null },
                            modifier = Modifier.fillMaxSize(),
                        )
                    }
                } else {
                    MoreDestinationPlaceholder(destination = destination, modifier = Modifier.fillMaxSize())
                }
            } else if (destination == com.lextures.android.core.navigation.MoreDestination.Catalog) {
                if (shell.platformFeatures.ffPublicCatalog) {
                    when {
                        openCatalogCourseSlug != null -> com.lextures.android.features.catalog.CourseLandingScreen(
                            session = session,
                            shell = shell,
                            slug = openCatalogCourseSlug!!,
                            onBack = { openCatalogCourseSlug = null },
                            modifier = Modifier.fillMaxSize(),
                        )
                        openCatalogPathSlug != null -> com.lextures.android.features.paths.PathLandingScreen(
                            session = session,
                            slug = openCatalogPathSlug!!,
                            onEnrolled = { openCatalogPathSlug = null },
                            onBack = { openCatalogPathSlug = null },
                            modifier = Modifier.fillMaxSize(),
                        )
                        else -> com.lextures.android.features.catalog.CatalogScreen(
                            session = session,
                            shell = shell,
                            onOpenCourse = { openCatalogCourseSlug = it },
                            onOpenPath = { openCatalogPathSlug = it },
                            modifier = Modifier.fillMaxSize(),
                        )
                    }
                } else {
                    MoreDestinationPlaceholder(destination = destination, modifier = Modifier.fillMaxSize())
                }
            } else if (destination == com.lextures.android.core.navigation.MoreDestination.Marketplace) {
                if (shell.platformFeatures.ffCourseMarketplace) {
                    when {
                        openMarketplaceSlug != null -> com.lextures.android.features.marketplace.MarketplaceDetailScreen(
                            session = session,
                            shell = shell,
                            slug = openMarketplaceSlug!!,
                            onBack = { openMarketplaceSlug = null },
                            modifier = Modifier.fillMaxSize(),
                        )
                        else -> com.lextures.android.features.marketplace.MarketplaceScreen(
                            session = session,
                            shell = shell,
                            onOpenCourse = { openMarketplaceSlug = it },
                            modifier = Modifier.fillMaxSize(),
                        )
                    }
                } else {
                    MoreDestinationPlaceholder(destination = destination, modifier = Modifier.fillMaxSize())
                }
            } else if (destination == com.lextures.android.core.navigation.MoreDestination.Paths) {
                if (shell.platformFeatures.ffLearningPaths) {
                    when {
                        openPathCourse != null -> com.lextures.android.features.courses.CourseDetailScreen(
                            session = session,
                            course = openPathCourse!!,
                            onBack = { openPathCourse = null },
                            modifier = Modifier.fillMaxSize(),
                        )
                        openPathLandingSlug != null -> com.lextures.android.features.paths.PathLandingScreen(
                            session = session,
                            slug = openPathLandingSlug!!,
                            onEnrolled = {
                                openPathLandingSlug = null
                                showPathsCatalog = false
                            },
                            onBack = { openPathLandingSlug = null },
                            modifier = Modifier.fillMaxSize(),
                        )
                        openPathProgress != null -> com.lextures.android.features.paths.PathRunnerScreen(
                            session = session,
                            initialPath = openPathProgress!!,
                            onOpenCourse = { openPathCourse = it },
                            onBack = { openPathProgress = null },
                            modifier = Modifier.fillMaxSize(),
                        )
                        showPathsCatalog -> com.lextures.android.features.paths.PathsCatalogScreen(
                            session = session,
                            onOpenPath = { openPathLandingSlug = it },
                            modifier = Modifier.fillMaxSize(),
                        )
                        else -> com.lextures.android.features.paths.MyPathsScreen(
                            session = session,
                            onOpenPath = { openPathProgress = it },
                            onBrowseCatalog = { showPathsCatalog = true },
                            modifier = Modifier.fillMaxSize(),
                        )
                    }
                } else {
                    MoreDestinationPlaceholder(destination = destination, modifier = Modifier.fillMaxSize())
                }
            } else if (destination == com.lextures.android.core.navigation.MoreDestination.Portfolio) {
                if (com.lextures.android.core.lms.PortfolioLogic.portfolioEnabled(shell.platformFeatures)) {
                    when {
                        openPortfolioArtifact != null && openPortfolioId != null -> {
                            com.lextures.android.features.portfolio.ArtifactDetailScreen(
                                session = session,
                                localePrefs = localePreferences,
                                portfolioId = openPortfolioId!!,
                                artifact = openPortfolioArtifact!!,
                                onArtifactUpdated = { openPortfolioArtifact = it },
                                onDeleted = {
                                    openPortfolioArtifact = null
                                },
                                onBack = { openPortfolioArtifact = null },
                                modifier = Modifier.fillMaxSize(),
                            )
                        }
                        openPortfolioId != null -> Column(modifier = Modifier.fillMaxSize().padding(16.dp)) {
                            TextButton(onClick = {
                                openPortfolioId = null
                                openPortfolioTitle = null
                            }) {
                                Text(L.text(context, localePreferences, R.string.mobile_ia_close))
                            }
                            com.lextures.android.features.portfolio.PortfolioDetailScreen(
                                session = session,
                                localePrefs = localePreferences,
                                portfolioId = openPortfolioId!!,
                                initialTitle = openPortfolioTitle.orEmpty(),
                                onOpenArtifact = { openPortfolioArtifact = it },
                                modifier = Modifier.fillMaxSize(),
                            )
                        }
                        else -> com.lextures.android.features.portfolio.PortfolioScreen(
                            session = session,
                            localePrefs = localePreferences,
                            onOpenPortfolio = { portfolio ->
                                openPortfolioId = portfolio.id
                                openPortfolioTitle = portfolio.title
                            },
                            modifier = Modifier.fillMaxSize(),
                        )
                    }
                } else {
                    MoreDestinationPlaceholder(destination = destination, modifier = Modifier.fillMaxSize())
                }
            } else if (destination == com.lextures.android.core.navigation.MoreDestination.Wallet) {
                if (com.lextures.android.core.lms.WalletLogic.walletEnabled(shell.platformFeatures)) {
                    com.lextures.android.features.wallet.WalletScreen(
                        session = session,
                        localePrefs = localePreferences,
                        platform = shell.platformFeatures,
                        modifier = Modifier.fillMaxSize(),
                    )
                } else {
                    MoreDestinationPlaceholder(destination = destination, modifier = Modifier.fillMaxSize())
                }
            } else if (destination == com.lextures.android.core.navigation.MoreDestination.Credentials) {
                if (com.lextures.android.core.lms.CredentialsLogic.credentialsEnabled(shell.platformFeatures)) {
                    when {
                        openCredential != null -> Column(modifier = Modifier.fillMaxSize().padding(16.dp)) {
                            TextButton(onClick = { openCredential = null }) {
                                Text(L.text(context, localePreferences, R.string.mobile_ia_close))
                            }
                            com.lextures.android.features.credentials.CredentialDetailScreen(
                                session = session,
                                localePrefs = localePreferences,
                                credential = openCredential!!,
                                modifier = Modifier.fillMaxSize(),
                            )
                        }
                        else -> com.lextures.android.features.credentials.CredentialsScreen(
                            session = session,
                            localePrefs = localePreferences,
                            onOpenCredential = { openCredential = it },
                            modifier = Modifier.fillMaxSize(),
                        )
                    }
                } else {
                    MoreDestinationPlaceholder(destination = destination, modifier = Modifier.fillMaxSize())
                }
            } else if (destination == com.lextures.android.core.navigation.MoreDestination.Gamification) {
                if (com.lextures.android.core.lms.GamificationLogic.gamificationEnabled(shell.platformFeatures)) {
                    com.lextures.android.features.gamification.GamificationScreen(
                        session = session,
                        localePrefs = localePreferences,
                        modifier = Modifier.fillMaxSize(),
                    )
                } else {
                    MoreDestinationPlaceholder(destination = destination, modifier = Modifier.fillMaxSize())
                }
            } else if (destination == com.lextures.android.core.navigation.MoreDestination.Advising) {
                if (com.lextures.android.core.lms.AdvisingLogic.advisingEnabled(shell.platformFeatures)) {
                    com.lextures.android.features.advising.AdvisingScreen(
                        session = session,
                        localePrefs = localePreferences,
                        modifier = Modifier.fillMaxSize(),
                    )
                } else {
                    MoreDestinationPlaceholder(destination = destination, modifier = Modifier.fillMaxSize())
                }
            } else {
                MoreDestinationPlaceholder(destination = destination, modifier = Modifier.fillMaxSize())
            }
        }
        return
    }

    if (showMoreHub) {
        Column(modifier = modifier.fillMaxSize()) {
            TextButton(
                onClick = { showMoreHub = false },
                modifier = Modifier.padding(horizontal = 8.dp),
            ) {
                Text(L.text(context, localePreferences, R.string.mobile_ia_close))
            }
            MoreHubScreen(
                shell = shell,
                onOpenDestination = { openMoreDestination = it },
                modifier = Modifier.fillMaxSize(),
            )
        }
        return
    }

    val displayName = shell.accountProfile?.resolvedDisplayName()
        ?: shell.profile?.displayName?.trim().orEmpty()
            .ifEmpty { shell.profile?.firstName ?: "Welcome" }
    val profileInitials = shell.accountProfile?.resolvedInitials()
        ?: shell.profile?.initials ?: "··"
    val avatarUrl = shell.accountProfile?.avatarUrl
    val email = shell.profile?.email ?: ""

    Column(
        modifier = modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        // Identity hero
        Box(
            modifier = Modifier
                .fillMaxWidth()
                .clip(RoundedCornerShape(24.dp))
                .background(HeroBrush),
        ) {
            Box(
                modifier = Modifier
                    .size(150.dp)
                    .offset(x = 250.dp, y = (-56).dp)
                    .clip(CircleShape)
                    .background(Color.White.copy(alpha = 0.07f)),
            )
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(vertical = 26.dp),
                horizontalAlignment = Alignment.CenterHorizontally,
                verticalArrangement = Arrangement.spacedBy(10.dp),
            ) {
                ProfileAvatar(
                    avatarUrl = avatarUrl,
                    initials = profileInitials,
                    size = 76.dp,
                    initialsBackground = Color.White.copy(alpha = 0.16f),
                    initialsForeground = Color.White,
                )
                Text(text = displayName, style = LexturesType.display(22), color = Color.White)
                if (email.isNotEmpty()) {
                    Text(text = email, fontSize = 13.sp, color = Color.White.copy(alpha = 0.8f))
                }
            }
        }

        if (shell.iaRedesignEnabled && shell.roleSnapshot.availableContexts.size > 1) {
            LmsCard {
                Text(
                    text = L.text(context, localePreferences, R.string.mobile_ia_context_title),
                    style = LexturesType.display(17),
                    color = textPrimary(),
                )
                shell.roleSnapshot.availableContexts.forEach { roleContext ->
                    val selected = shell.activeRoleContext == roleContext
                    TextButton(onClick = { shell.setRoleContext(roleContext) { } }) {
                        Text(
                            text = when (roleContext) {
                                com.lextures.android.core.navigation.MobileRoleContext.Learning ->
                                    L.text(context, localePreferences, R.string.mobile_ia_context_learning)
                                com.lextures.android.core.navigation.MobileRoleContext.Teaching ->
                                    L.text(context, localePreferences, R.string.mobile_ia_context_teaching)
                                com.lextures.android.core.navigation.MobileRoleContext.Parent ->
                                    L.text(context, localePreferences, R.string.mobile_ia_context_parent)
                            },
                            fontWeight = if (selected) FontWeight.Bold else FontWeight.Normal,
                        )
                    }
                }
            }
        }

        if (shell.iaRedesignEnabled &&
            com.lextures.android.core.navigation.MobileDestinations.moreDestinations(
                shell.activeRoleContext,
                shell.platformFeatures,
            ).isNotEmpty()
        ) {
            LmsCard {
                SettingsNavRow(
                    icon = Icons.Default.Apps,
                    title = L.text(context, localePreferences, R.string.mobile_ia_more_title),
                    subtitle = L.text(context, localePreferences, R.string.mobile_ia_more_search),
                    onClick = { showMoreHub = true },
                )
            }
        }

        // Personal: edit profile + accommodations
        LmsCard {
            SettingsNavRow(
                icon = Icons.Default.Person,
                title = L.text(R.string.mobile_editProfile_title),
                subtitle = L.text(R.string.mobile_editProfile_subtitle),
                onClick = { showEditProfile = true },
            )
            SettingsNavRow(
                icon = Icons.Default.VerifiedUser,
                title = L.text(R.string.mobile_accommodations_title),
                subtitle = L.text(R.string.mobile_accommodations_subtitle),
                onClick = { showAccommodations = true },
                modifier = Modifier.padding(top = 4.dp),
            )
        }

        if (shell.profileDepthEnabled && (personalDetailsVisible || researchVisible)) {
            LmsCard {
                if (personalDetailsVisible) {
                    SettingsNavRow(
                        icon = Icons.AutoMirrored.Filled.FormatListBulleted,
                        title = L.text(R.string.mobile_profileDepth_personalDetails_title),
                        subtitle = L.text(R.string.mobile_profileDepth_personalDetails_subtitle),
                        onClick = { showPersonalDetails = true },
                    )
                }
                if (researchVisible) {
                    SettingsNavRow(
                        icon = Icons.Default.Security,
                        title = L.text(R.string.mobile_profileDepth_research_title),
                        subtitle = L.text(R.string.mobile_profileDepth_research_subtitle),
                        onClick = { showResearchStudies = true },
                        modifier = if (personalDetailsVisible) Modifier.padding(top = 4.dp) else Modifier,
                    )
                }
            }
        }

        if (com.lextures.android.core.lms.AccountIntegrationsLogic.integrationsEnabled(shell.platformFeatures)) {
            LmsCard {
                SettingsNavRow(
                    icon = Icons.Default.Link,
                    title = L.text(R.string.mobile_integrations_title),
                    subtitle = L.text(R.string.mobile_integrations_entry_subtitle),
                    onClick = { showIntegrations = true },
                )
            }
        }

        if (com.lextures.android.core.lms.ArchivedCoursesAdminLogic.shouldShowEntry(
                shell.platformFeatures,
                shell.permissions,
            )
        ) {
            LmsCard {
                SettingsNavRow(
                    icon = Icons.Filled.Archive,
                    title = L.text(R.string.mobile_admin_archivedCourses_title),
                    subtitle = L.text(R.string.mobile_admin_archivedCourses_entry_subtitle),
                    onClick = { showArchivedCoursesAdmin = true },
                )
            }
        }

        if (com.lextures.android.core.lms.RolesPermissionsAdminLogic.shouldShowEntry(
                shell.platformFeatures,
                shell.permissions,
            )
        ) {
            LmsCard {
                SettingsNavRow(
                    icon = Icons.Default.Person,
                    title = L.text(R.string.mobile_admin_roles_title),
                    subtitle = L.text(R.string.mobile_admin_roles_entry_subtitle),
                    onClick = { showRolesPermissionsAdmin = true },
                )
            }
        }

        if (com.lextures.android.core.lms.PeopleAdminLogic.shouldShowEntry(
                shell.platformFeatures,
                shell.permissions,
            )
        ) {
            LmsCard {
                SettingsNavRow(
                    icon = Icons.Default.Person,
                    title = L.text(R.string.mobile_admin_people_title),
                    subtitle = L.text(R.string.mobile_admin_people_entry_subtitle),
                    onClick = { showPeopleAdmin = true },
                )
            }
        }

        if (com.lextures.android.core.lms.OrgStructureAdminLogic.shouldShowEntry(
                shell.platformFeatures,
                shell.permissions,
            )
        ) {
            LmsCard {
                SettingsNavRow(
                    icon = Icons.Filled.Apartment,
                    title = L.text(R.string.mobile_admin_orgStructure_title),
                    subtitle = L.text(R.string.mobile_admin_orgStructure_entry_subtitle),
                    onClick = { showOrgStructureAdmin = true },
                )
            }
        }

        if (com.lextures.android.core.lms.OrgBrandingAdminLogic.shouldShowEntry(
                shell.platformFeatures,
                shell.permissions,
            )
        ) {
            LmsCard {
                SettingsNavRow(
                    icon = Icons.Filled.Brush,
                    title = L.text(R.string.mobile_admin_orgBranding_title),
                    subtitle = L.text(R.string.mobile_admin_orgBranding_entry_subtitle),
                    onClick = { showOrgBrandingAdmin = true },
                )
            }
        }

        if (com.lextures.android.core.lms.PlatformSettingsAdminLogic.shouldShowEntry(
                shell.platformFeatures,
                shell.permissions,
            )
        ) {
            LmsCard {
                SettingsNavRow(
                    icon = Icons.Default.Settings,
                    title = L.text(R.string.mobile_admin_platform_title),
                    subtitle = L.text(R.string.mobile_admin_platform_entry_subtitle),
                    onClick = { showPlatformSettingsAdmin = true },
                )
            }
        }

        if (com.lextures.android.core.lms.LearnerProfileLogic.learnerProfileEnabled(shell.platformFeatures)) {
            LmsCard {
                SettingsNavRow(
                    icon = Icons.Default.Person,
                    title = L.text(R.string.mobile_learnerProfile_title),
                    subtitle = L.text(R.string.mobile_learnerProfile_entry_subtitle),
                    onClick = { showLearnerProfile = true },
                )
            }
        }

        if (com.lextures.android.core.lms.FeedbackLogic.feedbackEnabled(shell.platformFeatures)) {
            LmsCard {
                SettingsNavRow(
                    icon = Icons.Default.Chat,
                    title = L.text(R.string.mobile_feedback_entry),
                    subtitle = L.text(R.string.mobile_feedback_entrySubtitle),
                    onClick = { showShareFeedback = true },
                )
            }
        }

        feedbackSuccessMessage?.let { message ->
            LmsCard {
                Text(
                    text = message,
                    fontSize = 14.sp,
                    fontWeight = FontWeight.Medium,
                    color = accentColor(),
                )
            }
        }

        if (pendingCount > 0) {
            LmsCard {
                Text(text = "Pending sync", style = LexturesType.display(17), color = textPrimary())
                Text(
                    text = "$pendingCount change${if (pendingCount == 1) "" else "s"} waiting to upload",
                    fontSize = 12.sp,
                    color = textSecondary(),
                )
                outboxItems.filter {
                    val status = it.outboxStatus()
                    status == OutboxStatus.Queued || status == OutboxStatus.Failed || status == OutboxStatus.Conflict
                }.forEach { item ->
                    Column(modifier = Modifier.padding(top = 8.dp)) {
                        Text(text = item.label, fontSize = 14.sp, fontWeight = FontWeight.Medium, color = textPrimary())
                        OutboxStatusChip(status = item.outboxStatus())
                        if (item.outboxStatus() == OutboxStatus.Failed || item.outboxStatus() == OutboxStatus.Conflict) {
                            TextButton(onClick = {
                                scope.launch {
                                    offline.retryOutboxItem(item.id, accessToken)
                                }
                            }) {
                                Text("Retry")
                            }
                        }
                    }
                }
            }
        }

        // Appearance (theme override)
        LmsCard {
            Text(
                text = L.text(R.string.mobile_settings_appearance),
                style = LexturesType.display(17),
                color = textPrimary(),
            )
            Text(
                text = L.text(R.string.mobile_settings_appearance_description),
                fontSize = 12.sp,
                color = textSecondary(),
            )
            Row(
                modifier = Modifier.fillMaxWidth().padding(top = 8.dp),
                horizontalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                val options = listOf(
                    ThemeAppearance.SYSTEM to R.string.mobile_settings_theme_system,
                    ThemeAppearance.LIGHT to R.string.mobile_settings_theme_light,
                    ThemeAppearance.DARK to R.string.mobile_settings_theme_dark,
                )
                options.forEach { (appearance, labelRes) ->
                    val selected = themePreference.appearance == appearance
                    Box(
                        modifier = Modifier
                            .weight(1f)
                            .clip(RoundedCornerShape(10.dp))
                            .background(
                                if (selected) {
                                    LexturesColors.Primary.copy(alpha = 0.16f)
                                } else {
                                    Color.Transparent
                                },
                            )
                            .clickable { themePreference.update(appearance) }
                            .padding(vertical = 10.dp),
                        contentAlignment = Alignment.Center,
                    ) {
                        Text(
                            text = L.text(labelRes),
                            fontSize = 14.sp,
                            fontWeight = if (selected) FontWeight.SemiBold else FontWeight.Normal,
                            color = if (selected) accentColor() else textSecondary(),
                        )
                    }
                }
            }
        }

        LmsCard {
            Text(text = "Offline storage", style = LexturesType.display(17), color = textPrimary())
            InfoRow(
                Icons.Default.Storage,
                "Cache size",
                android.text.format.Formatter.formatFileSize(context, storageBytes),
            )
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .clip(RoundedCornerShape(10.dp))
                    .clickable { confirmingClearCache = true }
                    .padding(vertical = 8.dp),
                horizontalArrangement = Arrangement.spacedBy(12.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Icon(Icons.Default.Delete, contentDescription = null, tint = LexturesColors.Error)
                Text(
                    text = "Clear cached data",
                    fontSize = 14.sp,
                    fontWeight = FontWeight.SemiBold,
                    color = LexturesColors.Error,
                )
            }
            if (shell?.universalSearchEnabled == true) {
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .clip(RoundedCornerShape(10.dp))
                        .clickable { confirmingClearSearchHistory = true }
                        .padding(vertical = 8.dp),
                    horizontalArrangement = Arrangement.spacedBy(12.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Icon(Icons.Default.Search, contentDescription = null, tint = LexturesColors.Error)
                    Text(
                        text = L.text(context, localePreferences, R.string.mobile_search_clearHistory),
                        fontSize = 14.sp,
                        fontWeight = FontWeight.SemiBold,
                        color = LexturesColors.Error,
                    )
                }
            }
        }

        LmsCard {
            Text(text = L.text(R.string.common_locale_label), style = LexturesType.display(17), color = textPrimary())
            Text(text = L.text(R.string.common_locale_description), fontSize = 12.sp, color = textSecondary())
            ExposedDropdownMenuBox(
                expanded = localeExpanded,
                onExpandedChange = { localeExpanded = it },
            ) {
                val selected = LocalePreferences.localeOptions.firstOrNull { it.tag == localePreferences.localeTag }
                OutlinedTextField(
                    value = selected?.let {
                        if (it.tag == LocalePreferences.SYSTEM_TAG) L.text(R.string.common_locale_systemDefault) else it.label
                    }.orEmpty(),
                    onValueChange = {},
                    readOnly = true,
                    trailingIcon = { ExposedDropdownMenuDefaults.TrailingIcon(expanded = localeExpanded) },
                    modifier = Modifier
                        .menuAnchor()
                        .fillMaxWidth(),
                )
                ExposedDropdownMenu(
                    expanded = localeExpanded,
                    onDismissRequest = { localeExpanded = false },
                ) {
                    LocalePreferences.localeOptions.forEach { option ->
                        DropdownMenuItem(
                            text = {
                                Text(
                                    if (option.tag == LocalePreferences.SYSTEM_TAG) {
                                        L.text(R.string.common_locale_systemDefault)
                                    } else {
                                        option.label
                                    },
                                )
                            },
                            onClick = {
                                localeExpanded = false
                                val previous = localePreferences.localeTag
                                localePreferences.updateLocaleTag(option.tag)
                                val token = accessToken
                                if (token != null) {
                                    scope.launch {
                                        try {
                                            val apiTag = if (option.tag == LocalePreferences.SYSTEM_TAG) {
                                                java.util.Locale.getDefault().toLanguageTag()
                                            } else {
                                                option.tag
                                            }
                                            LocaleApi.saveLocale(apiTag, token)
                                        } catch (_: Exception) {
                                            localePreferences.updateLocaleTag(previous)
                                            localeError = L.text(context, localePreferences, R.string.common_locale_saveError)
                                        }
                                    }
                                }
                            },
                        )
                    }
                }
            }
            localeError?.let {
                Text(text = it, fontSize = 12.sp, color = LexturesColors.Error)
            }
        }

        if (uiModeStore.featureEnabled) {
            LmsCard {
                Text(text = L.text(R.string.mobile_uiMode_title), style = LexturesType.display(17), color = textPrimary())
                Text(
                    text = L.text(R.string.mobile_uiMode_description),
                    fontSize = 12.sp,
                    color = textSecondary(),
                )
                if (uiModeStore.hasAdminOverride) {
                    Text(
                        text = L.text(R.string.mobile_uiMode_adminOverride),
                        fontSize = 12.sp,
                        color = textSecondary(),
                    )
                }
                var expanded by remember { mutableStateOf(false) }
                ExposedDropdownMenuBox(expanded = expanded, onExpandedChange = { expanded = it }) {
                    OutlinedTextField(
                        value = L.text(uiModeStore.localPreference.labelRes),
                        onValueChange = {},
                        readOnly = true,
                        enabled = !uiModeStore.hasAdminOverride,
                        trailingIcon = { ExposedDropdownMenuDefaults.TrailingIcon(expanded) },
                        modifier = Modifier
                            .fillMaxWidth()
                            .menuAnchor(),
                    )
                    ExposedDropdownMenu(expanded = expanded, onDismissRequest = { expanded = false }) {
                        com.lextures.android.core.design.UIModePreference.entries.forEach { option ->
                            DropdownMenuItem(
                                text = { Text(L.text(option.labelRes)) },
                                onClick = {
                                    expanded = false
                                    uiModeStore.updateLocalPreference(option)
                                },
                            )
                        }
                    }
                }
            }
        }

        LmsCard {
            Text(text = L.text(R.string.mobile_profile_accessibility), style = LexturesType.display(17), color = textPrimary())
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Column(modifier = Modifier.weight(1f)) {
                    Text(
                        text = "Dyslexia-friendly display",
                        fontSize = 14.sp,
                        fontWeight = FontWeight.Medium,
                        color = textPrimary(),
                    )
                    Text(
                        text = "Rounded type, extra spacing, and relaxed line height app-wide.",
                        fontSize = 12.sp,
                        color = textSecondary(),
                    )
                }
                Switch(
                    checked = accessibilityState.dyslexiaDisplayEnabled,
                    onCheckedChange = accessibilityState::updateDyslexiaDisplayEnabled,
                    colors = SwitchDefaults.colors(checkedTrackColor = LexturesColors.Primary),
                )
            }
        }

        LmsCard {
            Text(
                text = L.text(context, localePreferences, R.string.mobile_profile_security),
                style = LexturesType.display(17),
                color = textPrimary(),
            )
            if (biometricGate.canEnableBiometrics) {
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(top = 8.dp),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Column(modifier = Modifier.weight(1f)) {
                        Text(
                            text = localePreferences.localizedContext(context).getString(
                                R.string.mobile_biometric_toggle,
                                biometricGate.biometryLabel(context),
                            ),
                            fontSize = 14.sp,
                            fontWeight = FontWeight.Medium,
                            color = textPrimary(),
                        )
                        Text(
                            text = L.text(context, localePreferences, R.string.mobile_biometric_toggleDescription),
                            fontSize = 12.sp,
                            color = textSecondary(),
                        )
                    }
                    Switch(
                        checked = biometricGate.isEnabled,
                        onCheckedChange = { biometricGate.isEnabled = it },
                        colors = SwitchDefaults.colors(checkedTrackColor = LexturesColors.Primary),
                    )
                }
            }
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(top = 12.dp)
                    .clickable { showDeviceSessions = true },
                horizontalArrangement = Arrangement.spacedBy(12.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Box(
                    modifier = Modifier
                        .size(32.dp)
                        .clip(RoundedCornerShape(10.dp))
                        .background(LexturesColors.BrandTeal.copy(alpha = if (isDarkTheme()) 0.18f else 0.14f)),
                    contentAlignment = Alignment.Center,
                ) {
                    Icon(Icons.Default.Dns, contentDescription = null, tint = accentColor(), modifier = Modifier.size(16.dp))
                }
                Column(modifier = Modifier.weight(1f)) {
                    Text(
                        text = L.text(context, localePreferences, R.string.mobile_sessions_title),
                        fontSize = 14.sp,
                        fontWeight = FontWeight.SemiBold,
                        color = textPrimary(),
                    )
                    Text(
                        text = L.text(context, localePreferences, R.string.mobile_sessions_profileHint),
                        fontSize = 12.sp,
                        color = textSecondary(),
                    )
                }
                Icon(Icons.AutoMirrored.Filled.KeyboardArrowRight, contentDescription = null, tint = textSecondary())
            }
        }

        // Account
        LmsCard {
            Text(text = "Account", style = LexturesType.display(17), color = textPrimary())
            InfoRow(Icons.Default.Person, "Display name", displayName)
            InfoRow(Icons.Default.Email, "Email", email.ifEmpty { "—" })
            if (com.lextures.android.core.lms.BillingLogic.billingEnabled(shell.platformFeatures)) {
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .clickable { showBilling = true }
                        .padding(vertical = 8.dp),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Row(horizontalArrangement = Arrangement.spacedBy(12.dp), verticalAlignment = Alignment.CenterVertically) {
                        Icon(Icons.Default.CreditCard, contentDescription = null, tint = accentColor())
                        Text(
                            L.text(context, localePreferences, R.string.mobile_billing_title),
                            fontSize = 14.sp,
                            fontWeight = FontWeight.SemiBold,
                            color = textPrimary(),
                        )
                    }
                    Icon(Icons.AutoMirrored.Filled.KeyboardArrowRight, contentDescription = null, tint = textSecondary())
                }
            }
        }

        // Notifications
        LmsCard(onClick = { showNotifications = true }) {
            Row(
                horizontalArrangement = Arrangement.spacedBy(12.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Box(
                    modifier = Modifier
                        .size(32.dp)
                        .clip(RoundedCornerShape(10.dp))
                        .background(LexturesColors.BrandTeal.copy(alpha = if (isDarkTheme()) 0.18f else 0.14f)),
                    contentAlignment = Alignment.Center,
                ) {
                    Icon(
                        Icons.Default.Notifications,
                        contentDescription = null,
                        tint = accentColor(),
                        modifier = Modifier.size(16.dp),
                    )
                }
                Column(modifier = Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
                    Text(
                        text = "Notifications",
                        fontSize = 15.sp,
                        fontWeight = FontWeight.SemiBold,
                        color = textPrimary(),
                    )
                    Text(
                        text = if (shell.unreadNotifications > 0) {
                            "${shell.unreadNotifications} unread"
                        } else {
                            "You're all caught up"
                        },
                        fontSize = 12.sp,
                        color = textSecondary(),
                    )
                }
                if (shell.unreadNotifications > 0) {
                    Text(
                        text = "${shell.unreadNotifications}",
                        fontSize = 12.sp,
                        fontWeight = FontWeight.Bold,
                        color = Color.White,
                        modifier = Modifier
                            .clip(RoundedCornerShape(50))
                            .background(LexturesColors.Coral)
                            .padding(horizontal = 8.dp, vertical = 3.dp),
                    )
                }
                Icon(
                    Icons.AutoMirrored.Filled.KeyboardArrowRight,
                    contentDescription = null,
                    tint = textSecondary().copy(alpha = 0.6f),
                    modifier = Modifier.size(16.dp),
                )
            }
        }

        // Privacy & trust
        LmsCard {
            Text(
                text = L.text(R.string.mobile_settings_privacyTrust),
                style = LexturesType.display(17),
                color = textPrimary(),
            )
            LegalLinkRow(
                title = L.text(R.string.mobile_settings_privacyCenter),
                path = "/privacy",
            )
            LegalLinkRow(
                title = L.text(R.string.mobile_settings_trustCenter),
                path = "/security",
            )
            LegalLinkRow(
                title = L.text(R.string.mobile_settings_accessibilityStatement),
                path = "/accessibility",
            )
        }

        // About
        LmsCard {
            Text(text = "About", style = LexturesType.display(17), color = textPrimary())
            InfoRow(Icons.Default.Apps, "Version", BuildConfig.VERSION_NAME)
            InfoRow(Icons.Default.Dns, "Server", AppConfiguration.apiBaseUrl)
        }

        // Sign out
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .clip(RoundedCornerShape(14.dp))
                .background(LexturesColors.Error.copy(alpha = 0.09f))
                .clickable { confirmingSignOut = true }
                .padding(vertical = 14.dp),
            horizontalArrangement = Arrangement.Center,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Icon(
                Icons.AutoMirrored.Filled.Logout,
                contentDescription = null,
                tint = LexturesColors.Error,
                modifier = Modifier.size(17.dp),
            )
            Box(modifier = Modifier.width(8.dp))
            Text(
                text = "Sign out",
                fontSize = 15.sp,
                fontWeight = FontWeight.SemiBold,
                color = LexturesColors.Error,
            )
        }
    }

    androidx.compose.runtime.LaunchedEffect(shell.pendingMoreDestination) {
        shell.consumePendingMoreDestination()?.let { showMoreHub = true }
    }

    androidx.compose.runtime.LaunchedEffect(shell.pendingBilling) {
        if (shell.consumePendingBilling()) {
            showBilling = true
        }
    }

    if (confirmingClearSearchHistory) {
        AlertDialog(
            onDismissRequest = { confirmingClearSearchHistory = false },
            title = { Text(L.text(context, localePreferences, R.string.mobile_search_clearHistoryConfirm)) },
            text = { Text(L.text(context, localePreferences, R.string.mobile_search_clearHistoryMessage)) },
            confirmButton = {
                TextButton(onClick = {
                    confirmingClearSearchHistory = false
                    SearchRecentsStore.clearAll(context)
                }) {
                    Text(L.text(context, localePreferences, R.string.mobile_search_clearHistory), color = LexturesColors.Error)
                }
            },
            dismissButton = {
                TextButton(onClick = { confirmingClearSearchHistory = false }) { Text("Cancel") }
            },
        )
    }

    if (showShareFeedback) {
        com.lextures.android.features.feedback.ShareFeedbackSheet(
            session = session,
            localePrefs = localePreferences,
            isOnline = isOnline,
            onDismiss = { showShareFeedback = false },
            onSuccess = {
                feedbackSuccessMessage = L.text(context, localePreferences, R.string.mobile_feedback_success)
            },
        )
    }

    if (confirmingClearCache) {
        AlertDialog(
            onDismissRequest = { confirmingClearCache = false },
            title = { Text("Clear offline storage?") },
            text = {
                Text("Removes cached reads and downloads from this device. Queued changes are kept until they sync.")
            },
            confirmButton = {
                TextButton(onClick = {
                    confirmingClearCache = false
                    offline.clearStorage()
                }) {
                    Text("Clear cache", color = LexturesColors.Error)
                }
            },
            dismissButton = {
                TextButton(onClick = { confirmingClearCache = false }) { Text("Cancel") }
            },
        )
    }

    if (confirmingSignOut) {
        AlertDialog(
            onDismissRequest = { confirmingSignOut = false },
            title = { Text("Sign out of Lextures?") },
            confirmButton = {
                TextButton(onClick = {
                    confirmingSignOut = false
                    session.signOut()
                }) {
                    Text("Sign out", color = LexturesColors.Error)
                }
            },
            dismissButton = {
                TextButton(onClick = { confirmingSignOut = false }) { Text("Cancel") }
            },
        )
    }
}

@Composable
private fun SettingsNavRow(
    icon: ImageVector,
    title: String,
    subtitle: String,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
) {
    Row(
        modifier = modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(10.dp))
            .clickable(onClick = onClick)
            .padding(vertical = 8.dp),
        horizontalArrangement = Arrangement.spacedBy(12.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Box(
            modifier = Modifier
                .size(32.dp)
                .clip(RoundedCornerShape(10.dp))
                .background(LexturesColors.BrandTeal.copy(alpha = if (isDarkTheme()) 0.18f else 0.14f)),
            contentAlignment = Alignment.Center,
        ) {
            Icon(icon, contentDescription = null, tint = accentColor(), modifier = Modifier.size(16.dp))
        }
        Column(modifier = Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
            Text(text = title, fontSize = 14.sp, fontWeight = FontWeight.SemiBold, color = textPrimary())
            Text(text = subtitle, fontSize = 12.sp, color = textSecondary())
        }
        Icon(
            Icons.AutoMirrored.Filled.KeyboardArrowRight,
            contentDescription = null,
            tint = textSecondary().copy(alpha = 0.6f),
        )
    }
}

@Composable
private fun LegalLinkRow(title: String, path: String) {
    val context = LocalContext.current
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(10.dp))
            .clickable {
                val uri = android.net.Uri.parse(AppConfiguration.webUrl(path))
                context.startActivity(android.content.Intent(android.content.Intent.ACTION_VIEW, uri))
            }
            .padding(vertical = 10.dp),
        horizontalArrangement = Arrangement.spacedBy(12.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(Icons.Default.OpenInNew, contentDescription = null, tint = accentColor(), modifier = Modifier.size(17.dp))
        Text(
            text = title,
            fontSize = 14.sp,
            fontWeight = FontWeight.Medium,
            color = textPrimary(),
            modifier = Modifier.weight(1f),
        )
    }
}

@Composable
private fun InfoRow(icon: ImageVector, label: String, value: String) {
    Row(
        horizontalArrangement = Arrangement.spacedBy(12.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(icon, contentDescription = null, tint = accentColor(), modifier = Modifier.size(17.dp))
        Column(verticalArrangement = Arrangement.spacedBy(2.dp)) {
            Text(text = label, fontSize = 11.sp, color = textSecondary())
            Text(
                text = value,
                fontSize = 14.sp,
                fontWeight = FontWeight.Medium,
                color = textPrimary(),
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
        }
    }
}
