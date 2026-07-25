/**
 * Re-export PS.3 pin telemetry through the PS.4 typed schema module.
 * Prefer importing from `settings-telemetry` for new call sites.
 */
export {
  emitPinnedSettingsTelemetry,
  onPinnedSettingsTelemetry,
  type PinnedSettingsTelemetryEvent,
} from './settings-telemetry'
