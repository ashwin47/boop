import SwiftUI

@main
struct BoopApp: App {
    @UIApplicationDelegateAdaptor(AppDelegate.self) private var appDelegate
    @State private var session = AppSession()
    @State private var push = PushManager()

    init() {}

    var body: some Scene {
        WindowGroup {
            RootView()
                .environment(session)
                .environment(push)
                .tint(DS.Colors.accent)
                .preferredColorScheme(.light)
                .task {
                    push.attach(session: session)
                    AppDelegate.push = push
                    await push.refreshAuthorization()
                    if session.isPaired {
                        await push.requestAndRegister()
                        await session.checkConnection()
                    }
                }
        }
    }
}
