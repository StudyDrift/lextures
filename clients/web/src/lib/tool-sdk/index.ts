export { TOOL_SDK_CONTRACT_VERSION } from './contract'
export type { ToolProps, ToolManifestLike, DefinedTool } from './contract'
export { defineTool } from './define-tool'
export {
  useToolState,
  useToolAction,
  useToolAnnounce,
  useToolI18n,
  useRafBatch,
} from './hooks'
export {
  ToolShell,
  ToolPrompt,
  ToolActions,
  ToolFeedback,
  ToolScore,
} from './ui/primitives'
export { renderTool } from './harness/render-tool'
export type { RenderToolOptions, RenderToolResult } from './harness/render-tool'
export {
  BRIDGE_VERSION,
  BRIDGE_MAX_MESSAGE_BYTES,
  BRIDGE_MAX_MESSAGES_PER_SEC,
  BridgeRateLimiter,
  isBridgeFromTool,
  isBridgeToTool,
  measureMessageBytes,
} from './bridge/protocol'
export type { BridgeToTool, BridgeFromTool, BridgeMessage } from './bridge/protocol'
export { parseSemVer, compareSemVer, resolveWithinMajor } from './versioning/semver'
export {
  classifySchemaDiff,
  assertVersionCoversSchemaDiff,
} from './versioning/schema-diff'
export type { BumpKind, SchemaDiffFinding } from './versioning/schema-diff'
