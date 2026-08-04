# App Store Review Rejection — 2026-08-04

**Submission ID:** `84c8078f-15ed-4909-9121-5826a6c36bcc`  
**Device:** iPad Air 11-inch (M3) · **Version reviewed:** 1.0 (2)

Use this guide when replying in App Store Connect and when uploading the next binary.

---

## Summary of resolution

| Guideline | Resolution type | Action |
|-----------|-----------------|--------|
| **5.1.2(i)** Tracking / ATT | App Store Connect (not ATT code) | Privacy labels incorrectly marked *Used to Track You*. Uncheck tracking. App does **not** track. |
| **2.3.10** Android metadata | App Store Connect description | Remove Android / Play Store wording from listing text. |
| **5.1.1(v)** Account deletion | Already in app; UX clarified | Profile → **Delete account** section; permanent erasure via `DELETE /api/v1/settings/account`. Record a device screen recording for Review Notes. |
| **2.1(b)** Business model | Reply in App Store Connect | Answer the five questions below (reader-app / institutional LMS model). |

**Binary changes in this repo for the resubmission:**

- Clearer **Delete account** danger-zone card on Profile (accessibility id `profile.deleteAccount`).
- Privacy manifest `PrivacyInfo.xcprivacy` with `NSPrivacyTracking = false`.
- Marketplace copy no longer mentions Google Play / Android.
- Marketing version **1.0.1**, build **3** (update in Xcode if you manage versions there).

---

## 1. Guideline 5.1.2(i) — Privacy / Tracking (do this first)

### What Apple thinks is wrong

App Privacy in App Store Connect says the app collects **Email Address**, **Product Interaction**, and **User ID** **for tracking**. There is no App Tracking Transparency (ATT) prompt.

### Correct product behavior

Lextures does **not** track under Apple’s definition:

- No advertising SDKs, IDFA, or ad networks.
- No sharing user data with data brokers.
- No linking app data with third-party data for advertising or advertising measurement.
- First-party account email / user id / product usage (courses, grades, LMS analytics) are for **app functionality**, **account management**, and **analytics** only — not tracking.

### Fix (Account Holder or Admin)

1. App Store Connect → your app → **App Privacy**.
2. For each data type (Email, Product Interaction, User ID, etc.):
   - If collected: keep **Collected**.
   - **Uncheck** “Used to Track You” / purposes that imply tracking.
   - Typical purposes for an LMS: **App Functionality**, **Account Management**, **Analytics** (first-party only), **Product Personalization** if applicable.
3. Confirm **Data Linked to the User** where true (account data is linked).
4. Confirm **Data Used to Track You** = **No** for all types.
5. Publish the privacy questionnaire update **before** or with the next submission.

### Do **not** add ATT

Implementing `ATTrackingManager` without actually tracking confuses users and does not match the product. Only add ATT if you later introduce cross-app advertising tracking.

### Reply template (optional)

> Lextures does not track users as defined by Apple. We do not use advertising identifiers, ad networks, or data brokers, and we do not link app data with third-party data for advertising. Email Address, User ID, and Product Interaction are used only for account management, delivering the LMS experience, and first-party product analytics. We have updated App Privacy in App Store Connect so these data types are no longer marked “Used to Track You.” ATT is not applicable.

---

## 2. Guideline 2.3.10 — Accurate Metadata (Android references)

### Fix

App Store Connect → **App Information** / **What’s New** / **Description** (all locales):

- Remove mentions of **Android**, **Google Play**, or dual-platform marketing that is not about the iOS experience.
- Describe only what the **iOS/iPadOS app** does.

### Suggested description (edit to taste)

> Lextures is a learning platform for students, instructors, parents, and schools. Sign in with your school account or personal account to access courses, assignments, grades, discussions, notebooks, and study tools on iPhone and iPad.
>
> Features include:
> • Course modules, assignments, and quizzes  
> • Grades, planner, and notifications  
> • Collaboration boards and messaging  
> • Parent view when your school enables it  
> • Optional marketplace courses for individual learners  
>
> Your school or organization provides access for institutional accounts. Individual learners can also create an account for self-paced study where available.

### In-app copy

In-app marketplace hint no longer references Play Store (shared mobile locales).

---

## 3. Guideline 5.1.1(v) — Account deletion

### Already implemented

| Layer | Location |
|-------|----------|
| API | `DELETE /api/v1/settings/account` — permanent anonymization + sign-out everywhere |
| iOS | **Profile** tab → **Delete account** card (with Account / billing) → confirm |
| Web | Settings → Account → Delete account |

Deletion is **permanent** (not soft deactivate only). Profile data is anonymized; legally required non-identifying academic records may remain as tombstones.

### Path for App Review (Review Notes)

```
1. Sign in with the demo account (or create a new account via Sign Up).
2. Open the Profile tab (person icon / Profile).
3. Scroll to the "Delete account" card (directly under Account / billing).
4. Tap "Delete account".
5. Confirm in the dialog. Account is erased and the app returns to sign-in.
```

Accessibility identifier: `profile.deleteAccount`.

### Screen recording (required for resubmit)

Capture on a **physical device**:

1. Create account **or** sign in with demo account.  
2. Navigate to Profile → Delete account.  
3. Full flow through confirmation.  

Attach the recording in **App Review Information → Notes** (or link), and keep a short note with the path above.

### Reply template

> Account deletion is available in the app. After signing in, open the Profile tab and use the “Delete account” card under Account. Confirming permanently anonymizes the account and signs the user out on all devices via DELETE /api/v1/settings/account. A screen recording of the flow is included in Review Notes.

---

## 4. Guideline 2.1(b) — Business model questions

Copy/adapt this reply into App Store Connect.

### 1. Who are the users that will use the paid content, subscriptions, features, and services in the app?

- **Institutional users (primary):** Students, instructors, parents, and staff at schools, districts, and universities whose organization licenses or self-hosts Lextures. Access is granted by the institution (roster/SSO); they do not buy the LMS seat inside the iOS app.
- **Individual / homeschool learners:** People who create a personal account for self-paced courses and optional marketplace enrollments.
- **Organization admins:** Configure the platform and billing outside the student mobile experience.

### 2. Where can users purchase the content, subscriptions, features, and services that can be accessed in the app?

- **Institutional access:** Purchased/contracted **outside the app** (school/district/university agreement, invoice, or self-hosted deployment). Users then sign in with school credentials or accounts provisioned by the institution.
- **Individual hosted access / course purchases:** On the **website** (e.g. self.lextures.com) via browser checkout (Stripe). The iOS app may open an external browser for digital course checkout; it does **not** use Apple In-App Purchase for those digital goods.
- **No IAP:** The app does not unlock digital content via StoreKit IAP.

### 3. What specific types of previously purchased content, subscriptions, features, and services can a user access in the app?

- Course enrollments and content granted by their school or by a prior web purchase/claim.
- Platform features enabled for their organization (grades, discussions, live meetings, parent context, etc.).
- Marketplace courses already owned or claimed on the account.
- Subscription/entitlement state established on the web (where applicable), visible under Profile → Billing / Purchases when those features are enabled.

### 4. What paid content, subscriptions, or features are unlocked within the app that do not use In-App Purchase?

- **None unlocked only inside the app via a non-IAP payment UI that grants digital goods without leaving the app.** Paid digital course checkout is completed **in the browser** (Stripe), after which content is available on all clients including iOS.
- Institutional seats and org-level features are unlocked by **organization provisioning**, not by an in-app store.
- Optional individual hosting plans (e.g. monthly access marketed on the website) are purchased on the **web**, not via IAP.

### 5. What is the maximum number of users that can participate in your app's live, real-time services?

- Live features (live meetings, live quizzes, real-time collaboration/whiteboards where enabled) are **class-scale education tools**, not large public broadcasts.
- There is **no hard product marketing cap** like a paid “max concurrent seats” SKU inside the app; practical limits follow the host service and institution configuration (typical classroom / course cohort sizes, often on the order of tens to a few hundred participants depending on deployment).
- The app is not a ticketed live-event marketplace; live sessions are for enrolled course participants and staff.

---

## 5. Resubmission checklist

- [ ] Update **App Privacy** (no “Used to Track You”).
- [ ] Remove **Android** / **Play** from App Store description (all locales).
- [ ] Paste **business model** answers into the rejection reply.
- [ ] Put **account deletion path** + attach **screen recording** in Review Notes.
- [ ] Archive & upload binary **1.0.1 (3)** (or next version) including Profile danger-zone + PrivacyInfo.
- [ ] Demo account: deletable **or** use a fresh signup for the deletion recording (do not use a system/protected account).
- [ ] Confirm production API serves `DELETE /api/v1/settings/account`.

---

## 6. Optional Review Notes block (paste)

```
ACCOUNT DELETION
Sign in → Profile tab → “Delete account” card (under Account) → confirm.
API: DELETE /api/v1/settings/account (permanent anonymization). Screen recording attached.

TRACKING / PRIVACY
We do not track users (no ads, no data brokers, no IDFA). App Privacy labels updated so
Email / Product Interaction / User ID are not marked Used to Track You. No ATT prompt
because we do not track.

BUSINESS MODEL
Institutional LMS: schools provision access outside the app. Individual digital courses
and hosted plans are purchased on the website (Stripe browser checkout), not via IAP.
The iOS app reads existing entitlements and course access. Live features are classroom-
scale, not paid public broadcasts.

METADATA
App description revised to remove Android references.
```
