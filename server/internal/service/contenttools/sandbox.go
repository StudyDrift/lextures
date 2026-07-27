package contenttools

import (
	"os"
	"strings"
)

// Sandbox mode values (manifest + platform flag).
const (
	SandboxInProcess = "inprocess"
	SandboxIframe    = "iframe"
)

// Platform sandbox mode env values (CONTENT_TOOLS_SANDBOX_MODE).
const (
	EnvSandboxMode = "CONTENT_TOOLS_SANDBOX_MODE"

	SandboxModeOff      = "off"
	SandboxModeOptIn    = "optin"
	SandboxModeRequired = "required"
)

// ContractVersion is the host↔tool runtime contract major the SDK/host speak (FR-17).
const ContractVersion = 1

// SupportedContractMin/Max define the accepted contract range.
const (
	SupportedContractMin = 1
	SupportedContractMax = 1
)

// PlatformSandboxMode returns off|optin|required (default optin).
func PlatformSandboxMode() string {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(EnvSandboxMode)))
	switch v {
	case SandboxModeOff, SandboxModeOptIn, SandboxModeRequired:
		return v
	case "":
		return SandboxModeOptIn
	default:
		return SandboxModeOptIn
	}
}

// EffectiveSandboxMode resolves the mount mode for a tool manifest under the platform flag.
func EffectiveSandboxMode(manifestSandbox string) string {
	ms := strings.ToLower(strings.TrimSpace(manifestSandbox))
	if ms == "" {
		ms = SandboxInProcess
	}
	switch PlatformSandboxMode() {
	case SandboxModeOff:
		return SandboxInProcess
	case SandboxModeRequired:
		return SandboxIframe
	default: // optin
		if ms == SandboxIframe {
			return SandboxIframe
		}
		return SandboxInProcess
	}
}

// ContractSupported reports whether a tool's declared contract is mountable.
func ContractSupported(contract int) bool {
	if contract <= 0 {
		contract = 1
	}
	return contract >= SupportedContractMin && contract <= SupportedContractMax
}
