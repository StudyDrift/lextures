import SwiftUI

/// Menu alternative to drag for card arrangement (VC.M3 AC-6).
struct CardArrangeMenu: View {
    let post: BoardPost
    let sections: [BoardSection]
    let siblings: [BoardPost]
    var showTimeline: Bool = false
    var showMap: Bool = false
    var onMoveToSection: (String) -> Void
    var onReorder: (Double) -> Void
    var onSetEventDate: ((String?) -> Void)?
    var onSetCoords: ((Double, Double) -> Void)?

    @State private var showDatePicker = false
    @State private var eventDate = Date()
    @State private var showCoords = false
    @State private var latText = ""
    @State private var lngText = ""

    var body: some View {
        Menu {
            Button(L.text("mobile.boards.arrange.moveUp")) {
                if let idx = BoardsLogic.sortIndexMovingUp(post: post, siblings: siblings) {
                    onReorder(idx)
                }
            }
            .disabled(BoardsLogic.sortIndexMovingUp(post: post, siblings: siblings) == nil)

            Button(L.text("mobile.boards.arrange.moveDown")) {
                if let idx = BoardsLogic.sortIndexMovingDown(post: post, siblings: siblings) {
                    onReorder(idx)
                }
            }
            .disabled(BoardsLogic.sortIndexMovingDown(post: post, siblings: siblings) == nil)

            if !sections.isEmpty {
                Menu(L.text("mobile.boards.arrange.moveToSection")) {
                    ForEach(BoardsLogic.sortedSections(sections)) { section in
                        Button(section.title) {
                            onMoveToSection(section.id)
                        }
                        .disabled(post.sectionId == section.id)
                    }
                }
            }

            if showTimeline, onSetEventDate != nil {
                Button(L.text("mobile.boards.arrange.eventDate")) {
                    if let raw = post.eventDate, let parsed = ISO8601DateFormatter().date(from: raw)
                        ?? DateFormatting.parse(raw) {
                        eventDate = parsed
                    } else {
                        eventDate = Date()
                    }
                    showDatePicker = true
                }
                if post.eventDate != nil {
                    Button(L.text("mobile.boards.arrange.clearEventDate"), role: .destructive) {
                        onSetEventDate?(nil)
                    }
                }
            }

            if showMap, onSetCoords != nil {
                Button(L.text("mobile.boards.arrange.editCoords")) {
                    latText = post.lat.map { String($0) } ?? ""
                    lngText = post.lng.map { String($0) } ?? ""
                    showCoords = true
                }
            }
        } label: {
            Image(systemName: "arrow.up.arrow.down.circle")
                .foregroundStyle(.secondary)
        }
        .accessibilityLabel(L.text("mobile.boards.arrange.menuAria"))
        .sheet(isPresented: $showDatePicker) {
            NavigationStack {
                DatePicker(
                    L.text("mobile.boards.arrange.eventDate"),
                    selection: $eventDate,
                    displayedComponents: .date
                )
                .datePickerStyle(.graphical)
                .padding()
                .navigationTitle(L.text("mobile.boards.arrange.eventDate"))
                .navigationBarTitleDisplayMode(.inline)
                .toolbar {
                    ToolbarItem(placement: .cancellationAction) {
                        Button(L.text("mobile.common.cancel")) { showDatePicker = false }
                    }
                    ToolbarItem(placement: .confirmationAction) {
                        Button(L.text("mobile.common.save")) {
                            let formatter = ISO8601DateFormatter()
                            formatter.formatOptions = [.withFullDate]
                            onSetEventDate?(formatter.string(from: eventDate))
                            showDatePicker = false
                        }
                    }
                }
            }
            .presentationDetents([.medium])
        }
        .alert(L.text("mobile.boards.arrange.setCoords"), isPresented: $showCoords) {
            TextField(L.text("mobile.boards.arrange.latPrompt"), text: $latText)
                .keyboardType(.decimalPad)
            TextField(L.text("mobile.boards.arrange.lngPrompt"), text: $lngText)
                .keyboardType(.decimalPad)
            Button(L.text("mobile.boards.arrange.saveCoords")) {
                guard let lat = Double(latText), let lng = Double(lngText),
                      lat >= -90, lat <= 90, lng >= -180, lng <= 180 else { return }
                onSetCoords?(lat, lng)
            }
            Button(L.text("mobile.common.cancel"), role: .cancel) {}
        }
    }
}
