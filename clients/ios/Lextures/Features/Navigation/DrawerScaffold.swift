import SwiftUI

/// Reusable left-drawer container implementing the web-parity, two-level navigation:
/// a leading-edge swipe reveals the drawer; when a course is active the first swipe
/// opens the course menu and a second edge swipe escalates to the global menu.
///
/// Interaction summary:
/// - Leading-edge drag (≈24pt hot zone) while closed → slides the drawer in.
///   Target = course menu when `courseAvailable`, otherwise the global menu.
/// - Leading-edge drag while the course menu is open → escalates to the global menu.
/// - Drag left on an open panel, or tap the scrim → closes.
/// - Selecting a drawer row closes the panel while the incoming page slides in lockstep.
struct DrawerScaffold<Main: View, GlobalPanel: View, CoursePanel: View>: View {
    @Binding var state: DrawerState
    /// Reports 0 (closed) … 1 (open) so main content can sync page transitions.
    @Binding var openProgress: CGFloat
    /// Whether a course is active — decides the first-swipe target.
    let courseAvailable: Bool
    @ViewBuilder var main: () -> Main
    @ViewBuilder var globalPanel: () -> GlobalPanel
    @ViewBuilder var coursePanel: () -> CoursePanel

    @State private var openTranslation: CGFloat = 0
    @State private var closeTranslation: CGFloat = 0
    /// Animated open amount (0…1) used when state changes outside of an active drag.
    @State private var settledProgress: CGFloat = 0

    private let edgeHotZone: CGFloat = 24
    private let openThresholdFraction: CGFloat = 0.33
    private let drawerAnimation = Animation.easeInOut(duration: 0.35)

    var body: some View {
        GeometryReader { geo in
            let panelW = min(geo.size.width * 0.82, 380)
            let progress = effectiveProgress(panelW: panelW)
            let visible = progress > 0.001

            ZStack(alignment: .leading) {
                main()
                    .offset(x: panelW * progress)
                    .allowsHitTesting(state == .none && progress < 0.02)

                if visible {
                    // Dimming scrim — tap to dismiss.
                    Color.black
                        .opacity(0.45 * progress)
                        .ignoresSafeArea()
                        .contentShape(Rectangle())
                        .onTapGesture { setState(.none) }

                    panelContent
                        .frame(width: panelW)
                        .frame(maxHeight: .infinity, alignment: .top)
                        .background(.clear)
                        .offset(x: -panelW * (1 - progress))
                        .gesture(closeDrag(panelW: panelW))
                }

                // Leading-edge catcher: opens (closed) or escalates (course menu open).
                if state == .none || state == .course {
                    Color.clear
                        .frame(width: edgeHotZone)
                        .frame(maxHeight: .infinity, alignment: .leading)
                        .contentShape(Rectangle())
                        .gesture(edgeDrag(panelW: panelW))
                }
            }
            .animation(isDragging ? nil : drawerAnimation, value: settledProgress)
            .onChange(of: progress) { _, new in
                openProgress = new
            }
        }
        .onAppear {
            settledProgress = state == .none ? 0 : 1
            openProgress = settledProgress
        }
        .onChange(of: state) { _, new in
            guard !isDragging else { return }
            withAnimation(drawerAnimation) {
                settledProgress = new == .none ? 0 : 1
            }
        }
    }

    private var isDragging: Bool {
        openTranslation > 0 || closeTranslation > 0
    }

    // MARK: Panel content

    @ViewBuilder
    private var panelContent: some View {
        // While opening from `.none`, `state` is still `.none`; show the pending target.
        let displayed: DrawerState = state == .none
            ? (courseAvailable ? .course : .global)
            : state
        switch displayed {
        case .course:
            coursePanel()
        case .global, .none:
            globalPanel()
        }
    }

    // MARK: Geometry

    /// 0 (fully closed) … 1 (fully open), accounting for live drags and animated settle.
    private func effectiveProgress(panelW: CGFloat) -> CGFloat {
        if state != .none && closeTranslation > 0 {
            return max(0, 1 - closeTranslation / panelW)
        }
        if state == .none && openTranslation > 0 {
            return min(1, openTranslation / panelW)
        }
        return settledProgress
    }

    // MARK: Gestures

    private func edgeDrag(panelW: CGFloat) -> some Gesture {
        DragGesture(minimumDistance: 8)
            .onChanged { value in
                guard state == .none else { return } // escalation resolves on end
                openTranslation = min(panelW, max(0, value.translation.width))
            }
            .onEnded { value in
                if state == .course {
                    // Second edge swipe (rightward) escalates to the global menu.
                    if value.translation.width > panelW * 0.25 {
                        setState(.global)
                    }
                    return
                }
                let opened = openTranslation > panelW * openThresholdFraction
                    || value.predictedEndTranslation.width > panelW * 0.6
                if opened {
                    setState(courseAvailable ? .course : .global)
                }
                withAnimation(drawerAnimation) { openTranslation = 0 }
            }
    }

    private func closeDrag(panelW: CGFloat) -> some Gesture {
        DragGesture(minimumDistance: 8)
            .onChanged { value in
                guard state != .none else { return }
                closeTranslation = min(panelW, max(0, -value.translation.width))
            }
            .onEnded { value in
                let closed = closeTranslation > panelW * openThresholdFraction
                    || value.predictedEndTranslation.width < -panelW * 0.6
                if closed {
                    setState(.none)
                }
                withAnimation(drawerAnimation) { closeTranslation = 0 }
            }
    }

    private func setState(_ new: DrawerState) {
        withAnimation(drawerAnimation) {
            state = new
            settledProgress = new == .none ? 0 : 1
            openTranslation = 0
            closeTranslation = 0
        }
    }
}