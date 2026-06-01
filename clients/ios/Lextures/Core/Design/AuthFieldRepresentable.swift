import SwiftUI
import UIKit

/// UITextField wrapper that avoids SwiftUI keyboard accessory layout conflicts on recent iOS versions.
struct AuthFieldRepresentable: UIViewRepresentable {
    @Binding var text: String
    var placeholder: String
    var isSecure: Bool
    var keyboard: UIKeyboardType
    var textContentType: UITextContentType?
    var autocapitalization: UITextAutocapitalizationType

    func makeUIView(context: Context) -> AuthUITextField {
        let field = AuthUITextField()
        field.delegate = context.coordinator
        field.borderStyle = .none
        field.backgroundColor = .clear
        field.autocorrectionType = .no
        field.spellCheckingType = .no
        field.smartDashesType = .no
        field.smartInsertDeleteType = .no
        field.smartQuotesType = .no
        field.autocapitalizationType = autocapitalization
        field.keyboardType = keyboard
        field.textContentType = textContentType
        field.isSecureTextEntry = isSecure
        field.placeholder = placeholder
        field.font = .preferredFont(forTextStyle: .body)
        field.textColor = UIColor.label
        field.addTarget(
            context.coordinator,
            action: #selector(Coordinator.textDidChange),
            for: .editingChanged
        )
        return field
    }

    func updateUIView(_ uiView: AuthUITextField, context: Context) {
        if uiView.text != text {
            uiView.text = text
        }

        uiView.placeholder = placeholder
        uiView.keyboardType = keyboard
        uiView.textContentType = textContentType
        uiView.autocapitalizationType = autocapitalization

        if uiView.isSecureTextEntry != isSecure {
            let current = uiView.text
            uiView.isSecureTextEntry = isSecure
            uiView.text = current
        }
    }

    func makeCoordinator() -> Coordinator {
        Coordinator(text: $text)
    }

    final class Coordinator: NSObject, UITextFieldDelegate {
        @Binding var text: String

        init(text: Binding<String>) {
            _text = text
        }

        @objc func textDidChange(_ sender: UITextField) {
            text = sender.text ?? ""
        }
    }
}

final class AuthUITextField: UITextField {
    override var inputAssistantItem: UITextInputAssistantItem {
        let item = super.inputAssistantItem
        item.leadingBarButtonGroups = []
        item.trailingBarButtonGroups = []
        return item
    }
}
