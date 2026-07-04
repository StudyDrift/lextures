import AVKit
import SwiftUI

/// Video player with synced captions and transcript sheet (M6.3).
struct CaptionedPlayerView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.accessibilityReduceMotion) private var reduceMotion

    let url: URL
    var storageObjectId: String?

    @State private var player = AVPlayer()
    @State private var captions: [CaptionRecord] = []
    @State private var cues: [ReaderLogic.VttCue] = []
    @State private var selectedCaptionId: String?
    @State private var captionsEnabled = false
    @State private var currentCue: ReaderLogic.VttCue?
    @State private var showTranscript = false
    @State private var timeObserver: Any?

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            ZStack(alignment: .bottom) {
                VideoPlayer(player: player)
                    .frame(maxWidth: .infinity)
                    .aspectRatio(16 / 9, contentMode: .fit)
                    .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
                    .accessibilityLabel("Course video")

                if captionsEnabled, let cue = currentCue {
                    Text(cue.text)
                        .font(.caption.weight(.semibold))
                        .multilineTextAlignment(.center)
                        .foregroundStyle(.white)
                        .padding(.horizontal, 12)
                        .padding(.vertical, 6)
                        .background(Color.black.opacity(0.72))
                        .clipShape(RoundedRectangle(cornerRadius: 8, style: .continuous))
                        .padding(.bottom, 10)
                        .padding(.horizontal, 12)
                        .accessibilityAddTraits(.updatesFrequently)
                        .animation(reduceMotion ? nil : .easeInOut(duration: 0.15), value: cue.text)
                }
            }

            if !captions.isEmpty {
                HStack(spacing: 10) {
                    Button {
                        captionsEnabled.toggle()
                    } label: {
                        Label(captionsEnabled ? "Captions on" : "Captions off", systemImage: "captions.bubble")
                            .font(.caption.weight(.semibold))
                    }
                    .minimumTapTarget()

                    if captions.count > 1 {
                        Picker("Caption language", selection: $selectedCaptionId) {
                            ForEach(captions) { caption in
                                Text(ReaderLogic.localeLabel(caption.lang)).tag(Optional(caption.id))
                            }
                        }
                        .pickerStyle(.menu)
                        .onChange(of: selectedCaptionId) { _, newValue in
                            Task { await loadVtt(captionId: newValue) }
                        }
                    }

                    Button("Transcript") { showTranscript = true }
                        .font(.caption.weight(.semibold))
                        .minimumTapTarget()
                }
            }
        }
        .onAppear {
            player.replaceCurrentItem(with: AVPlayerItem(url: url))
            addTimeObserver()
            Task { await loadCaptions() }
        }
        .onDisappear {
            if let timeObserver {
                player.removeTimeObserver(timeObserver)
            }
            player.pause()
        }
        .sheet(isPresented: $showTranscript) {
            NavigationStack {
                ScrollView {
                    VStack(alignment: .leading, spacing: 10) {
                        ForEach(Array(cues.enumerated()), id: \.offset) { _, cue in
                            Text(cue.text)
                                .font(.subheadline)
                                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                        }
                    }
                    .padding(16)
                }
                .navigationTitle("Transcript")
                .navigationBarTitleDisplayMode(.inline)
                .toolbar {
                    ToolbarItem(placement: .confirmationAction) {
                        Button("Done") { showTranscript = false }
                    }
                }
            }
        }
    }

    private func addTimeObserver() {
        let interval = CMTime(seconds: 0.25, preferredTimescale: 600)
        timeObserver = player.addPeriodicTimeObserver(forInterval: interval, queue: .main) { time in
            let seconds = time.seconds
            currentCue = ReaderLogic.activeCue(at: seconds, in: cues)
        }
    }

    private func loadCaptions() async {
        guard let token = session.accessToken else { return }
        let objectId = storageObjectId ?? ReaderLogic.storageObjectId(from: url)
        guard let objectId else { return }
        let records = (try? await LMSAPI.fetchCaptions(objectId: objectId, accessToken: token)) ?? []
        captions = ReaderLogic.readyCaptions(records)
        selectedCaptionId = captions.first?.id
        if let first = captions.first?.id {
            await loadVtt(captionId: first)
        }
    }

    private func loadVtt(captionId: String?) async {
        guard let token = session.accessToken,
              let objectId = storageObjectId ?? ReaderLogic.storageObjectId(from: url),
              let captionId
        else { return }
        let raw = (try? await LMSAPI.fetchCaptionVtt(
            objectId: objectId,
            captionId: captionId,
            accessToken: token
        )) ?? ""
        cues = ReaderLogic.parseVtt(raw)
    }
}