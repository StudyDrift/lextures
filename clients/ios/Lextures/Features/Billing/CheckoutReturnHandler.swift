import SwiftUI

/// Polls entitlement after Stripe checkout success and routes into the purchased course (M9.2).
struct CheckoutReturnOverlay: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme

    let phase: CheckoutReturnPhase

    @State private var status: Status = .verifying
    @State private var message: String?

    enum Status {
        case verifying
        case ready
        case timeout
        case cancelled
    }

    var body: some View {
        ZStack {
            Color.black.opacity(0.35).ignoresSafeArea()
            LMSCard {
                VStack(spacing: 12) {
                    switch status {
                    case .verifying:
                        ProgressView()
                        Text(L.text("mobile.billing.verifyingPayment"))
                            .font(.headline)
                        Text(L.text("mobile.billing.verifyingHint"))
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                            .multilineTextAlignment(.center)
                    case .ready:
                        Image(systemName: "checkmark.circle.fill")
                            .font(.largeTitle)
                            .foregroundStyle(LexturesTheme.primary)
                        Text(L.text("mobile.billing.paymentConfirmed"))
                            .font(.headline)
                        if let message {
                            Text(message)
                                .font(.caption)
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        }
                    case .timeout:
                        Image(systemName: "clock")
                            .font(.largeTitle)
                            .foregroundStyle(LexturesTheme.amber)
                        Text(L.text("mobile.billing.paymentProcessing"))
                            .font(.headline)
                        Text(L.text("mobile.billing.paymentProcessingHint"))
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                            .multilineTextAlignment(.center)
                    case .cancelled:
                        Image(systemName: "xmark.circle")
                            .font(.largeTitle)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        Text(L.text("mobile.billing.checkoutCancelled"))
                            .font(.headline)
                    }

                    if status != .verifying {
                        Button(L.text("mobile.common.close")) {
                            shell.checkoutReturnPhase = nil
                        }
                        .buttonStyle(.borderedProminent)
                        .tint(LexturesTheme.primary)
                    }
                }
                .frame(maxWidth: .infinity)
                .padding(.vertical, 8)
            }
            .padding(24)
        }
        .task { await run() }
    }

    private func run() async {
        switch phase {
        case .cancel:
            status = .cancelled
        case let .success(courseId):
            await verify(courseId: courseId)
        }
    }

    private func verify(courseId: String?) async {
        guard let token = session.accessToken else {
            status = .timeout
            return
        }

        let targetCourseId = courseId ?? shell.pendingCheckout?.courseId
        let courseCode = shell.pendingCheckout?.courseCode

        if targetCourseId == nil, courseCode == nil {
            status = .ready
            shell.pendingCheckout = nil
            return
        }

        guard let userId = shell.profile?.id else {
            status = .timeout
            return
        }

        for attempt in 1 ... BillingLogic.entitlementPollAttempts {
            if Task.isCancelled { return }
            do {
                if let targetCourseId,
                   try await LMSAPI.checkEntitlement(
                       userId: userId,
                       courseId: targetCourseId,
                       accessToken: token
                   ) {
                    if let courseCode {
                        let course = try await LMSAPI.fetchCourse(courseCode: courseCode, accessToken: token)
                        shell.activeCourse = course
                        shell.activeCourseRoot = .profile
                        shell.activeCourseSection = .modules
                        shell.select(.courses)
                        message = course.title
                    }
                    shell.pendingCheckout = nil
                    status = .ready
                    return
                }
            } catch {
                // keep polling
            }
            if attempt < BillingLogic.entitlementPollAttempts {
                try? await Task.sleep(nanoseconds: BillingLogic.entitlementPollIntervalSeconds * 1_000_000_000)
            }
        }
        status = .timeout
    }
}