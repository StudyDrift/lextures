import SwiftUI

/// Full-screen distraction-free present mode (MOB.8 / VC.9).
struct BoardPresentModeView: View {
    @Environment(\.dismiss) private var dismiss
    @Environment(\.accessibilityReduceMotion) private var reduceMotion

    let boardTitle: String
    let posts: [BoardPost]
    let sections: [BoardSection]

    @State private var index = 0
    @State private var overview = false

    private var ordered: [BoardPost] {
        BoardsAdvancedLogic.orderedPostsForPresent(posts: posts, sections: sections)
    }

    var body: some View {
        ZStack {
            Color.black.ignoresSafeArea()
            if overview {
                overviewGrid
            } else {
                slideshow
            }
        }
        .statusBarHidden(true)
        .onAppear {
            BoardsAdvancedObservability.record("board_presented")
        }
        .accessibilityElement(children: .contain)
    }

    private var slideshow: some View {
        VStack(spacing: 16) {
            HStack {
                Text(boardTitle)
                    .font(.headline)
                    .foregroundStyle(.white)
                    .lineLimit(1)
                Spacer()
                Button {
                    overview = true
                } label: {
                    Image(systemName: "square.grid.2x2")
                }
                .accessibilityLabel(L.text("mobile.boards.present.overview"))
                Button {
                    dismiss()
                } label: {
                    Image(systemName: "xmark")
                }
                .accessibilityLabel(L.text("mobile.boards.present.close"))
            }
            .foregroundStyle(.white)
            .padding(.horizontal, 20)
            .padding(.top, 12)

            Spacer()

            if ordered.isEmpty {
                Text(L.text("mobile.boards.present.empty"))
                    .foregroundStyle(.white.opacity(0.8))
            } else {
                let post = ordered[min(index, ordered.count - 1)]
                VStack(alignment: .leading, spacing: 12) {
                    if let section = sectionTitle(for: post) {
                        Text(section)
                            .font(.caption.weight(.semibold))
                            .foregroundStyle(.white.opacity(0.7))
                    }
                    if !post.title.isEmpty {
                        Text(post.title)
                            .font(.largeTitle.weight(.bold))
                            .foregroundStyle(.white)
                    }
                    Text(BoardsAdvancedLogic.postBodyText(post))
                        .font(.title3)
                        .foregroundStyle(.white.opacity(0.95))
                }
                .padding(24)
                .frame(maxWidth: .infinity, alignment: .leading)
                .accessibilityElement(children: .combine)
            }

            Spacer()

            HStack {
                Button {
                    move(-1)
                } label: {
                    Image(systemName: "chevron.left.circle.fill")
                        .font(.largeTitle)
                }
                .disabled(index <= 0)
                .accessibilityLabel(L.text("mobile.boards.present.prev"))

                Spacer()
                Text("\(min(index + 1, max(ordered.count, 1))) / \(max(ordered.count, 1))")
                    .foregroundStyle(.white.opacity(0.8))
                    .accessibilityLabel(L.text("mobile.boards.present.position"))
                Spacer()

                Button {
                    move(1)
                } label: {
                    Image(systemName: "chevron.right.circle.fill")
                        .font(.largeTitle)
                }
                .disabled(index >= ordered.count - 1)
                .accessibilityLabel(L.text("mobile.boards.present.next"))
            }
            .foregroundStyle(.white)
            .padding(.horizontal, 24)
            .padding(.bottom, 24)
        }
        .gesture(
            DragGesture(minimumDistance: 40).onEnded { value in
                if value.translation.width < -40 { move(1) }
                else if value.translation.width > 40 { move(-1) }
            }
        )
    }

    private var overviewGrid: some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack {
                Text(L.text("mobile.boards.present.overview"))
                    .font(.headline)
                    .foregroundStyle(.white)
                Spacer()
                Button(L.text("mobile.boards.present.slideshow")) {
                    overview = false
                }
                .foregroundStyle(.white)
            }
            .padding(.horizontal, 16)
            .padding(.top, 12)

            ScrollView {
                LazyVStack(alignment: .leading, spacing: 10) {
                    ForEach(Array(ordered.enumerated()), id: \.element.id) { idx, post in
                        Button {
                            index = idx
                            overview = false
                        } label: {
                            VStack(alignment: .leading, spacing: 4) {
                                Text(post.title.isEmpty ? BoardsAdvancedLogic.postBodyText(post) : post.title)
                                    .font(.headline)
                                    .foregroundStyle(.white)
                                    .lineLimit(2)
                            }
                            .padding(12)
                            .frame(maxWidth: .infinity, alignment: .leading)
                            .background(Color.white.opacity(0.12), in: RoundedRectangle(cornerRadius: 10))
                        }
                    }
                }
                .padding(16)
            }
        }
    }

    private func sectionTitle(for post: BoardPost) -> String? {
        guard let sid = post.sectionId else { return nil }
        return sections.first(where: { $0.id == sid })?.title
    }

    private func move(_ delta: Int) {
        let next = max(0, min(ordered.count - 1, index + delta))
        if reduceMotion {
            index = next
        } else {
            withAnimation(.easeInOut(duration: 0.2)) { index = next }
        }
    }
}
