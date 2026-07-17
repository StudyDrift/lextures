import Foundation

/// Helpers for visual collaboration boards (VC.M1–VC.M7).
enum BoardsLogic {
    private static let isoFractional: ISO8601DateFormatter = {
        let f = ISO8601DateFormatter()
        f.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        return f
    }()
    private static let isoBasic: ISO8601DateFormatter = {
        let f = ISO8601DateFormatter()
        f.formatOptions = [.withInternetDateTime]
        return f
    }()

    /// Client-side debounce of repeat reports on the same target (server also rate-limits).
    private static var reportedTargets = Set<String>()
    private static let reportedLock = NSLock()
    /// Unknown / future layouts fall back to stream (AC-7).
    static func resolveLayout(_ raw: String?) -> BoardLayout {
        guard let raw, let layout = BoardLayout(rawValue: raw.lowercased()) else {
            return .stream
        }
        return layout
    }

    /// Fractional index between neighbors (client-side helper for drag reorder).
    static func midpointSortIndex(before: Double?, after: Double?) -> Double {
        switch (before, after) {
        case (nil, nil): return 0
        case (nil, let after?): return after - 1
        case (let before?, nil): return before + 1
        case (let before?, let after?):
            if after <= before { return before + 1 }
            return (before + after) / 2
        }
    }

    static func canArrangePost(
        post: BoardPost,
        board: Board?,
        currentUserId: String?,
        canManage: Bool
    ) -> Bool {
        if canManage { return true }
        if isBoardLocked(board) { return false }
        if let caps = board?.capabilities?.canArrange, caps == false { return false }
        if board?.canArrange == false { return false }
        if board?.layoutLocked == true { return false }
        guard let currentUserId, !currentUserId.isEmpty, let author = post.authorId else { return false }
        return author.caseInsensitiveCompare(currentUserId) == .orderedSame
    }

    static func parseBoardDate(_ raw: String?) -> Date? {
        guard let raw, !raw.isEmpty else { return nil }
        if let d = isoFractional.date(from: raw) { return d }
        return isoBasic.date(from: raw)
    }

    static func isBoardLocked(_ board: Board?) -> Bool {
        board?.locked == true
    }

    static func isBoardFrozen(_ board: Board?, now: Date = Date()) -> Bool {
        guard let until = parseBoardDate(board?.frozenUntil) else { return false }
        return until > now
    }

    /// Lock blocks all non-manager writes; freeze blocks post/comment only (matches server write-gate).
    static func canWritePosts(board: Board?, canManage: Bool, now: Date = Date()) -> Bool {
        if canManage { return true }
        if isBoardLocked(board) { return false }
        if isBoardFrozen(board, now: now) { return false }
        return true
    }

    static func canWriteInteractions(board: Board?, canManage: Bool) -> Bool {
        if canManage { return true }
        return !isBoardLocked(board)
    }

    static func postSafetyState(_ post: BoardPost) -> BoardPostSafetyState {
        if post.hidden || post.status.lowercased() == BoardPostStatus.rejected.rawValue {
            return .removed
        }
        if post.status.lowercased() == BoardPostStatus.pending.rawValue {
            return .pendingApproval
        }
        if let scan = post.attachment?.scanStatus.lowercased() {
            if scan == "blocked" { return .fileBlocked }
            if scan == "pending" { return .fileScanning }
        }
        return .normal
    }

    static func reportTargetKey(postId: String?, commentId: String?) -> String {
        if let commentId, !commentId.isEmpty { return "c:\(commentId)" }
        if let postId, !postId.isEmpty { return "p:\(postId)" }
        return ""
    }

    static func hasReported(postId: String? = nil, commentId: String? = nil) -> Bool {
        let key = reportTargetKey(postId: postId, commentId: commentId)
        guard !key.isEmpty else { return false }
        reportedLock.lock()
        defer { reportedLock.unlock() }
        return reportedTargets.contains(key)
    }

    static func markReported(postId: String? = nil, commentId: String? = nil) {
        let key = reportTargetKey(postId: postId, commentId: commentId)
        guard !key.isEmpty else { return }
        reportedLock.lock()
        reportedTargets.insert(key)
        reportedLock.unlock()
    }

    /// Test helper — clears in-memory report debounce state.
    static func resetReportedTargetsForTests() {
        reportedLock.lock()
        reportedTargets.removeAll()
        reportedLock.unlock()
    }

    static func isFilterBlockMessage(_ message: String?) -> Bool {
        let msg = (message ?? "").lowercased()
        return msg.contains("could not be posted") || msg.contains("couldn't post") || msg.contains("revise")
    }

    static func isLockOrFreezeMessage(_ message: String?) -> Bool {
        let msg = (message ?? "").lowercased()
        return msg.contains("locked") || msg.contains("frozen")
    }

    static func moderationMode(for board: Board?) -> BoardModerationMode {
        BoardModerationMode(rawValue: (board?.moderationMode ?? "open").lowercased()) ?? .open
    }

    static func filterAction(for board: Board?) -> BoardFilterAction {
        BoardFilterAction(rawValue: (board?.filterAction ?? "flag").lowercased()) ?? .flag
    }

    /// Org floor: managers cannot loosen below approval + block.
    static func moderationControlsLockedByOrgFloor(_ board: Board?) -> Bool {
        board?.minorModerationFloor == true
    }

    static func sortedSections(_ sections: [BoardSection]) -> [BoardSection] {
        sections.sorted { lhs, rhs in
            if lhs.sortIndex != rhs.sortIndex { return lhs.sortIndex < rhs.sortIndex }
            return lhs.createdAt < rhs.createdAt
        }
    }

    static func postsInSection(_ posts: [BoardPost], sectionId: String?) -> [BoardPost] {
        posts
            .filter { sectionId == nil ? $0.sectionId == nil : $0.sectionId == sectionId }
            .sorted { lhs, rhs in
                if lhs.sortIndex != rhs.sortIndex { return lhs.sortIndex < rhs.sortIndex }
                return lhs.createdAt < rhs.createdAt
            }
    }

    static func datedPosts(_ posts: [BoardPost]) -> [BoardPost] {
        posts
            .filter { !($0.eventDate ?? "").isEmpty }
            .sorted { ($0.eventDate ?? "") < ($1.eventDate ?? "") }
    }

    static func undatedPosts(_ posts: [BoardPost]) -> [BoardPost] {
        posts.filter { ($0.eventDate ?? "").isEmpty }
    }

    static func pinnedPosts(_ posts: [BoardPost]) -> [BoardPost] {
        posts.filter { $0.lat != nil && $0.lng != nil }
    }

    static func unpinnedPosts(_ posts: [BoardPost]) -> [BoardPost] {
        posts.filter { $0.lat == nil || $0.lng == nil }
    }

    struct MapPinCluster: Hashable {
        var lat: Double
        var lng: Double
        var postIds: [String]
    }

    /// Simple grid clustering (parity with web map layout).
    static func clusterPins(posts: [BoardPost], zoom: Double) -> [MapPinCluster] {
        let pins = pinnedPosts(posts)
        let cell = max(2.0, 40.0 / max(zoom, 1))
        var buckets: [String: MapPinCluster] = [:]
        for post in pins {
            guard let lat = post.lat, let lng = post.lng else { continue }
            let key = "\(Int(floor((lat + 90) / cell)))_\(Int(floor((lng + 180) / cell)))"
            if var existing = buckets[key] {
                existing.postIds.append(post.id)
                let n = Double(existing.postIds.count)
                existing.lat = ((existing.lat * (n - 1)) + lat) / n
                existing.lng = ((existing.lng * (n - 1)) + lng) / n
                buckets[key] = existing
            } else {
                buckets[key] = MapPinCluster(lat: lat, lng: lng, postIds: [post.id])
            }
        }
        return Array(buckets.values)
    }

    static func sortIndexMovingUp(post: BoardPost, siblings: [BoardPost]) -> Double? {
        let ordered = siblings.sorted { $0.sortIndex < $1.sortIndex }
        guard let idx = ordered.firstIndex(where: { $0.id == post.id }), idx > 0 else { return nil }
        let before = idx >= 2 ? ordered[idx - 2].sortIndex : nil
        let after = ordered[idx - 1].sortIndex
        return midpointSortIndex(before: before, after: after)
    }

    static func sortIndexMovingDown(post: BoardPost, siblings: [BoardPost]) -> Double? {
        let ordered = siblings.sorted { $0.sortIndex < $1.sortIndex }
        guard let idx = ordered.firstIndex(where: { $0.id == post.id }), idx < ordered.count - 1 else {
            return nil
        }
        let before = ordered[idx + 1].sortIndex
        let after = idx + 2 < ordered.count ? ordered[idx + 2].sortIndex : nil
        return midpointSortIndex(before: before, after: after)
    }

    static func layoutHidesSortControls(_ layout: BoardLayout) -> Bool {
        switch layout {
        case .canvas, .timeline, .map, .columns: return true
        case .wall, .stream, .grid: return false
        }
    }

    static func canCreateBoards(courseCode: String, permissions: [String]) -> Bool {
        CourseSettingsLogic.canManageCourse(courseCode: courseCode, permissions: permissions)
    }

    /// Prefer server `capabilities.canManage`; fall back to course create permission (VC.M6 FR-1).
    static func canManageBoard(
        board: Board?,
        courseCode: String,
        permissions: [String]
    ) -> Bool {
        if let caps = board?.capabilities?.canManage {
            return caps
        }
        return canCreateBoards(courseCode: courseCode, permissions: permissions)
    }

    /// Prefer server `capabilities.canPost` / `canPost`; until those are absent, fall back to create permission (VC.M2 FR-11).
    /// Callers should also AND with `canWritePosts` for lock/freeze.
    static func canPost(
        board: Board?,
        courseCode: String,
        permissions: [String]
    ) -> Bool {
        if let caps = board?.capabilities?.canPost {
            return caps
        }
        if let canPost = board?.canPost {
            return canPost
        }
        return canCreateBoards(courseCode: courseCode, permissions: permissions)
    }

    /// Prefer server `capabilities.canInteract` / `canInteract`; default true when unset (VC.M5 FR-9 / VC.M6).
    /// Callers should also AND with `canWriteInteractions` / `canWritePosts` for lock/freeze.
    static func canInteract(board: Board?) -> Bool {
        if let caps = board?.capabilities?.canInteract {
            return caps
        }
        if let canInteract = board?.canInteract {
            return canInteract
        }
        return true
    }

    static func canGrade(board: Board?, canManage: Bool) -> Bool {
        canManage && BoardReactionMode.fromAPI(board?.reactionMode) == .grade
    }

    /// Authorship strictly from server payload — never invent an author (VC.M6 FR-2).
    static func attributionLabel(authorId: String?, guestDisplayName: String?) -> String? {
        if let guest = guestDisplayName?.trimmingCharacters(in: .whitespacesAndNewlines), !guest.isEmpty {
            return guest
        }
        if let authorId = authorId?.trimmingCharacters(in: .whitespacesAndNewlines), !authorId.isEmpty {
            return authorId
        }
        return nil
    }

    static func attributionLabel(for post: BoardPost) -> String? {
        attributionLabel(authorId: post.authorId, guestDisplayName: post.guestDisplayName)
    }

    /// External share links when the server allows (and not COPPA-floored).
    static func externalSharingAllowed(board: Board?) -> Bool {
        if board?.minorModerationFloor == true { return false }
        return board?.externalSharingAllowed == true
    }

    /// `public`/`link` visibility when external sharing is allowed (unlocked with VC.M7 moderation).
    static func visibilityOptions(for board: Board?) -> [BoardVisibility] {
        var opts = BoardVisibility.inCourse
        if externalSharingAllowed(board: board) {
            opts.append(contentsOf: [.link, .public])
        }
        return opts
    }

    static func shareURL(for share: BoardShare) -> URL? {
        if let raw = share.url?.trimmingCharacters(in: .whitespacesAndNewlines), !raw.isEmpty {
            if raw.hasPrefix("http://") || raw.hasPrefix("https://") {
                return URL(string: raw)
            }
            return AppConfiguration.webURL(path: raw.hasPrefix("/") ? raw : "/\(raw)")
        }
        if let token = share.token?.trimmingCharacters(in: .whitespacesAndNewlines), !token.isEmpty {
            return AppConfiguration.webURL(path: "/board-links/\(token)")
        }
        return nil
    }

    static func classifyBoardLinkError(status: Int, message: String?) -> BoardLinkAccessState {
        let msg = (message ?? "").lowercased()
        if status == 401 || msg.contains("password") || msg.contains("incorrect") {
            return .needsPassword
        }
        return .denied
    }

    static func assignmentLinked(_ board: Board?) -> Bool {
        guard let id = board?.assignmentId?.trimmingCharacters(in: .whitespacesAndNewlines) else {
            return false
        }
        return !id.isEmpty
    }

    static func canEditOrDeletePost(
        post: BoardPost,
        currentUserId: String?,
        canManage: Bool
    ) -> Bool {
        if canManage { return true }
        guard let currentUserId, !currentUserId.isEmpty, let author = post.authorId else { return false }
        return author == currentUserId
    }

    /// FERPA-safe: render only the grade the server returned for this viewer (nil for peers).
    static func visibleGrade(for post: BoardPost) -> Double? {
        post.grade
    }

    static func applyReactionResult(_ post: BoardPost, result: BoardReactionResult) -> BoardPost {
        var next = post
        next.reactionCount = result.reactionCount ?? post.reactionCount
        next.myReaction = result.active ? (result.myReaction ?? post.myReaction) : nil
        next.avgStars = result.avgStars ?? (result.active ? post.avgStars : nil)
        next.commentCount = result.commentCount ?? post.commentCount
        next.grade = result.grade ?? (result.active ? post.grade : nil)
        return next
    }

    /// Optimistic like/vote toggle before the server responds.
    static func optimisticToggleReaction(_ post: BoardPost, kind: String) -> BoardPost {
        var next = post
        let pressed = post.myReaction != nil
        if pressed {
            next.myReaction = nil
            next.reactionCount = max(0, (post.reactionCount ?? 1) - 1)
        } else {
            next.myReaction = BoardMyReaction(kind: kind)
            next.reactionCount = (post.reactionCount ?? 0) + 1
        }
        return next
    }

    static func optimisticSetStars(_ post: BoardPost, value: Int) -> BoardPost {
        var next = post
        let had = post.myReaction != nil
        next.myReaction = BoardMyReaction(kind: "star", value: Double(value))
        if !had {
            next.reactionCount = (post.reactionCount ?? 0) + 1
        }
        return next
    }

    static func boardPostReactionScore(_ post: BoardPost, mode: BoardReactionMode) -> Double {
        switch mode {
        case .star:
            return (post.avgStars ?? 0) * 1000 + Double(post.reactionCount ?? 0)
        case .grade:
            return post.grade ?? Double(post.reactionCount ?? 0)
        case .like, .vote:
            return Double(post.reactionCount ?? 0)
        case .none:
            return 0
        }
    }

    struct NestedComment: Hashable {
        var comment: BoardComment
        var children: [BoardComment]
    }

    static func nestComments(_ comments: [BoardComment]) -> [NestedComment] {
        var byParent: [String?: [BoardComment]] = [:]
        for c in comments {
            byParent[c.parentId, default: []].append(c)
        }
        let roots = byParent[nil] ?? []
        return roots.map { NestedComment(comment: $0, children: byParent[$0.id] ?? []) }
    }

    static func visibleComments(_ comments: [BoardComment], canManageBoard: Bool) -> [BoardComment] {
        canManageBoard ? comments : comments.filter { !$0.hidden }
    }

    static func commentPlainText(_ comment: BoardComment) -> String {
        if let text = comment.body?.text?.trimmingCharacters(in: .whitespacesAndNewlines), !text.isEmpty {
            return text
        }
        if let html = comment.body?.html, !html.isEmpty {
            return stripHTML(html)
        }
        return ""
    }

    static func formatAvgStars(_ avg: Double) -> String {
        String(format: "%.1f", avg)
    }

    static func formatGrade(_ value: Double) -> String {
        if value.rounded() == value {
            return String(Int(value))
        }
        return String(format: "%g", value)
    }

    /// Sort by `updatedAt` descending (newest first). Archived boards stay unless filtered out.
    static func sortedBoards(_ boards: [Board], includeArchived: Bool) -> [Board] {
        let filtered = includeArchived ? boards : boards.filter { !$0.archived }
        return filtered.sorted { lhs, rhs in
            let l = DateFormatting.parse(lhs.updatedAt) ?? .distantPast
            let r = DateFormatting.parse(rhs.updatedAt) ?? .distantPast
            return l > r
        }
    }

    /// Sort posts for layouts that support sort controls (FR-9). Defaults to newest.
    static func sortedPosts(
        _ posts: [BoardPost],
        mode: BoardSortMode = .newest,
        reactionMode: BoardReactionMode = .none
    ) -> [BoardPost] {
        switch mode {
        case .newest:
            return posts.sorted { lhs, rhs in
                let l = DateFormatting.parse(lhs.createdAt) ?? .distantPast
                let r = DateFormatting.parse(rhs.createdAt) ?? .distantPast
                if l != r { return l > r }
                return lhs.sortIndex > rhs.sortIndex
            }
        case .oldest:
            return posts.sorted { lhs, rhs in
                let l = DateFormatting.parse(lhs.createdAt) ?? .distantPast
                let r = DateFormatting.parse(rhs.createdAt) ?? .distantPast
                if l != r { return l < r }
                return lhs.sortIndex < rhs.sortIndex
            }
        case .author:
            return posts.sorted { lhs, rhs in
                let aa = (lhs.authorId ?? "").lowercased()
                let bb = (rhs.authorId ?? "").lowercased()
                if aa != bb { return aa < bb }
                return lhs.sortIndex < rhs.sortIndex
            }
        case .mostReacted:
            return posts.sorted { lhs, rhs in
                let sa = boardPostReactionScore(lhs, mode: reactionMode)
                let sb = boardPostReactionScore(rhs, mode: reactionMode)
                if sa != sb { return sa > sb }
                let l = DateFormatting.parse(lhs.createdAt) ?? .distantPast
                let r = DateFormatting.parse(rhs.createdAt) ?? .distantPast
                return l > r
            }
        }
    }

    static func relativeUpdatedLabel(_ board: Board) -> String {
        let relative = DateFormatting.formatRelative(board.updatedAt)
        guard !relative.isEmpty else { return "" }
        return L.format("mobile.boards.updatedRelative", relative)
    }

    static func isKnownContentType(_ raw: String) -> Bool {
        BoardContentType.known.contains(raw.lowercased())
    }

    static func bodyPlainText(_ post: BoardPost) -> String {
        if let text = post.body?.text?.trimmingCharacters(in: .whitespacesAndNewlines), !text.isEmpty {
            return text
        }
        if let html = post.body?.html, !html.isEmpty {
            return stripHTML(html)
        }
        return ""
    }

    /// Never returns a URL when AV scan is pending/blocked (VC.M2 FR-8).
    static func attachmentMediaURL(_ attachment: BoardAttachment?) -> URL? {
        guard let attachment else { return nil }
        guard attachment.scanStatus.lowercased() == "clean" else { return nil }
        guard let raw = attachment.url?.trimmingCharacters(in: .whitespacesAndNewlines), !raw.isEmpty else {
            return nil
        }
        return absoluteURL(raw)
    }

    static func absoluteURL(_ pathOrURL: String) -> URL? {
        let trimmed = pathOrURL.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return nil }
        if trimmed.hasPrefix("http://") || trimmed.hasPrefix("https://") {
            return URL(string: trimmed)
        }
        return AppConfiguration.apiURL(path: trimmed.hasPrefix("/") ? trimmed : "/\(trimmed)")
    }

    static func validateCompose(
        contentType: BoardContentType,
        text: String,
        linkUrl: String,
        hasFile: Bool,
        altText: String,
        hasAudio: Bool
    ) -> BoardComposeValidation {
        switch contentType {
        case .text:
            return text.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty ? .missingText : .ok
        case .link, .video:
            return linkUrl.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty ? .missingLink : .ok
        case .image:
            if !hasFile { return .missingFile }
            if altText.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty { return .missingAltText }
            return .ok
        case .file:
            return hasFile ? .ok : .missingFile
        case .audio:
            return hasAudio || hasFile ? .ok : .missingAudio
        case .drawing:
            return .missingFile
        }
    }

    static func videoEmbedFromUrl(_ urlString: String) -> BoardVideoEmbed? {
        let trimmed = urlString.trimmingCharacters(in: .whitespacesAndNewlines)
        guard let url = URL(string: trimmed), let host = url.host?.lowercased() else { return nil }
        if host == "youtu.be" {
            let id = url.path.split(separator: "/").first.map(String.init)
            if let id, !id.isEmpty { return BoardVideoEmbed(provider: "youtube", id: id) }
            return nil
        }
        if host.contains("youtube.com") {
            if let v = URLComponents(url: url, resolvingAgainstBaseURL: false)?
                .queryItems?.first(where: { $0.name == "v" })?.value, !v.isEmpty {
                return BoardVideoEmbed(provider: "youtube", id: v)
            }
            let parts = url.path.split(separator: "/").map(String.init)
            if let idx = parts.firstIndex(where: { $0 == "embed" || $0 == "shorts" || $0 == "v" }),
               idx + 1 < parts.count {
                return BoardVideoEmbed(provider: "youtube", id: parts[idx + 1])
            }
        }
        if host.contains("vimeo.com") {
            if let id = url.path.split(separator: "/").last.map(String.init),
               !id.isEmpty, id.allSatisfy(\.isNumber) {
                return BoardVideoEmbed(provider: "vimeo", id: id)
            }
        }
        return nil
    }

    static func embedURL(for embed: BoardVideoEmbed) -> URL? {
        switch embed.provider {
        case "youtube":
            return URL(string: "https://www.youtube.com/embed/\(embed.id)")
        case "vimeo":
            return URL(string: "https://player.vimeo.com/video/\(embed.id)")
        default:
            return nil
        }
    }

    static func formatFileSize(_ bytes: Int64) -> String {
        let formatter = ByteCountFormatter()
        formatter.countStyle = .file
        return formatter.string(fromByteCount: bytes)
    }

    static func parseDrawingElements(_ data: JSONValue?) -> [WhiteboardElement] {
        guard let data else { return [] }
        let encoded: Data
        do {
            encoded = try JSONEncoder().encode(data)
        } catch {
            return []
        }
        if let list = try? JSONDecoder().decode([WhiteboardElement].self, from: encoded) {
            return list
        }
        // Some payloads wrap strokes: { "elements": [...] }
        if case let .object(obj) = data, let elements = obj["elements"] {
            return parseDrawingElements(elements)
        }
        return []
    }

    static func makeTextBody(_ text: String) -> BoardPostBody {
        let trimmed = text.trimmingCharacters(in: .whitespacesAndNewlines)
        let escaped = trimmed
            .replacingOccurrences(of: "&", with: "&amp;")
            .replacingOccurrences(of: "<", with: "&lt;")
            .replacingOccurrences(of: ">", with: "&gt;")
        return BoardPostBody(html: "<p>\(escaped)</p>", text: trimmed)
    }

    private static func stripHTML(_ html: String) -> String {
        var result = html
        if let regex = try? NSRegularExpression(pattern: "<[^>]+>", options: .caseInsensitive) {
            let range = NSRange(result.startIndex..., in: result)
            result = regex.stringByReplacingMatches(in: result, options: [], range: range, withTemplate: "")
        }
        return result
            .replacingOccurrences(of: "&nbsp;", with: " ")
            .replacingOccurrences(of: "&amp;", with: "&")
            .replacingOccurrences(of: "&lt;", with: "<")
            .replacingOccurrences(of: "&gt;", with: ">")
            .trimmingCharacters(in: .whitespacesAndNewlines)
    }
}
