import SwiftUI

/// Presented as a 3/4-height sheet from the inbox (and from a notification tap).
struct EventDetailView: View {
    @Environment(AppSession.self) private var session
    @Environment(\.dismiss) private var dismiss
    let route: EventRoute

    @State private var event: Event?
    @State private var error: String?
    @State private var showRaw = false

    var body: some View {
        NavigationStack {
            content
                .navigationTitle(event?.projectName ?? "Event")
                .navigationBarTitleDisplayMode(.inline)
                .toolbar {
                    ToolbarItem(placement: .topBarLeading) {
                        Button("Close", systemImage: "xmark") { dismiss() }
                    }
                    if let event {
                        ToolbarItem(placement: .topBarTrailing) {
                            ShareLink(item: event.data.pretty, subject: Text(event.title)) {
                                Image(systemName: "square.and.arrow.up")
                            }
                        }
                    }
                }
        }
    }

    private var content: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: DS.Space.s4) {
                if let event {
                    if event.silenced {
                        Notice(tone: .info, text: "Matched a silence rule, so it was not pushed. Manage silences in the web UI.")
                    }
                    summary(event)
                    sections(event)
                    raw(event)
                } else if let error {
                    offline(error)
                } else {
                    loading
                }
            }
            .padding(DS.Space.s4)
        }
        .background(DS.Colors.bg)
        .task(id: route.id) { await load() }
    }

    // MARK: Pieces

    private func summary(_ e: Event) -> some View {
        Card {
            VStack(alignment: .leading, spacing: 8) {
                HStack(spacing: 8) {
                    ProjectIcon(icon: e.projectIcon, size: 14)
                    Text(e.projectName).font(DS.Text.meta).foregroundStyle(DS.Colors.textSecondary)
                    Text("·").foregroundStyle(DS.Colors.textFaint)
                    LevelBadge(level: e.level)
                    if !e.source.isEmpty {
                        Text("·").foregroundStyle(DS.Colors.textFaint)
                        Text(e.source).font(DS.Text.meta).foregroundStyle(DS.Colors.textMuted)
                    }
                }
                Text(e.title)
                    .font(DS.Text.metric)
                    .foregroundStyle(DS.Colors.ink)
                    .textSelection(.enabled)
                if !e.body.isEmpty {
                    Text(e.body)
                        .font(DS.Text.statusLine)
                        .foregroundStyle(DS.Colors.textSecondary)
                        .textSelection(.enabled)
                }
                Rectangle().fill(DS.Colors.borderHairline).frame(height: 1).padding(.vertical, 6)
                LazyVGrid(columns: [GridItem(.flexible()), GridItem(.flexible())], alignment: .leading, spacing: 12) {
                    Fact(label: "Occurred", value: Formatting.full(e.occurredAt))
                    Fact(label: "Received", value: Formatting.full(e.createdAt))
                    if !e.type.isEmpty { Fact(label: "Type", value: e.type) }
                    if !e.fingerprint.isEmpty { Fact(label: "Fingerprint", value: e.fingerprint, mono: true) }
                    if let ext = e.externalID, !ext.isEmpty { Fact(label: "External id", value: ext, mono: true) }
                    Fact(label: "Event id", value: e.id, mono: true)
                }
            }
        }
    }

    @ViewBuilder
    private func sections(_ e: Event) -> some View {
        let s = e.sections
        if let exc = s.exception {
            Card(title: "Exception") {
                VStack(alignment: .leading, spacing: 6) {
                    if let t = exc["type"]?.display { Text(t).font(DS.Text.rowTitle).foregroundStyle(DS.Colors.ink) }
                    if let m = exc["message"]?.display {
                        Text(m).font(DS.Text.code).foregroundStyle(DS.Colors.textSecondary).textSelection(.enabled)
                    }
                    ForEach(exc.keys.filter { $0 != "type" && $0 != "message" }.sorted(), id: \.self) { k in
                        JSONTreeView(name: k, value: exc[k]!, depth: 0, expanded: false)
                    }
                }
            }
        }
        if let frames = s.frames {
            Card(title: "Stacktrace") {
                VStack(spacing: DS.Space.s1) {
                    ForEach(frames) { f in
                        StackFrameRow(frame: f)
                    }
                }
            }
        }
        if let tags = s.tags, !tags.isEmpty {
            Card(title: "Tags") {
                FlowLayout(spacing: 8) {
                    ForEach(tags, id: \.0) { k, v in
                        HStack(spacing: 5) {
                            Text(k).font(DS.Text.small).fontWeight(.medium).foregroundStyle(DS.Colors.textMuted)
                            Text(v.display).font(DS.Text.small).foregroundStyle(DS.Colors.ink)
                        }
                        .padding(.horizontal, 10)
                        .padding(.vertical, 5)
                        .background(DS.Colors.bgHover, in: Capsule())
                        .overlay(Capsule().stroke(DS.Colors.ink.opacity(0.06)))
                    }
                }
            }
        }
        if let ctx = s.context, !ctx.isEmpty {
            Card(title: "Context") {
                VStack(alignment: .leading, spacing: 10) {
                    ForEach(ctx, id: \.0) { k, v in
                        if v.isContainer {
                            JSONTreeView(name: k, value: v, depth: 0, expanded: true)
                        } else {
                            HStack(alignment: .firstTextBaseline, spacing: 12) {
                                Text(k).font(DS.Text.caption).foregroundStyle(DS.Colors.textMuted).frame(width: 110, alignment: .leading)
                                Text(v.display).font(DS.Text.code).foregroundStyle(DS.Colors.ink).textSelection(.enabled)
                            }
                        }
                    }
                }
            }
        }
        if let bcs = s.breadcrumbs, !bcs.isEmpty {
            Card(title: "Breadcrumbs") {
                VStack(alignment: .leading, spacing: 8) {
                    ForEach(bcs) { b in
                        HStack(alignment: .firstTextBaseline, spacing: 10) {
                            Text(b.timestamp).font(DS.Text.caption).foregroundStyle(DS.Colors.textMuted).frame(width: 64, alignment: .leading)
                            Text(b.category).font(DS.Text.caption).foregroundStyle(DS.Colors.textSecondary).frame(width: 80, alignment: .leading)
                            Text(b.message).font(DS.Text.meta).foregroundStyle(DS.Colors.ink)
                        }
                    }
                }
            }
        }
        if !s.rest.isEmpty {
            Card(title: "Data") {
                VStack(alignment: .leading, spacing: 2) {
                    ForEach(s.rest, id: \.0) { k, v in
                        JSONTreeView(name: k, value: v, depth: 0, expanded: true)
                    }
                }
            }
        }
    }

    private func raw(_ e: Event) -> some View {
        Card {
            HStack {
                Text("Raw JSON").font(DS.Text.setting).foregroundStyle(DS.Colors.ink)
                Spacer()
                Button(showRaw ? "Hide" : "Show") { withAnimation(DS.Motion.snappy) { showRaw.toggle() } }
                    .buttonStyle(.boopGhost)
            }
            if showRaw {
                CodeBlock(text: e.data.pretty)
                    .transition(.opacity.combined(with: .move(edge: .top)))
            }
        }
    }

    private var loading: some View {
        VStack(spacing: 12) {
            if let s = route.summary {
                Card {
                    Text(s.title).font(DS.Text.metric).foregroundStyle(DS.Colors.ink)
                    if !s.body.isEmpty { Text(s.body).font(DS.Text.statusLine).foregroundStyle(DS.Colors.textSecondary) }
                }
            }
            ProgressView().tint(DS.Colors.accent).padding(.top, DS.Space.s5)
        }
    }

    private func offline(_ message: String) -> some View {
        VStack(spacing: DS.Space.s4) {
            if let s = route.summary {
                Card {
                    Text(s.title).font(DS.Text.metric).foregroundStyle(DS.Colors.ink)
                    if !s.body.isEmpty { Text(s.body).font(DS.Text.statusLine).foregroundStyle(DS.Colors.textSecondary) }
                    Text("From the notification · full details unavailable")
                        .font(DS.Text.caption).foregroundStyle(DS.Colors.textMuted)
                }
            }
            Notice(tone: .bad, text: message)
            Button("Retry") { Task { await load() } }.buttonStyle(.boopSecondary)
        }
    }

    private func load() async {
        guard let client = session.client else { return }
        error = nil
        do {
            let e = try await client.event(id: route.id)
            event = e
        } catch {
            self.error = (error as? APIError)?.message ?? error.localizedDescription
        }
    }
}

struct StackFrameRow: View {
    let frame: StackFrame

    var body: some View {
        VStack(alignment: .leading, spacing: 2) {
            Text(frame.function).font(DS.Text.ui).foregroundStyle(DS.Colors.ink)
            if !frame.file.isEmpty {
                HStack(spacing: 0) {
                    Text(frame.file).font(DS.Text.code).foregroundStyle(DS.Colors.textSecondary)
                    if let line = frame.line {
                        Text(":").font(DS.Text.code).foregroundStyle(DS.Colors.textFaint)
                        Text(line).font(DS.Text.code).foregroundStyle(DS.Colors.textSecondary)
                    }
                }
                .lineLimit(2)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(.horizontal, 12)
        .padding(.vertical, 8)
        .background(frame.inApp ? DS.Colors.accentTint : DS.Colors.bgHover, in: RoundedRectangle(cornerRadius: DS.Radius.row, style: .continuous))
        .overlay {
            if frame.inApp {
                RoundedRectangle(cornerRadius: DS.Radius.row, style: .continuous).stroke(DS.Colors.ink.opacity(0.06))
            }
        }
    }
}

/// Wrapping horizontal layout for tag pills.
struct FlowLayout: Layout {
    var spacing: CGFloat = 8

    func sizeThatFits(proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) -> CGSize {
        let width = proposal.width ?? .infinity
        var x: CGFloat = 0, y: CGFloat = 0, rowH: CGFloat = 0
        for s in subviews {
            let size = s.sizeThatFits(.unspecified)
            if x + size.width > width, x > 0 { x = 0; y += rowH + spacing; rowH = 0 }
            x += size.width + spacing
            rowH = max(rowH, size.height)
        }
        return CGSize(width: width == .infinity ? x : width, height: y + rowH)
    }

    func placeSubviews(in bounds: CGRect, proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) {
        var x = bounds.minX, y = bounds.minY, rowH: CGFloat = 0
        for s in subviews {
            let size = s.sizeThatFits(.unspecified)
            if x + size.width > bounds.maxX, x > bounds.minX { x = bounds.minX; y += rowH + spacing; rowH = 0 }
            s.place(at: CGPoint(x: x, y: y), proposal: .unspecified)
            x += size.width + spacing
            rowH = max(rowH, size.height)
        }
    }
}
