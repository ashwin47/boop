import SwiftUI

/// Expandable JSON viewer for unrecognised event data.
struct JSONTreeView: View {
    let name: String?
    let value: JSONValue
    var depth = 0
    @State private var expanded: Bool

    init(name: String? = nil, value: JSONValue, depth: Int = 0, expanded: Bool? = nil) {
        self.name = name
        self.value = value
        self.depth = depth
        _expanded = State(initialValue: expanded ?? (depth < 1))
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 2) {
            if value.isContainer {
                Button {
                    withAnimation(DS.Motion.snappy) { expanded.toggle() }
                } label: {
                    line {
                        Image(systemName: "chevron.right")
                            .font(.system(size: 9, weight: .bold))
                            .foregroundStyle(DS.Colors.textInactive)
                            .rotationEffect(.degrees(expanded ? 90 : 0))
                            .frame(width: 10)
                        if let name { key(name) }
                        Text(value.display).font(DS.Text.code).foregroundStyle(DS.Colors.textMuted)
                    }
                }
                .buttonStyle(.plain)
                if expanded {
                    ForEach(children, id: \.0) { k, v in
                        JSONTreeView(name: k, value: v, depth: depth + 1)
                    }
                    .transition(.opacity.combined(with: .move(edge: .top)))
                }
            } else {
                line {
                    Spacer().frame(width: 10)
                    if let name { key(name) }
                    Text(scalarText)
                        .font(DS.Text.code)
                        .foregroundStyle(scalarColor)
                        .textSelection(.enabled)
                }
            }
        }
    }

    private var children: [(String, JSONValue)] {
        if let o = value.objectValue { return o.sorted { $0.key < $1.key } }
        if let a = value.arrayValue { return a.enumerated().map { (String($0.offset), $0.element) } }
        return []
    }

    private var scalarText: String {
        if case .string(let s) = value { return "\"\(s)\"" }
        return value.display
    }

    private var scalarColor: Color {
        switch value {
        case .number, .bool: DS.Colors.accentHover
        case .null: DS.Colors.textMuted
        default: DS.Colors.ink
        }
    }

    @ViewBuilder
    private func line<C: View>(@ViewBuilder _ c: () -> C) -> some View {
        HStack(alignment: .firstTextBaseline, spacing: 6) { c() }
            .padding(.leading, CGFloat(depth) * 14)
            .padding(.vertical, 2)
            .frame(maxWidth: .infinity, alignment: .leading)
    }

    private func key(_ k: String) -> some View {
        HStack(spacing: 2) {
            Text(k).font(DS.Text.code).foregroundStyle(DS.Colors.textSecondary)
            Text(":").font(DS.Text.code).foregroundStyle(DS.Colors.textFaint)
        }
    }
}
