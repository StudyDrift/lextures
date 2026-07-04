import Foundation

/// Feature gating for immersive reader tools (M6.3).
struct ImmersiveReaderCapabilities: Equatable {
    var toolbarEnabled = true
    var readAloudEnabled = true
    var translationEnabled = true
    var captionsEnabled = true
    var preferencesEnabled = true
}