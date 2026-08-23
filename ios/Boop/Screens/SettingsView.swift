import SwiftUI
import UserNotifications

struct SettingsView: View {
    @Environment(AppSession.self) private var session
    @Environment(PushManager.self) private var push
    @State private var name = ""
    @State private var confirmDisconnect = false
    @State private var rePair = false

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(spacing: DS.Space.s4) {
                    serverCard
                    deviceCard
                    notificationsCard
                    aboutCard
                }
                .padding(DS.Space.s4)
            }
            .background(DS.Colors.bg)
            .navigationTitle("Settings")
            .task {
                name = session.deviceName
                await push.refreshAuthorization()
                await session.checkConnection()
            }
            .refreshable {
                await push.refreshAuthorization()
                await session.checkConnection()
            }
            .confirmationDialog("Disconnect from this server?", isPresented: $confirmDisconnect, titleVisibility: .visible) {
                Button("Disconnect", role: .destructive) { Task { await session.disconnect() } }
                Button("Cancel", role: .cancel) {}
            } message: {
                Text("The device is removed from the server and pushes stop. You can pair again any time.")
            }
        }
    }

    private var serverCard: some View {
        Card(title: "Server") {
            VStack(alignment: .leading, spacing: 8) {
                Text(session.serverURL?.absoluteString ?? "—")
                    .font(DS.Text.code).foregroundStyle(DS.Colors.ink).textSelection(.enabled)
                connectionStatus
                    .animation(DS.Motion.gentle, value: session.connection)
            }
            SettingRow(label: "Re-pair", hint: "Scan a new code, for example after moving the server.") {
                Button("Re-pair") { session.forget() }.buttonStyle(BoopButtonStyle(variant: .secondary))
            }
            SettingRow(label: "Disconnect", hint: "Removes this phone from the server.") {
                Button("Disconnect") { confirmDisconnect = true }.buttonStyle(BoopButtonStyle(variant: .danger))
            }
        }
    }

    @ViewBuilder
    private var connectionStatus: some View {
        switch session.connection {
        case .unknown: StatusDot(tone: .muted, text: "Not checked")
        case .checking: StatusDot(tone: .muted, text: "Checking")
        case .connected: StatusDot(tone: .ok, text: "Connected")
        case .unauthorized: StatusDot(tone: .bad, text: "Credential rejected · re-pair")
        case .unreachable(let why): StatusDot(tone: .bad, text: "Unreachable · \(why)")
        }
    }

    private var deviceCard: some View {
        Card(title: "Device") {
            SettingRow(label: "Name", hint: "Shown in the web UI's device list.") {
                BoopTextField(placeholder: "iPhone", text: $name)
                    .frame(width: 170)
                    .onSubmit { Task { await session.rename(name) } }
            }
            SettingRow(label: "Push registration", hint: session.pushRegistered ? "APNs token is on the server." : "No APNs token yet. Push registration fails in the simulator.") {
                StatusDot(tone: session.pushRegistered ? .ok : .warn, text: session.pushRegistered ? "Registered" : "Not registered")
            }
            if let err = push.lastError {
                Text(err).font(DS.Text.caption).foregroundStyle(DS.Colors.textMuted)
            }
        }
    }

    private var notificationsCard: some View {
        Card(title: "Notifications") {
            SettingRow(label: "Permission") {
                switch push.authorization {
                case .authorized, .provisional, .ephemeral: StatusDot(tone: .ok, text: "Allowed")
                case .denied: StatusDot(tone: .bad, text: "Denied")
                case .notDetermined: StatusDot(tone: .warn, text: "Not asked")
                @unknown default: StatusDot(tone: .muted, text: "Unknown")
                }
            }
            if push.authorization == .denied {
                Button("Open iOS Settings") {
                    if let url = URL(string: UIApplication.openSettingsURLString) { UIApplication.shared.open(url) }
                }
                .buttonStyle(.boopSecondary)
            } else if push.authorization == .notDetermined {
                Button("Allow notifications") { Task { await push.requestAndRegister() } }
                    .buttonStyle(.boopPrimary)
            } else {
                Button("Re-register with APNs") { Task { await push.requestAndRegister() } }
                    .buttonStyle(.boopGhost)
            }
        }
    }

    private var aboutCard: some View {
        Card(title: "About") {
            LazyVGrid(columns: [GridItem(.flexible()), GridItem(.flexible())], alignment: .leading, spacing: 12) {
                Fact(label: "App version", value: Bundle.main.infoDictionary?["CFBundleShortVersionString"] as? String ?? "—")
                Fact(label: "Bundle id", value: Bundle.main.bundleIdentifier ?? "—", mono: true)
            }
            Text("Boop is open source. This build is signed with your own Apple team; the bundle id must match APNS_BUNDLE_ID on your server.")
                .font(DS.Text.caption).foregroundStyle(DS.Colors.textMuted)
        }
    }
}
