# App Store Review — Guideline 3.1.2(c) Subscriptions

Use this when Apple rejects for missing Privacy Policy / Terms of Use (EULA) links on the auto-renewable subscription purchase flow.

## What Review asked for

Apps offering auto-renewable subscriptions must include **in the app**:

- Title of the auto-renewing subscription
- Length of the subscription
- Price, and price per unit if appropriate
- Functional links to the **Privacy Policy** and **Terms of Use (EULA)**

App Store Connect metadata must also include those URLs (Privacy Policy field + EULA / App Description).

## What we shipped

The Homeschool subscribe paywall (`HomeschoolSubscribePaywallView`) sells the outcome first, then shows two plans (annual default) with a **sticky legal footer**:

| Item | Implementation |
|------|----------------|
| Title | StoreKit `displayName` (Monthly Access / Annual Access) |
| Length | Subscription period label |
| Price | StoreKit `displayPrice` |
| Price per unit | Weekly equivalent under each plan |
| Privacy Policy | Footer control `paywall.privacyPolicy` → https://lextures.com/privacy (Safari) |
| Terms of Use (EULA) | Footer control `paywall.termsOfUse` → https://lextures.com/terms (Safari) |
| Auto-renew | Footer names selected title, price, and period |

Links use `UIApplication.shared.open` so they are not classified as in-app deep links.

## Screen recording (attach in Review Notes)

```
1. Homeschool → sign in with the demo account (no active subscription).
2. Wait ~5 seconds for the subscribe paywall, or Profile → Billing → Subscribe.
3. Footer already shows Privacy Policy and Terms of Use (EULA).
4. Tap Privacy Policy → Safari opens https://lextures.com/privacy.
5. Return to the app. Tap Terms of Use (EULA) → Safari opens https://lextures.com/terms.
6. Confirm Monthly Access and Annual Access show title, length, price, and weekly price.
```

## Reply to paste in Resolution Center

> The subscribe screen now includes the auto-renewing subscription title, length, price, and price per week from StoreKit, plus functional links to our Privacy Policy (https://lextures.com/privacy) and Terms of Use / EULA (https://lextures.com/terms). Those links stay visible in the paywall footer and open in Safari. Auto-renew terms name the selected plan and explain how to cancel in Settings → Subscriptions. A screen recording is attached in Review Notes.

## App Store Connect (do this on every submission)

- Privacy Policy field: `https://lextures.com/privacy`
- Terms of Use (EULA) field or App Description: `https://lextures.com/terms`
- Repeat the screen-recording path in **App Review Information → Notes**
