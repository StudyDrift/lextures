import SwiftUI

/// Bottom sheet for font, spacing, and TTS speed on content surfaces (M6.3).
struct ReadingPreferencesSheet: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss
    @Environment(\.readingPreferencesStore) private var store

    private let fontOptions: [(id: String, label: String)] = [
        ("default", "Default"),
        ("open-dyslexic", "OpenDyslexic"),
        ("atkinson", "Atkinson Hyperlegible"),
        ("system", "System font"),
    ]

    private let spacingOptions: [(id: String, label: String)] = [
        ("normal", "Normal"),
        ("wide", "Wide"),
        ("wider", "Wider"),
    ]

    var body: some View {
        NavigationStack {
            Form {
                Section("Font") {
                    Picker("Font face", selection: fontFaceBinding) {
                        ForEach(fontOptions, id: \.id) { option in
                            Text(option.label).tag(option.id)
                        }
                    }
                    .pickerStyle(.inline)
                    .labelsHidden()
                }

                Section("Letter spacing") {
                    Picker("Letter spacing", selection: letterSpacingBinding) {
                        ForEach(spacingOptions, id: \.id) { option in
                            Text(option.label).tag(option.id)
                        }
                    }
                    .pickerStyle(.segmented)
                    .labelsHidden()
                }

                Section("Word spacing") {
                    Picker("Word spacing", selection: wordSpacingBinding) {
                        ForEach(spacingOptions, id: \.id) { option in
                            Text(option.label).tag(option.id)
                        }
                    }
                    .pickerStyle(.segmented)
                    .labelsHidden()
                }

                Section("Line height") {
                    Picker("Line height", selection: lineHeightBinding) {
                        ForEach(spacingOptions, id: \.id) { option in
                            Text(option.label).tag(option.id)
                        }
                    }
                    .pickerStyle(.segmented)
                    .labelsHidden()
                }

                Section("Read aloud speed") {
                    Slider(value: ttsSpeedBinding, in: 0.5 ... 2.0, step: 0.1) {
                        Text("Speech rate")
                    }
                    Text(String(format: "%.1f×", store.prefs.ttsSpeed))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
            }
            .navigationTitle("Reading preferences")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .confirmationAction) {
                    Button("Done") { dismiss() }
                }
            }
        }
        .presentationDetents([.medium, .large])
    }

    private var fontFaceBinding: Binding<String> {
        Binding(
            get: { store.prefs.fontFace },
            set: { newValue in
                Task {
                    await store.update(
                        ReadingPreferencesPatch(
                            fontFace: newValue,
                            dyslexiaDisplayEnabled: ReaderLogic.dyslexiaFromFontFace(newValue)
                        ),
                        accessToken: session.accessToken
                    )
                }
            }
        )
    }

    private var letterSpacingBinding: Binding<String> {
        Binding(
            get: { store.prefs.letterSpacing },
            set: { newValue in
                Task { await store.update(ReadingPreferencesPatch(letterSpacing: newValue), accessToken: session.accessToken) }
            }
        )
    }

    private var wordSpacingBinding: Binding<String> {
        Binding(
            get: { store.prefs.wordSpacing },
            set: { newValue in
                Task { await store.update(ReadingPreferencesPatch(wordSpacing: newValue), accessToken: session.accessToken) }
            }
        )
    }

    private var lineHeightBinding: Binding<String> {
        Binding(
            get: { store.prefs.lineHeight },
            set: { newValue in
                Task { await store.update(ReadingPreferencesPatch(lineHeight: newValue), accessToken: session.accessToken) }
            }
        )
    }

    private var ttsSpeedBinding: Binding<Double> {
        Binding(
            get: { store.prefs.ttsSpeed },
            set: { newValue in
                Task { await store.update(ReadingPreferencesPatch(ttsSpeed: newValue), accessToken: session.accessToken) }
            }
        )
    }
}