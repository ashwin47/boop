import Foundation
import Observation
import UIKit
import UserNotifications

/// Requests permission, registers with APNs, and turns notification taps into navigation.
@MainActor
@Observable
final class PushManager: NSObject, UNUserNotificationCenterDelegate {
    var authorization: UNAuthorizationStatus = .notDetermined
    /// Event id from the most recently tapped notification; the root view consumes it.
    var pendingEventID: String?
    /// Title/body from the tapped push, shown if the server cannot be reached.
    var pendingSummary: (title: String, body: String)?
    var lastError: String?

    private weak var session: AppSession?

    func attach(session: AppSession) {
        self.session = session
        UNUserNotificationCenter.current().delegate = self
    }

    func refreshAuthorization() async {
        authorization = await UNUserNotificationCenter.current().notificationSettings().authorizationStatus
    }

    /// Ask for permission (first time) and register with APNs. Safe to call on every launch.
    func requestAndRegister() async {
        do {
            let granted = try await UNUserNotificationCenter.current().requestAuthorization(options: [.alert, .sound, .badge])
            await refreshAuthorization()
            if granted {
                UIApplication.shared.registerForRemoteNotifications()
            }
        } catch {
            lastError = error.localizedDescription
        }
    }

    func didRegister(deviceToken: Data) {
        let hex = deviceToken.map { String(format: "%02x", $0) }.joined()
        lastError = nil
        Task { [weak self] in
            guard let self, let session = self.session else { return }
            do {
                try await session.registerAPNSToken(hex)
            } catch {
                self.lastError = (error as? APIError)?.message ?? error.localizedDescription
            }
        }
    }

    func didFailToRegister(_ error: any Error) {
        // Expected in the simulator and on builds without the push entitlement.
        lastError = error.localizedDescription
    }

    // MARK: UNUserNotificationCenterDelegate

    nonisolated func userNotificationCenter(_ center: UNUserNotificationCenter, willPresent notification: UNNotification) async -> UNNotificationPresentationOptions {
        [.banner, .sound, .list]
    }

    nonisolated func userNotificationCenter(_ center: UNUserNotificationCenter, didReceive response: UNNotificationResponse) async {
        let info = response.notification.request.content.userInfo
        let eventID = info["event_id"] as? String
        let title = response.notification.request.content.title
        let body = response.notification.request.content.body
        await MainActor.run {
            self.pendingSummary = (title, body)
            self.pendingEventID = eventID
        }
    }
}
