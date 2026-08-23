import UIKit

final class AppDelegate: NSObject, UIApplicationDelegate {
    static var push: PushManager?

    func application(_ application: UIApplication, didRegisterForRemoteNotificationsWithDeviceToken deviceToken: Data) {
        Self.push?.didRegister(deviceToken: deviceToken)
    }

    func application(_ application: UIApplication, didFailToRegisterForRemoteNotificationsWithError error: any Error) {
        Self.push?.didFailToRegister(error)
    }
}
