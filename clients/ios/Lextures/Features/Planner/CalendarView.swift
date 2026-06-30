import EventKit
import SwiftUI

struct CalendarView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    @Bindable var model: PlannerModel
    var onSubscribe: () -> Void

    @State private var monthAnchor = Date()
    @State private var selectedDay = Calendar.current.startOfDay(for: Date())
    @State private var eventToAdd: PlannerCalendarEvent?

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 14) {
                filterBar
                monthHeader
                monthGrid
                agendaSection
            }
            .padding(16)
        }
        .sheet(item: $eventToAdd) { event in
            CalendarEventActionsSheet(event: event, onSubscribe: onSubscribe)
        }
    }

    private var filterBar: some View {
        LMSSegmentedChips(
            options: [""] + model.courseFilters.map(\.courseCode),
            selection: Binding(
                get: { model.selectedCourseCode ?? "" },
                set: { model.selectedCourseCode = $0.isEmpty ? nil : $0 }
            )
        ) { code in
            if code.isEmpty {
                L.text("mobile.planner.filter.allCourses")
            } else {
                model.courseFilters.first(where: { $0.courseCode == code })?.title ?? code
            }
        }
    }

    private var monthHeader: some View {
        HStack {
            Button {
                monthAnchor = Calendar.current.date(byAdding: .month, value: -1, to: monthAnchor) ?? monthAnchor
            } label: {
                Image(systemName: "chevron.left")
            }
            Spacer()
            Text(monthAnchor.formatted(.dateTime.year().month(.wide)))
                .font(LexturesTheme.displayFont(18))
            Spacer()
            Button {
                monthAnchor = Calendar.current.date(byAdding: .month, value: 1, to: monthAnchor) ?? monthAnchor
            } label: {
                Image(systemName: "chevron.right")
            }
        }
        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
    }

    private var monthGrid: some View {
        let cells = PlannerLogic.monthGridCells(monthAnchor: monthAnchor)
        let counts = PlannerLogic.eventCountsByDay(events: model.filteredEvents)
        let columns = Array(repeating: GridItem(.flexible(), spacing: 4), count: 7)

        return VStack(spacing: 8) {
            LazyVGrid(columns: columns, spacing: 4) {
                ForEach(["M", "T", "W", "T", "F", "S", "S"], id: \.self) { label in
                    Text(label)
                        .font(.caption2.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        .frame(maxWidth: .infinity)
                }
                ForEach(cells, id: \.timeIntervalSince1970) { day in
                    dayCell(day, count: counts[PlannerLogic.dateKeyLocal(day)] ?? 0)
                }
            }
        }
        .padding(12)
        .background(LexturesTheme.cardBackground(for: colorScheme))
        .clipShape(RoundedRectangle(cornerRadius: 18, style: .continuous))
    }

    private func dayCell(_ day: Date, count: Int) -> some View {
        let inMonth = Calendar.current.isDate(day, equalTo: monthAnchor, toGranularity: .month)
        let selected = Calendar.current.isDate(day, inSameDayAs: selectedDay)
        let label = PlannerLogic.dateKeyLocal(day)

        return Button {
            selectedDay = Calendar.current.startOfDay(for: day)
        } label: {
            VStack(spacing: 4) {
                Text("\(Calendar.current.component(.day, from: day))")
                    .font(.subheadline.weight(selected ? .bold : .regular))
                    .foregroundStyle(
                        selected
                            ? .white
                            : (inMonth ? LexturesTheme.textPrimary(for: colorScheme) : LexturesTheme.textSecondary(for: colorScheme))
                    )
                Circle()
                    .fill(count > 0 ? LexturesTheme.coral : .clear)
                    .frame(width: 5, height: 5)
            }
            .frame(maxWidth: .infinity, minHeight: 38)
            .background(
                RoundedRectangle(cornerRadius: 10, style: .continuous)
                    .fill(selected ? LexturesTheme.accent(for: colorScheme) : .clear)
            )
        }
        .buttonStyle(.plain)
        .accessibilityLabel(accessibilityDayLabel(day: day, count: count, selected: selected))
    }

    private var agendaSection: some View {
        let dayEvents = PlannerLogic.events(on: selectedDay, events: model.filteredEvents)
        return VStack(alignment: .leading, spacing: 8) {
            LMSSectionHeader(
                title: selectedDay.formatted(.dateTime.weekday(.wide).month().day()),
                systemImage: "list.bullet"
            )
            if dayEvents.isEmpty {
                LMSCard {
                    Text(L.text("mobile.planner.calendar.emptyDay"))
                        .font(.subheadline)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
            } else {
                ForEach(dayEvents) { event in
                    Button {
                        eventToAdd = event
                    } label: {
                        agendaRow(event)
                    }
                    .buttonStyle(.plain)
                }
            }
        }
    }

    private func agendaRow(_ event: PlannerCalendarEvent) -> some View {
        LMSCard(accent: LexturesTheme.coral) {
            HStack(spacing: 12) {
                VStack(alignment: .leading, spacing: 3) {
                    Text(event.title)
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                        .lineLimit(2)
                    if let courseTitle = event.courseTitle {
                        Text(courseTitle)
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                    Text(calendarKindLabel(event.kind))
                        .font(.caption2.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
                Spacer(minLength: 0)
                Text(event.startsAt.formatted(date: .omitted, time: .shortened))
                    .font(.caption.weight(.bold))
                    .foregroundStyle(LexturesTheme.coral)
            }
        }
    }

    private func calendarKindLabel(_ kind: PlannerCalendarEventKind) -> String {
        switch kind {
        case .assignment: return L.text("mobile.planner.kind.assignment")
        case .quiz: return L.text("mobile.planner.kind.quiz")
        case .contentPage: return L.text("mobile.planner.kind.page")
        case .notebookTask: return L.text("mobile.planner.kind.task")
        case .academic: return L.text("mobile.planner.kind.academic")
        }
    }

    private func accessibilityDayLabel(day: Date, count: Int, selected: Bool) -> String {
        let dateText = day.formatted(.dateTime.month().day().year())
        let eventsText = count > 0 ? L.plural("mobile.planner.calendar.eventCount", count: count) : L.text("mobile.planner.calendar.noEvents")
        return selected ? "\(dateText), selected, \(eventsText)" : "\(dateText), \(eventsText)"
    }
}

private struct CalendarEventActionsSheet: View {
    @Environment(\.dismiss) private var dismiss
    @Environment(\.colorScheme) private var colorScheme
    let event: PlannerCalendarEvent
    let onSubscribe: () -> Void
    @State private var addResultMessage: String?

    var body: some View {
        NavigationStack {
            VStack(alignment: .leading, spacing: 16) {
                Text(event.title)
                    .font(LexturesTheme.displayFont(20))
                if let courseTitle = event.courseTitle {
                    Text(courseTitle)
                        .font(.subheadline)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
                Button(L.text("mobile.planner.calendar.addToDevice")) {
                    Task { await addToDeviceCalendar() }
                }
                .buttonStyle(.borderedProminent)
                Button(L.text("mobile.planner.subscribe.title"), action: onSubscribe)
                    .buttonStyle(.bordered)
                if let addResultMessage {
                    Text(addResultMessage)
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
                Spacer()
            }
            .padding(20)
            .navigationTitle(L.text("mobile.planner.calendar.eventActions"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(L.text("mobile.onboarding.back")) { dismiss() }
                }
            }
        }
    }

    private func addToDeviceCalendar() async {
        let store = EKEventStore()
        do {
            let granted = try await store.requestFullAccessToEvents()
            guard granted else {
                addResultMessage = L.text("mobile.planner.calendar.permissionDenied")
                return
            }
            let ekEvent = EKEvent(eventStore: store)
            ekEvent.title = event.title
            ekEvent.startDate = event.startsAt
            ekEvent.endDate = event.endsAt ?? event.startsAt.addingTimeInterval(3600)
            ekEvent.calendar = store.defaultCalendarForNewEvents
            try store.save(ekEvent, span: .thisEvent)
            addResultMessage = L.text("mobile.planner.calendar.added")
        } catch {
            addResultMessage = L.text("mobile.planner.calendar.addFailed")
        }
    }
}