import SwiftUI

struct ProfilePersonalDetailsRoute: Hashable {}

/// Demographics and org custom profile fields (M1.5).
struct ProfilePersonalDetailsView: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme

    @State private var phase: Phase = .loading
    @State private var fieldDefinitions: [ProfileFieldDefinition] = []
    @State private var customDraft: [String: String] = [:]
    @State private var demographics = StudentDemographics()
    @State private var raceSelection = ""
    @State private var boolDraft: [String: String] = [:]
    @State private var fieldErrors: [String: String] = [:]
    @State private var saveError: String?
    @State private var saved = false
    @State private var saving = false

    private enum Phase { case loading, ready, failed }

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()
            switch phase {
            case .loading:
                ProgressView().controlSize(.large)
            case .failed:
                failedState
            case .ready:
                form
            }
        }
        .navigationTitle(L.text("mobile.profileDepth.personalDetails.title"))
        .navigationBarTitleDisplayMode(.inline)
        .task { await load() }
    }

    private var failedState: some View {
        VStack(spacing: 12) {
            Text(L.text("mobile.profileDepth.loadError"))
                .font(.subheadline)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            Button(L.text("mobile.common.retry")) { Task { await load() } }
                .font(.subheadline.weight(.semibold))
        }
        .padding(32)
    }

    private var form: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                Text(L.text("mobile.profileDepth.personalDetails.description"))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                if shell.platformFeatures.ffDemographics {
                    demographicsSection
                }

                if !fieldDefinitions.isEmpty {
                    customFieldsSection
                }

                if let saveError {
                    Text(saveError)
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.error)
                        .accessibilityLabel(saveError)
                }

                if saved {
                    Text(L.text("mobile.profileDepth.saved"))
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                }

                Button {
                    Task { await save() }
                } label: {
                    Text(L.text("mobile.profileDepth.save"))
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(.white)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 14)
                        .background(LexturesTheme.primary)
                        .clipShape(RoundedRectangle(cornerRadius: 14, style: .continuous))
                }
                .buttonStyle(.plain)
                .disabled(saving)
            }
            .padding(16)
        }
    }

    private var demographicsSection: some View {
        LMSCard {
            Text(L.text("mobile.profileDepth.demographics.section"))
                .font(LexturesTheme.displayFont(17))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            Text(L.text("mobile.profileDepth.demographics.optional"))
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

            Picker(L.text("mobile.profileDepth.demographics.raceEthnicityCode"), selection: $raceSelection) {
                Text(L.text("mobile.profileDepth.preferNotToSay")).tag("")
                ForEach(ProfileDepthLogic.raceEthnicityOptions.filter { $0.code != ProfileDepthLogic.preferNotToSayRaceCode }, id: \.code) { option in
                    Text(L.dynamicText(option.labelKey)).tag(option.code)
                }
            }
            .pickerStyle(.menu)

            ForEach(["freeLunch", "reducedLunch", "ellStatus", "disabilityStatus", "homelessIndicator", "migrantIndicator"], id: \.self) { key in
                triStateRow(key: key)
            }
        }
    }

    private func triStateRow(key: String) -> some View {
        Picker(L.dynamicText(ProfileDepthLogic.demographicsLabelKey(for: key)), selection: Binding(
            get: { boolDraft[key] ?? "prefer" },
            set: { boolDraft[key] = $0 }
        )) {
            Text(L.text("mobile.profileDepth.preferNotToSay")).tag("prefer")
            Text(L.text("mobile.profileDepth.yes")).tag("true")
            Text(L.text("mobile.profileDepth.no")).tag("false")
        }
        .pickerStyle(.menu)
    }

    private var customFieldsSection: some View {
        LMSCard {
            Text(L.text("mobile.profileDepth.customFields.section"))
                .font(LexturesTheme.displayFont(17))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))

            ForEach(fieldDefinitions) { def in
                fieldEditor(for: def)
                if let error = fieldErrors[def.key] {
                    Text(error)
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.error)
                        .accessibilityLabel(error)
                }
                if def.id != fieldDefinitions.last?.id {
                    Divider()
                }
            }
        }
    }

    @ViewBuilder
    private func fieldEditor(for def: ProfileFieldDefinition) -> some View {
        let label = def.isRequired ? "\(def.label) *" : def.label
        switch def.fieldType {
        case "boolean":
            Picker(label, selection: Binding(
                get: { customDraft[def.key] ?? "prefer" },
                set: { customDraft[def.key] = $0 == "prefer" ? "" : $0 }
            )) {
                Text(L.text("mobile.profileDepth.preferNotToSay")).tag("prefer")
                Text(L.text("mobile.profileDepth.yes")).tag("true")
                Text(L.text("mobile.profileDepth.no")).tag("false")
            }
            .pickerStyle(.menu)
        case "select":
            Picker(label, selection: Binding(
                get: { customDraft[def.key] ?? "" },
                set: { customDraft[def.key] = $0 }
            )) {
                Text(L.text("mobile.emDash")).tag("")
                ForEach(def.selectOptions ?? [], id: \.self) { option in
                    Text(option).tag(option)
                }
            }
            .pickerStyle(.menu)
        default:
            AuthTextField(
                title: label,
                text: Binding(
                    get: { customDraft[def.key] ?? "" },
                    set: { customDraft[def.key] = $0 }
                ),
                placeholder: def.fieldType == "date" ? "YYYY-MM-DD" : "",
                keyboard: def.fieldType == "number" ? .decimalPad : (def.fieldType == "date" ? .numbersAndPunctuation : .default)
            )
        }
    }

    @MainActor
    private func load() async {
        phase = .loading
        saveError = nil
        saved = false
        guard let token = session.accessToken else {
            phase = .failed
            return
        }
        do {
            if shell.platformFeatures.customFieldsEnabled {
                let response = try await LMSAPI.fetchMyProfileFields(accessToken: token)
                fieldDefinitions = response.fields
                customDraft = ProfileDepthLogic.draftFromValues(definitions: response.fields, values: response.values)
            }
            if shell.platformFeatures.ffDemographics {
                demographics = try await LMSAPI.fetchMyDemographics(accessToken: token)
                raceSelection = demographics.raceEthnicityCode ?? ""
                boolDraft = [
                    "freeLunch": triStateString(demographics.freeLunch),
                    "reducedLunch": triStateString(demographics.reducedLunch),
                    "ellStatus": triStateString(demographics.ellStatus),
                    "disabilityStatus": triStateString(demographics.disabilityStatus),
                    "homelessIndicator": triStateString(demographics.homelessIndicator),
                    "migrantIndicator": triStateString(demographics.migrantIndicator),
                ]
            }
            phase = .ready
        } catch {
            phase = .failed
        }
    }

    @MainActor
    private func save() async {
        saveError = nil
        saved = false
        fieldErrors = ProfileDepthLogic.validateCustomFields(definitions: fieldDefinitions, draft: customDraft)
        guard fieldErrors.isEmpty else { return }
        guard let token = session.accessToken else { return }
        saving = true
        defer { saving = false }
        do {
            if shell.platformFeatures.customFieldsEnabled, !fieldDefinitions.isEmpty {
                let encoded = ProfileDepthLogic.encodeCustomFieldValues(definitions: fieldDefinitions, draft: customDraft)
                _ = try await LMSAPI.updateMyProfileFields(ProfileFieldsPatch(values: encoded), accessToken: token)
            }
            if shell.platformFeatures.ffDemographics {
                let patch = StudentDemographicsPatch(
                    freeLunch: parseTriState(boolDraft["freeLunch"]),
                    reducedLunch: parseTriState(boolDraft["reducedLunch"]),
                    ellStatus: parseTriState(boolDraft["ellStatus"]),
                    disabilityStatus: parseTriState(boolDraft["disabilityStatus"]),
                    raceEthnicityCode: raceSelection.isEmpty ? ProfileDepthLogic.preferNotToSayRaceCode : raceSelection,
                    homelessIndicator: parseTriState(boolDraft["homelessIndicator"]),
                    migrantIndicator: parseTriState(boolDraft["migrantIndicator"])
                )
                demographics = try await LMSAPI.updateMyDemographics(patch, accessToken: token)
            }
            saved = true
            await load()
        } catch {
            saveError = L.text("mobile.profileDepth.saveError")
        }
    }

    private func triStateString(_ value: Bool?) -> String {
        guard let value else { return "prefer" }
        return value ? "true" : "false"
    }

    private func parseTriState(_ raw: String?) -> Bool? {
        guard let raw, raw != "prefer" else { return nil }
        return raw == "true"
    }
}