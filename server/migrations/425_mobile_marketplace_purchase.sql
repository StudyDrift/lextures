-- MOB.7: staged rollout for marketplace purchases + purchased courses on iOS/Android.
ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_mobile_marketplace_purchase BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_mobile_marketplace_purchase IS
    'MOB.7: In-app marketplace claim/buy (Stripe checkout handoff) and Purchased courses library. Default OFF.';
