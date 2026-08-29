import SwiftUI

@MainActor
@Observable
final class InboxModel {
    var events: [Event] = []
    var cursor: String?
    var loading = false
    var loadingMore = false
    var error: String?
    var level: Level?
    var project: String?
    /// Collapse repeats of the same fingerprint into one row. Remembered across launches.
    var grouped: Bool {
        didSet { UserDefaults.standard.set(grouped, forKey: "boop.inbox.grouped") }
    }
    private(set) var loadedOnce = false

    init() {
        grouped = UserDefaults.standard.object(forKey: "boop.inbox.grouped") as? Bool ?? true
    }

    var projects: [Project] {
        var seen = Set<String>()
        return events.compactMap { e in
            guard seen.insert(e.projectID).inserted else { return nil }
            return Project(id: e.projectID, name: e.projectName, slug: e.projectSlug, icon: e.projectIcon)
        }.sorted { $0.name < $1.name }
    }

    struct Group: Identifiable {
        let label: String
        let events: [Event]
        var id: String { label }
    }

    var groups: [Group] {
        var out: [Group] = []
        for e in events {
            let label = Formatting.dayGroup(e.createdAt)
            if let last = out.last, last.label == label {
                out[out.count - 1] = Group(label: label, events: last.events + [e])
            } else {
                out.append(Group(label: label, events: [e]))
            }
        }
        return out
    }

    func refresh(_ client: APIClient) async {
        loading = true
        error = nil
        defer { loading = false; loadedOnce = true }
        do {
            let page = try await client.events(project: project, level: level, grouped: grouped)
            withAnimation(DS.Motion.settle) {
                events = page.events
                cursor = page.nextCursor
            }
        } catch {
            self.error = (error as? APIError)?.message ?? error.localizedDescription
        }
    }

    func loadMore(_ client: APIClient) async {
        guard let cursor, !loadingMore else { return }
        loadingMore = true
        defer { loadingMore = false }
        do {
            let page = try await client.events(project: project, level: level, grouped: grouped, before: cursor)
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

struct InboxView: View {
    @Environment(AppSession.self) private var session
    @Binding var selected: EventRoute?
    @State private var model = InboxModel()
    @State private var path: [GroupRoute] = []

    var body: some View {
        NavigationStack(path: $path) {
            list
                .background(DS.Colors.bg)
                .navigationTitle("Inbox")
                .navigationBarTitleDisplayMode(.large)
                .toolbar { toolbar }
                .sheet(item: $selected) { route in
                    EventDetailView(route: route)
                        .presentationDetents([.fraction(0.75), .large])
                        .presentationDragIndicator(.visible)
                        .presentationBackground(DS.Colors.bg)
                        .presentationCornerRadius(DS.Radius.card + 10)
                }
                .refreshable { await reload() }
                .task { if !model.loadedOnce { await reload() } }
                .onChange(of: model.level) { Task { await reload() } }
                .onChange(of: model.project) { Task { await reload() } }
                .onChange(of: model.grouped) { Task { await reload() } }
                .navigationDestination(for: GroupRoute.self) { route in
                    GroupView(route: route, selected: $selected)
                }
        }
    }

    @ViewBuilder
    private var list: some View {
        if model.loading && model.events.isEmpty {
            skeleton
        } else if let error = model.error, model.events.isEmpty {
            VStack(spacing: DS.Space.s4) {
                EmptyState(title: "Can't reach your server", message: error, systemImage: "wifi.exclamationmark")
                Button("Retry") { Task { await reload() } }.buttonStyle(.boopGhost)
            }
        } else if model.events.isEmpty {
            EmptyState(
                title: model.level == nil && model.project == nil ? "No events yet" : "Nothing matches",
                message: model.level == nil && model.project == nil
                    ? "Send one with curl or a client and it will show up here."
                    : "Try clearing the filters.",
                systemImage: "tray"
            )
        } else {
            List {
                ForEach(model.groups) { group in
                    Section {
                        ForEach(group.events) { event in
                            Button {
                                if event.isRepeated {
                                    path.append(GroupRoute(event: event))
                                } else {
                                    selected = EventRoute(id: event.id, summary: nil)
                                }
                            } label: {
                                EventRow(event: event)
                            }
                            .buttonStyle(RowButtonStyle())
                            .listRowInsets(EdgeInsets(top: 0, leading: DS.Space.s4, bottom: 0, trailing: DS.Space.s4))
                            .listRowSeparatorTint(DS.Colors.borderHairline)
                            .listRowBackground(DS.Colors.bg)
                            .transition(.opacity.combined(with: .move(edge: .top)))
                        }
                    } header: {
                        Text(group.label)
                            .font(DS.Text.caption)
                            .foregroundStyle(DS.Colors.textFaint)
                            .textCase(nil)
                    }
                }
                if model.cursor != nil {
                    HStack {
                        Spacer()
                        ProgressView().tint(DS.Colors.accent)
                        Spacer()
                    }
                    .listRowBackground(DS.Colors.bg)
                    .listRowSeparator(.hidden)
                    .task { if let c = session.client { await model.loadMore(c) } }
                }
            }
            .listStyle(.plain)
            .scrollContentBackground(.hidden)
            .animation(DS.Motion.settle, value: model.events)
        }
    }

    private var skeleton: some View {
        VStack(spacing: 0) {
            ForEach(0..<6, id: \.self) { i in
                HStack(spacing: 12) {
                    RoundedRectangle(cornerRadius: 4).fill(DS.Colors.borderHairline).frame(width: 60, height: 12)
                    VStack(alignment: .leading, spacing: 6) {
                        RoundedRectangle(cornerRadius: 4).fill(DS.Colors.borderHairline).frame(width: 160 + CGFloat(i % 3) * 30, height: 14)
                        RoundedRectangle(cornerRadius: 4).fill(DS.Colors.borderHairline).frame(width: 220, height: 11)
                    }
                    Spacer()
                }
                .padding(.horizontal, DS.Space.s4)
                .padding(.vertical, 14)
            }
            Spacer()
        }
        .modifier(Shimmer())
    }

    @ToolbarContentBuilder
    private var toolbar: some ToolbarContent {
        ToolbarItem(placement: .topBarTrailing) {
            Menu {
                Toggle("Group repeats", systemImage: "square.stack.3d.up", isOn: $model.grouped)
                Picker("Level", selection: $model.level) {
                    Text("All levels").tag(Level?.none)
                    ForEach(Level.allCases) { l in
                        Label { Text(l.label) } icon: { Image(uiImage: IconShape.circle.image(l.strong, size: 14)) }
                            .tag(Level?.some(l))
                    }
                }
                if !model.projects.isEmpty {
                    Picker("Project", selection: $model.project) {
                        Text("All projects").tag(String?.none)
                        ForEach(model.projects) { p in
                            if let parsed = ProjectIcon.parse(p.icon) {
                                Label { Text(p.name) } icon: { Image(uiImage: parsed.shape.image(parsed.color, size: 16)) }
                                    .tag(String?.some(p.id))
                            } else {
                                Text(p.icon.isEmpty ? p.name : "\(p.icon) \(p.name)").tag(String?.some(p.id))
                            }
                        }
                    }
                }
            } label: {
                Image(systemName: model.level != nil || model.project != nil ? "line.3.horizontal.decrease.circle.fill" : "line.3.horizontal.decrease.circle")
            }
            .tint(DS.Colors.accent)
        }
    }

    private func reload() async {
        guard let client = session.client else { return }
        await model.refresh(client)
    }
}

struct EventRow: View {
    let event: Event

    var body: some View {
        HStack(alignment: .top, spacing: 12) {
            VStack(alignment: .leading, spacing: 3) {
                HStack(spacing: 6) {
                    ProjectIcon(icon: event.projectIcon, size: 12)
                    Text(event.projectName).font(DS.Text.caption).foregroundStyle(DS.Colors.textSecondary)
                    Text("·").foregroundStyle(DS.Colors.textFaint).font(DS.Text.caption)
                    LevelBadge(level: event.level)
                    if event.silenced {
                        Text("· silenced").font(DS.Text.caption).foregroundStyle(DS.Colors.textInactive)
                    }
                }
                HStack(spacing: 8) {
                    Text(event.title)
                        .font(DS.Text.rowTitle)
                        .foregroundStyle(DS.Colors.ink)
                        .lineLimit(1)
                    if let g = event.group, g.count > 1 {
                        CountPill(count: g.count)
                    }
                }
                if let g = event.group, g.count > 1 {
                    Text(Formatting.seenRange(g))
                        .font(DS.Text.meta)
                        .foregroundStyle(DS.Colors.textSecondary)
                        .lineLimit(1)
                } else if !event.body.isEmpty {
                    Text(event.body)
                        .font(DS.Text.meta)
                        .foregroundStyle(DS.Colors.textMuted)
                        .lineLimit(2)
                }
            }
            Spacer(minLength: 8)
            VStack(alignment: .trailing, spacing: 6) {
                Text(Formatting.relative(event.createdAt))
                    .font(DS.Text.caption)
                    .foregroundStyle(DS.Colors.textMuted)
                    .monospacedDigit()
                if event.isRepeated {
                    Image(systemName: "chevron.right")
                        .font(.system(size: 11, weight: .semibold))
                        .foregroundStyle(DS.Colors.textFaint)
                        .accessibilityHidden(true)
                }
            }
        }
        .padding(.vertical, 12)
        .contentShape(Rectangle())
    }
}

/// "×47" pill for grouped rows.
struct CountPill: View {
    let count: Int

    var body: some View {
        Text("×\(count)")
            .font(DS.Text.small)
            .foregroundStyle(DS.Colors.accentHover)
            .padding(.horizontal, 8)
            .padding(.vertical, 1)
            .background(DS.Colors.accentTint, in: Capsule())
            .monospacedDigit()
    }
}

/// Rows tint to bg-hover on press, no scale.
struct RowButtonStyle: ButtonStyle {
    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .background(configuration.isPressed ? DS.Colors.bgHover : .clear, in: RoundedRectangle(cornerRadius: DS.Radius.control, style: .continuous))
            .animation(DS.Motion.gentle, value: configuration.isPressed)
    }
}

/// Soft shimmer for skeleton rows.
struct Shimmer: ViewModifier {
    @State private var phase: CGFloat = -1

    func body(content: Content) -> some View {
        content
            .overlay {
                LinearGradient(colors: [.clear, DS.Colors.bg.opacity(0.8), .clear], startPoint: .leading, endPoint: .trailing)
                    .offset(x: phase * 400)
                    .blendMode(.plusLighter)
            }
            .mask(content)
            .onAppear {
                withAnimation(.linear(duration: 1.3).repeatForever(autoreverses: false)) { phase = 1.5 }
            }
    }
}
