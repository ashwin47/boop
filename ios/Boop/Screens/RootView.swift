import SwiftUI

struct RootView: View {
    @Environment(AppSession.self) private var session
    @Environment(PushManager.self) private var push

    var body: some View {
        ZStack {
            DS.Colors.bg.ignoresSafeArea()
            if session.isPaired && !session.onboarding {
                MainTabs()
                    .transition(.opacity.combined(with: .scale(scale: 0.98)))
            } else {
                PairingView()
                    .transition(.opacity)
            }
        }
        .animation(DS.Motion.settle, value: session.isPaired && !session.onboarding)
    }
}

private struct MainTabs: View {
    @Environment(PushManager.self) private var push
    @State private var selected: EventRoute?
    @State private var tab: Section = .inbox

    enum Section: Hashable { case inbox, settings }

    var body: some View {
        TabView(selection: $tab) {
            Tab("Inbox", systemImage: "tray.full", value: Section.inbox) {
                InboxView(selected: $selected)
            }
            Tab("Settings", systemImage: "slider.horizontal.3", value: Section.settings) {
                SettingsView()
            }
        }
        .onChange(of: push.pendingEventID) { _, id in
            guard let id else { return }
            tab = .inbox
            selected = EventRoute(id: id, summary: push.pendingSummary)
            push.pendingEventID = nil
        }
    }
}

/// Navigation destination for an event, carrying the push title/body as a fallback.
struct EventRoute: Hashable, Identifiable {
    let id: String
    var summary: (title: String, body: String)?

    static func == (a: EventRoute, b: EventRoute) -> Bool { a.id == b.id }
    func hash(into hasher: inout Hasher) { hasher.combine(id) }
}
