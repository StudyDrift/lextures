import SwiftUI

/// Academic terms list with create and date edit (M14.4).
struct TermsAdminView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.openURL) private var openURL

    let orgId: String
    let organizations: [AdminOrgRow]
    let canPickOrg: Bool
    @Binding var selectedOrgId: String

    @State private var terms: [OrgTerm] = []
    @State private var loading = false
    @State private var errorMessage: String?
    @State private var showCreateSheet = false
    @State private var editTarget: OrgTerm?

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                webLinkCard

                if canPickOrg, !organizations.isEmpty {
                    orgPicker
                }

                Button {
                    showCreateSheet = true
                } label: {
                    Label(L.text("mobile.admin.orgStructure.terms.add"), systemImage: "plus")
                        .font(.subheadline.weight(.semibold))
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 12)
                }
                .buttonStyle(.borderedProminent)
                .disabled(effectiveOrgId.isEmpty)

                if let errorMessage {
                    LMSErrorBanner(message: errorMessage)
                }

                if loading && terms.isEmpty {
                    LMSSkeletonList(count: 3)
                } else if terms.isEmpty {
                    LMSEmptyState(
                        systemImage: "calendar",
                        title: L.text("mobile.admin.orgStructure.terms.emptyTitle"),
                        message: L.text("mobile.admin.orgStructure.terms.emptyMessage")
                    )
                } else {
                    ForEach(terms) { term in
                        termRow(term)
                    }
                }
            }
            .padding(16)
        }
        .refreshable { await loadTerms() }
        .task(id: effectiveOrgId) { await loadTerms() }
        .sheet(isPresented: $showCreateSheet) {
            TermEditorSheet(
                mode: .create,
                term: nil,
                onSave: { name, start, end in
                    Task { await createTerm(name: name, start: start, end: end) }
                }
            )
        }
        .sheet(item: $editTarget) { term in
            TermEditorSheet(
                mode: .edit,
                term: term,
                onSave: { _, start, end in
                    Task { await updateTerm(term, start: start, end: end) }
                }
            )
        }
    }

    private var effectiveOrgId: String {
        canPickOrg ? selectedOrgId : orgId
    }

    private var orgPicker: some View {
        LMSCard {
            Picker(L.text("mobile.admin.orgStructure.selectOrg"), selection: $selectedOrgId) {
                ForEach(organizations) { org in
                    Text(org.name).tag(org.id)
                }
            }
            .pickerStyle(.menu)
        }
    }

    private var webLinkCard: some View {
        Button {
            openURL(AppConfiguration.webURL(path: OrgStructureAdminLogic.webTermsPath()))
        } label: {
            LMSCard {
                HStack(spacing: 10) {
                    Image(systemName: "safari")
                        .foregroundStyle(LexturesTheme.brandTeal)
                    VStack(alignment: .leading, spacing: 2) {
                        Text(L.text("mobile.admin.orgStructure.webTitle"))
                            .font(.subheadline.weight(.semibold))
                            .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                        Text(L.text("mobile.admin.orgStructure.webHint"))
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                    Spacer(minLength: 0)
                    Image(systemName: "arrow.up.right")
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
            }
        }
        .buttonStyle(.plain)
    }

    private func termRow(_ term: OrgTerm) -> some View {
        LMSCard {
            HStack(alignment: .top, spacing: 12) {
                VStack(alignment: .leading, spacing: 4) {
                    Text(term.name)
                        .font(.headline)
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    Text(OrgStructureAdminLogic.formatDateRange(start: term.startDate, end: term.endDate))
                        .font(.subheadline)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    if let status = term.status, !status.isEmpty {
                        Text(status.capitalized)
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                }
                Spacer(minLength: 0)
                Button(L.text("mobile.admin.orgStructure.terms.edit")) {
                    editTarget = term
                }
                .font(.caption.weight(.semibold))
            }
        }
    }

    @MainActor
    private func loadTerms() async {
        let targetOrg = effectiveOrgId
        guard !targetOrg.isEmpty, let token = session.accessToken else {
            terms = []
            return
        }
        loading = true
        errorMessage = nil
        defer { loading = false }
        do {
            terms = try await LMSAPI.fetchOrgTerms(orgId: targetOrg, accessToken: token)
        } catch {
            terms = []
            errorMessage = OrgStructureAdminLogic.userFacingError(error)
        }
    }

    @MainActor
    private func createTerm(name: String, start: String, end: String) async {
        let targetOrg = effectiveOrgId
        guard !targetOrg.isEmpty, let token = session.accessToken else { return }
        guard OrgStructureAdminLogic.isValidDateRange(start: start, end: end) else { return }
        do {
            _ = try await LMSAPI.createAcademicTerm(
                orgId: targetOrg,
                body: OrgStructureAdminLogic.createTermRequest(
                    name: name,
                    termType: OrgStructureAdminLogic.defaultTermType,
                    startDate: start,
                    endDate: end
                ),
                accessToken: token
            )
            showCreateSheet = false
            await loadTerms()
        } catch {
            errorMessage = OrgStructureAdminLogic.userFacingError(error)
        }
    }

    @MainActor
    private func updateTerm(_ term: OrgTerm, start: String, end: String) async {
        let targetOrg = effectiveOrgId
        guard !targetOrg.isEmpty, let token = session.accessToken else { return }
        guard OrgStructureAdminLogic.isValidDateRange(start: start, end: end) else { return }
        do {
            _ = try await LMSAPI.patchAcademicTerm(
                orgId: targetOrg,
                termId: term.id,
                body: OrgStructureAdminLogic.patchTermDatesRequest(startDate: start, endDate: end),
                accessToken: token
            )
            editTarget = nil
            await loadTerms()
        } catch {
            errorMessage = OrgStructureAdminLogic.userFacingError(error)
        }
    }
}

private enum TermEditorMode {
    case create
    case edit
}

private struct TermEditorSheet: View {
    let mode: TermEditorMode
    let term: OrgTerm?
    let onSave: (String, String, String) -> Void

    @Environment(\.dismiss) private var dismiss
    @State private var name = ""
    @State private var startDate = Date()
    @State private var endDate = Date()
    @State private var showConfirm = false

    var body: some View {
        NavigationStack {
            Form {
                if mode == .create {
                    Section {
                        TextField(L.text("mobile.admin.orgStructure.terms.name"), text: $name)
                    }
                } else if let term {
                    Section {
                        Text(term.name)
                    }
                }

                Section {
                    DatePicker(
                        L.text("mobile.admin.orgStructure.terms.startDate"),
                        selection: $startDate,
                        displayedComponents: .date
                    )
                    DatePicker(
                        L.text("mobile.admin.orgStructure.terms.endDate"),
                        selection: $endDate,
                        displayedComponents: .date
                    )
                }
            }
            .navigationTitle(
                mode == .create
                    ? L.text("mobile.admin.orgStructure.terms.addTitle")
                    : L.text("mobile.admin.orgStructure.terms.editTitle")
            )
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(L.text("mobile.common.cancel")) { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button(L.text("mobile.admin.orgStructure.terms.save")) {
                        if mode == .edit {
                            showConfirm = true
                        } else {
                            submit()
                        }
                    }
                    .disabled(!canSave)
                }
            }
            .onAppear {
                if let term {
                    name = term.name
                    startDate = OrgStructureAdminLogic.date(fromIso: term.startDate) ?? Date()
                    endDate = OrgStructureAdminLogic.date(fromIso: term.endDate) ?? Date()
                }
            }
            .alert(L.text("mobile.admin.orgStructure.terms.saveConfirm"), isPresented: $showConfirm) {
                Button(L.text("mobile.common.cancel"), role: .cancel) {}
                Button(L.text("mobile.admin.orgStructure.terms.save"), role: .destructive) { submit() }
            } message: {
                Text(L.text("mobile.admin.orgStructure.terms.saveConfirmMessage"))
            }
        }
        .presentationDetents([.medium, .large])
    }

    private var canSave: Bool {
        let start = OrgStructureAdminLogic.isoDateString(from: startDate)
        let end = OrgStructureAdminLogic.isoDateString(from: endDate)
        if mode == .create {
            return OrgStructureAdminLogic.isValidTermName(name)
                && OrgStructureAdminLogic.isValidDateRange(start: start, end: end)
        }
        return OrgStructureAdminLogic.isValidDateRange(start: start, end: end)
    }

    private func submit() {
        let start = OrgStructureAdminLogic.isoDateString(from: startDate)
        let end = OrgStructureAdminLogic.isoDateString(from: endDate)
        onSave(name, start, end)
        dismiss()
    }
}
