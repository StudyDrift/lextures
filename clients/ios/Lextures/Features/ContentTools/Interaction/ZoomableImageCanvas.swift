// swiftlint:disable identifier_name
import SwiftUI

/// Fit-to-width image canvas with pinch-zoom / pan for diagram_hotspot (CT.M7).
struct ZoomableImageCanvas<Overlay: View>: View {
    let urlString: String
    let alt: String
    let naturalWidth: Double
    let naturalHeight: Double
    @Binding var zoom: CGFloat
    @Binding var pan: CGSize
    @ViewBuilder var overlay: (_ size: CGSize) -> Overlay

    @State private var baseZoom: CGFloat = 1
    @State private var basePan: CGSize = .zero

    private let minZoom: CGFloat = 1
    private let maxZoom: CGFloat = 4

    var body: some View {
        GeometryReader { geo in
            let size = geo.size
            ZStack {
                AuthorizedNotebookImage(urlString: urlString, alt: alt)
                    .frame(width: size.width, height: size.height)
                    .clipped()
                    .accessibilityLabel(alt)
                overlay(size)
            }
            .scaleEffect(zoom)
            .offset(pan)
            .gesture(
                SimultaneousGesture(
                    MagnificationGesture()
                        .onChanged { value in
                            zoom = min(maxZoom, max(minZoom, baseZoom * value))
                        }
                        .onEnded { _ in
                            baseZoom = zoom
                            if zoom <= minZoom {
                                zoom = minZoom
                                baseZoom = minZoom
                                pan = .zero
                                basePan = .zero
                            }
                        },
                    DragGesture()
                        .onChanged { value in
                            guard zoom > minZoom else { return }
                            pan = CGSize(
                                width: basePan.width + value.translation.width,
                                height: basePan.height + value.translation.height
                            )
                        }
                        .onEnded { _ in
                            basePan = pan
                        }
                )
            )
            .onAppear {
                _ = naturalWidth
                _ = naturalHeight
            }
        }
        .aspectRatio(
            naturalWidth > 0 && naturalHeight > 0 ? CGFloat(naturalWidth / naturalHeight) : 4 / 3,
            contentMode: .fit
        )
        .clipped()
    }
}
