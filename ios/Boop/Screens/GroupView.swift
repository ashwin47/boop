import SwiftUI

/// Navigation target for the occurrences of one fingerprint within a project.
struct GroupRoute: Hashable {
    let projectID: String
    let projectName: String
    let projectIcon: String
    let fingerprint: String
    let title: String

    init(event: Event) {
        projectID = event.projectID
        projectName = event.projectName
        projectIcon = event.projectIcon
        fingerprint = event.fingerprint
        title = event.title
    }
}

@MainActor
@Observable
final class GroupModel {
    var head: Event?
    var events: [Event] = []
    var cursor: String?
    var loading = false
    var loadingMore = false
    var error: String?

    func refresh(_ client: APIClient, route: GroupRoute) async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            async let headPage = client.events(project: route.projectID, fingerprint: route.fingerprint, grouped: true, limit: 1)
            async let page = client.events(project: route.projectID, fingerprint: route.fingerprint)
            head = try await headPage.events.first
            let p = try await page
            withAnimation(DS.Motion.settle) {
                events = p.events
                cursor = p.nextCursor
            }
        } catch {
            self.error = (error as? APIError)?.message ?? error.localizedDescription
        }
    }

    func loadMore(_ client: APIClient, route: GroupRoute) async {
        guard let cursor, !loadingMore else { return }
        loadingMore = true
        defer { loadingMore = false }
        do {
            let page = try await client.events(project: route.projectID, fingerprint: route.fingerprint, before: cursor)
            let known = Set(events.map(\.id))
            withAnimation(DS.Motion.settle) {
                events += page.events.filter { !known.contains($0.id) }
                self.cursor = page.nextCursor
            }
        } catch {
            self.error = (error as? APIError)?.message ?? error.localizedDescription
        }
    }
}

/// Every occurrence of a fingerprint, newest first. Pushed from a grouped inbox row.
struct GroupView: View {
    @Environment(AppSession.self) private var session
    let route: GroupRoute
    @Binding var selected: EventRoute?
    @State private var model = GroupModel()

    var body: some View {
        List {
            Section {
                header
                    .listRowInsets(EdgeInsets(top: 4, leading: DS.Space.s4, bottom: 8, trailing: DS.Space.s4))
                    .listRowSeparator(.hidden)
                    .listRowBackground(DS.Colors.bg)
            }
            Section {
                if let error = model.error, model.events.isEmpty {
                    VStack(spacing: DS.Space.s3) {
                        Notice(tone: .bad, text: error)
                        Button("Retry") { Task { await reload() } }.buttonStyle(.boopGhost)
                    }
                    .listRowBackground(DS.Colors.bg)
                    .listRowSeparator(.hidden)
                } else if model.events.isEmpty && !model.loading {
                    EmptyState(title: "No occurrences", message: "Nothing with this fingerprint in this project.", systemImage: "square.stack.3d.up.slash")
                        .listRowBackground(DS.Colors.bg)
                        .listRowSeparator(.hidden)
                }
                ForEach(model.events) { event in
                    Button {
                        selected = EventRoute(id: event.id, summary: nil)
                    } label: {
                        OccurrenceRow(event: event)
                    }
                    .buttonStyle(RowButtonStyle())
                    .listRowInsets(EdgeInsets(top: 0, leading: DS.Space.s4, bottom: 0, trailing: DS.Space.s4))
                    .listRowSeparatorTint(DS.Colors.borderHairline)
                    .listRowBackground(DS.Colors.bg)
                }
                if model.cursor != nil {
                    HStack {
                        Spacer()
                        ProgressView().tint(DS.Colors.accent)
                        Spacer()
                    }
                    .listRowBackground(DS.Colors.bg)
                    .listRowSeparator(.hidden)
                    .task { if let c = session.client { await model.loadMore(c, route: route) } }
                }
            } header: {
                Text("Occurrences")
                    .font(DS.Text.caption)
                    .foregroundStyle(DS.Colors.textFaint)
                    .textCase(nil)
            }
        }
        .listStyle(.plain)
        .scrollContentBackground(.hidden)
        .background(DS.Colors.bg)
        .navigationTitle(route.title)
        .navigationBarTitleDisplayMode(.inline)
        .refreshable { await reload() }
        .task { await reload() }
    }

    private var header: some View {
        Card {
            VStack(alignment: .leading, spacing: 8) {
                HStack(spacing: 8) {
                    ProjectIcon(icon: route.projectIcon, size: 14)
                    Text(route.projectName).font(DS.Text.meta).foregroundStyle(DS.Colors.textSecondary)
                    if let h = model.head {
                        Text("·").foregroundStyle(DS.Colors.textFaint)
                        LevelBadge(level: h.level)
                    }
                }
                HStack(alignment: .firstTextBaseline, spacing: 8) {
                    Text(route.title).font(DS.Text.metric).foregroundStyle(DS.Colors.ink)
                    if let g = model.head?.group { CountPill(count: g.count) }
                }
                if let g = model.head?.group {
                    Text(Formatting.seenRange(g)).font(DS.Text.statusLine).foregroundStyle(DS.Colors.textSecondary)
                }
                Rectangle().fill(DS.Colors.borderHairline).frame(height: 1).padding(.vertical, 6)
                LazyVGrid(columns: [GridItem(.flexible()), GridItem(.flexible())], alignment: .leading, spacing: 12) {
                    Fact(label: "Fingerprint", value: route.fingerprint, mono: true)
                    if let g = model.head?.group {
                        Fact(label: "First seen", value: Formatting.full(g.firstSeen))
                        Fact(label: "Last seen", value: Formatting.full(g.lastSeen))
                    }
                }
            }
        }
    }

    private func reload() async {
        guard let client = session.client else { return }
        await model.refresh(client, route: route)
    }
}

struct OccurrenceRow: View {
    let event: Event

    var body: some View {
        HStack(alignment: .top, spacing: 12) {
            VStack(alignment: .leading, spacing: 3) {
                HStack(spacing: 6) {
                    LevelBadge(level: event.level)
                    if event.silenced {
                        Text("· silenced").font(DS.Text.caption).foregroundStyle(DS.Colors.textInactive)
                    }
                }
                Text(event.title).font(DS.Text.rowTitle).foregroundStyle(DS.Colors.ink).lineLimit(1)
                if !event.body.isEmpty {
                    Text(event.body).font(DS.Text.meta).foregroundStyle(DS.Colors.textMuted).lineLimit(2)
                }
            }
            Spacer(minLength: 8)
            VStack(alignment: .trailing, spacing: 3) {
                Text(Formatting.relative(event.createdAt)).font(DS.Text.caption).foregroundStyle(DS.Colors.textMuted).monospacedDigit()
                Text(Formatting.clock(event.createdAt)).font(DS.Text.caption).foregroundStyle(DS.Colors.textFaint).monospacedDigit()
            }
        }
        .padding(.vertical, 12)
        .contentShape(Rectangle())
    }
}
