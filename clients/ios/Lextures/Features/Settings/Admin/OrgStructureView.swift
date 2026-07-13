import SwiftUI

/// Org unit tree with rename support (M14.4).
struct OrgStructureView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.openURL) private var openURL

    let orgId: String
    let organizations: [AdminOrgRow]
    let canPickOrg: Bool
    @Binding var selectedOrgId: String

    @State private var tree: [OrgUnitTreeNode] = []
    @State private var loading = false
    @State private var errorMessage: String?
    @State private var renameTarget: OrgUnitTreeNode?
    @State private var renameDraft = ""
    @State private var savingRename = false

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                webLinkCard

                if canPickOrg, !organizations.isEmpty {
                    orgPicker
                }

                if let errorMessage {
                    LMSErrorBanner(message: errorMessage)
                }

                if loading && tree.isEmpty {
                    LMSSkeletonList(count: 4)
                } else if tree.isEmpty {
                    LMSEmptyState(
                        systemImage: "folder.tree",
                        title: L.text("mobile.admin.orgStructure.orgUnits.emptyTitle"),
                        message: L.text("mobile.admin.orgStructure.orgUnits.emptyMessage")
                    )
                } else {
                    VStack(alignment: .leading, spacing: 0) {
                        ForEach(tree) { node in
                            OrgUnitTreeBranch(
                                node: node,
                                depth: 0,
                                onRename: { renameTarget = $0; renameDraft = $0.name }
                            )
                        }
                    }
                }
            }
            .padding(16)
        }
        .refreshable { await loadTree() }
        .task(id: effectiveOrgId) { await loadTree() }
        .sheet(item: $renameTarget) { unit in
            OrgUnitRenameSheet(
                unitName: unit.name,
                draft: $renameDraft,
                saving: savingRename,
                onSave: { Task { await saveRename(unit) } },
                onCancel: { renameTarget = nil }
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
            openURL(AppConfiguration.webURL(path: OrgStructureAdminLogic.webOrgUnitsPath()))
        } label: {
            LMSCard {
                HStack(spacing: 10) {
                    Image(systemName: "safari")
                        .foregroundStyle(LexturesTheme.brandTeal)
                    VStack(alignment: .leading, spacing: 2) {
                        Text(L.text("mobile.admin.orgStructure.webTitle"))
                            .font(.subheadline.weight(.semibold))
                            .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                        Text(L.text("mobile.admin.orgStructure.webHintOrgUnits"))
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

    @MainActor
    private func loadTree() async {
        let targetOrg = effectiveOrgId
        guard !targetOrg.isEmpty, let token = session.accessToken else {
            tree = []
            return
        }
        loading = true
        errorMessage = nil
        defer { loading = false }
        do {
            tree = try await LMSAPI.fetchOrgUnitTree(orgId: targetOrg, accessToken: token)
        } catch {
            tree = []
            errorMessage = OrgStructureAdminLogic.userFacingError(error)
        }
    }

    @MainActor
    private func saveRename(_ unit: OrgUnitTreeNode) async {
        let targetOrg = effectiveOrgId
        guard !targetOrg.isEmpty,
              let token = session.accessToken,
              OrgStructureAdminLogic.isValidTermName(renameDraft) else { return }
        savingRename = true
        defer { savingRename = false }
        do {
            try await LMSAPI.patchOrgUnit(
                orgId: targetOrg,
                unitId: unit.id,
                body: OrgStructureAdminLogic.patchOrgUnitNameRequest(name: renameDraft),
                accessToken: token
            )
            renameTarget = nil
            await loadTree()
        } catch {
            errorMessage = OrgStructureAdminLogic.userFacingError(error)
        }
    }
}

private struct OrgUnitTreeBranch: View {
    @Environment(\.colorScheme) private var colorScheme

    let node: OrgUnitTreeNode
    let depth: Int
    let onRename: (OrgUnitTreeNode) -> Void

    @State private var expanded = true

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            HStack(spacing: 8) {
                if !(node.children ?? []).isEmpty {
                    Button {
                        expanded.toggle()
                    } label: {
                        Image(systemName: expanded ? "chevron.down" : "chevron.right")
                            .font(.caption.weight(.semibold))
                            .frame(width: 28, height: 28)
                    }
                    .buttonStyle(.plain)
                    .accessibilityLabel(expanded ? "Collapse" : "Expand")
                } else {
                    Color.clear.frame(width: 28, height: 28)
                }

                VStack(alignment: .leading, spacing: 2) {
                    Text(node.name)
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    HStack(spacing: 8) {
                        Text(node.unitType)
                        if let count = node.childCourseCount, count > 0 {
                            Text(L.format("mobile.admin.orgStructure.orgUnits.courseCount", Int(count)))
                        }
                    }
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }

                Spacer(minLength: 0)

                Button(L.text("mobile.admin.orgStructure.orgUnits.rename")) {
                    onRename(node)
                }
                .font(.caption.weight(.semibold))
            }
            .padding(.leading, CGFloat(depth) * 14)
            .padding(.vertical, 6)
            .accessibilityElement(children: .combine)

            if expanded {
                ForEach(node.children ?? []) { child in
                    OrgUnitTreeBranch(node: child, depth: depth + 1, onRename: onRename)
                }
            }
        }
    }
}

private struct OrgUnitRenameSheet: View {
    let unitName: String
    @Binding var draft: String
    let saving: Bool
    let onSave: () -> Void
    let onCancel: () -> Void

    var body: some View {
        NavigationStack {
            Form {
                Section {
                    TextField(unitName, text: $draft)
                        .textInputAutocapitalization(.words)
                }
            }
            .navigationTitle(L.text("mobile.admin.orgStructure.orgUnits.renameTitle"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(L.text("mobile.common.cancel"), action: onCancel)
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button(L.text("mobile.admin.orgStructure.orgUnits.renameSave")) {
                        onSave()
                    }
                    .disabled(saving || !OrgStructureAdminLogic.isValidTermName(draft))
                }
            }
        }
        .presentationDetents([.medium])
    }
}
