import Foundation

/// Caps live sandbox WebViews per screen (NFR: at most 3). Pure bookkeeping; UI owns teardown.
@MainActor
final class SandboxWebViewPool {
    static let shared = SandboxWebViewPool()

    private var alive: [String] = []
    private let maxAlive: Int

    init(maxAlive: Int = ContentToolSandboxLogic.maxLiveWebViews) {
        self.maxAlive = maxAlive
    }

    var aliveCount: Int { alive.count }

    var aliveIds: [String] { alive }

    /// Registers a mount. Returns instance ids that must be torn down to stay within budget.
    @discardableResult
    func retain(_ instanceId: String) -> [String] {
        alive.removeAll { $0 == instanceId }
        alive.append(instanceId)
        var evicted: [String] = []
        while ContentToolSandboxLogic.poolShouldEvict(aliveCount: alive.count, maxAlive: maxAlive) {
            let victim = alive.removeFirst()
            if victim != instanceId {
                evicted.append(victim)
            }
        }
        return evicted
    }

    func release(_ instanceId: String) {
        alive.removeAll { $0 == instanceId }
    }

    func reset() {
        alive.removeAll()
    }
}
