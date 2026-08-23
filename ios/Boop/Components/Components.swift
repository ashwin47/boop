import SwiftUI

/// Outlined card: hairline border, 14pt radius, paper fill, no shadow.
struct Card<Content: View>: View {
    var title: String?
    var padding: CGFloat = DS.Space.s4
    @ViewBuilder var content: Content

    var body: some View {
        VStack(alignment: .leading, spacing: DS.Space.s3) {
            if let title {
                Text(title).font(DS.Text.setting).foregroundStyle(DS.Colors.ink)
            }
            content
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(padding)
        .background(DS.Colors.bg, in: RoundedRectangle(cornerRadius: DS.Radius.card, style: .continuous))
        .overlay(RoundedRectangle(cornerRadius: DS.Radius.card, style: .continuous).stroke(DS.Colors.borderHairline, lineWidth: 1))
    }
}

/// 8pt dot + label, coloured by level.
struct LevelBadge: View {
    let level: Level
    var showLabel = true

    var body: some View {
        HStack(spacing: 6) {
            Circle()
                .fill(level.strong)
                .frame(width: 8, height: 8)
                .modifier(CriticalPulse(active: level == .critical))
            if showLabel {
                Text(level.label).font(DS.Text.small).foregroundStyle(level.strong)
            }
        }
    }
}

/// A slow, subtle pulse on critical dots only.
private struct CriticalPulse: ViewModifier {
    let active: Bool
    @State private var on = false

    func body(content: Content) -> some View {
        content
            .overlay {
                if active {
                    Circle()
                        .stroke(DS.Colors.levelCriticalStrong.opacity(on ? 0 : 0.6), lineWidth: 2)
                        .scaleEffect(on ? 2.4 : 1)
                        .animation(.easeOut(duration: 1.4).repeatForever(autoreverses: false), value: on)
                        .onAppear { on = true }
                }
            }
    }
}

/// Generic status line for settings.
struct StatusDot: View {
    enum Tone { case ok, warn, bad, muted }
    let tone: Tone
    let text: String

    var color: Color {
        switch tone {
        case .ok: DS.Colors.ok
        case .warn: DS.Colors.degradedStrong
        case .bad: DS.Colors.outageStrong
        case .muted: DS.Colors.textInactive
        }
    }

    var body: some View {
        HStack(spacing: 7) {
            Circle().fill(color).frame(width: 8, height: 8)
            Text(text).font(DS.Text.small).foregroundStyle(color)
        }
    }
}

/// Primary / secondary / ghost buttons on the 8pt control radius.
struct BoopButtonStyle: ButtonStyle {
    enum Variant { case primary, secondary, ghost, danger }
    var variant: Variant = .primary
    var expand = false

    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .font(DS.Text.ui)
            .foregroundStyle(foreground(pressed: configuration.isPressed))
            .padding(.horizontal, 16)
            .frame(maxWidth: expand ? .infinity : nil)
            .frame(height: 40)
            .background(background(pressed: configuration.isPressed), in: RoundedRectangle(cornerRadius: DS.Radius.control, style: .continuous))
            .overlay {
                if variant == .secondary || variant == .danger {
                    RoundedRectangle(cornerRadius: DS.Radius.control, style: .continuous).stroke(DS.Colors.borderControl, lineWidth: 1)
                }
            }
            .animation(DS.Motion.gentle, value: configuration.isPressed)
    }

    private func foreground(pressed: Bool) -> Color {
        switch variant {
        case .primary: DS.Colors.textOnDark
        case .secondary: DS.Colors.ink
        case .ghost: pressed ? DS.Colors.accentHover : DS.Colors.accent
        case .danger: pressed ? DS.Colors.outageStrong : DS.Colors.textSecondary
        }
    }

    private func background(pressed: Bool) -> Color {
        switch variant {
        case .primary: pressed ? DS.Colors.accentHover : DS.Colors.accent
        case .secondary, .danger: pressed ? DS.Colors.bgHover : DS.Colors.bg
        case .ghost: pressed ? DS.Colors.accentTint : .clear
        }
    }
}

extension ButtonStyle where Self == BoopButtonStyle {
    static var boopPrimary: BoopButtonStyle { BoopButtonStyle(variant: .primary, expand: true) }
    static var boopSecondary: BoopButtonStyle { BoopButtonStyle(variant: .secondary, expand: true) }
    static var boopGhost: BoopButtonStyle { BoopButtonStyle(variant: .ghost) }
    static var boopDanger: BoopButtonStyle { BoopButtonStyle(variant: .danger, expand: true) }
}

/// Dark code surface, radius 10.
struct CodeBlock: View {
    let text: String

    var body: some View {
        ScrollView(.horizontal, showsIndicators: false) {
            Text(text)
                .font(DS.Text.code)
                .foregroundStyle(DS.Colors.textOnDark)
                .textSelection(.enabled)
                .padding(14)
        }
        .background(DS.Colors.surfaceDark, in: RoundedRectangle(cornerRadius: DS.Radius.tooltip, style: .continuous))
    }
}

struct EmptyState: View {
    let title: String
    var message: String?
    var systemImage: String = "tray"

    var body: some View {
        VStack(spacing: 8) {
            Image(systemName: systemImage)
                .font(.system(size: 26, weight: .medium))
                .foregroundStyle(DS.Colors.textInactive)
                .padding(.bottom, 4)
            Text(title).font(DS.Text.setting).foregroundStyle(DS.Colors.textSecondary)
            if let message {
                Text(message).font(DS.Text.meta).foregroundStyle(DS.Colors.textMuted).multilineTextAlignment(.center)
            }
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, DS.Space.s6)
        .padding(.horizontal, DS.Space.s4)
    }
}

/// Inline notice using the pastel + strong pair.
struct Notice: View {
    enum Tone { case info, warn, bad, good }
    let tone: Tone
    let text: String

    var body: some View {
        Text(text)
            .font(DS.Text.meta)
            .foregroundStyle(fg)
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(12)
            .background(bg, in: RoundedRectangle(cornerRadius: DS.Radius.tooltip, style: .continuous))
    }

    private var fg: Color {
        switch tone {
        case .info: DS.Colors.accentHover
        case .warn: DS.Colors.degradedStrong
        case .bad: DS.Colors.outageStrong
        case .good: Color(hex: 0x2F8F6E)
        }
    }

    private var bg: Color {
        switch tone {
        case .info: DS.Colors.accentTint
        case .warn: DS.Colors.levelWarning
        case .bad: DS.Colors.levelError
        case .good: DS.Colors.levelSuccess
        }
    }
}

/// Key above value, used in fact grids.
struct Fact: View {
    let label: String
    let value: String
    var mono = false

    var body: some View {
        VStack(alignment: .leading, spacing: 2) {
            Text(label).font(DS.Text.caption).foregroundStyle(DS.Colors.textMuted)
            Text(value)
                .font(mono ? DS.Text.code : DS.Text.meta)
                .foregroundStyle(DS.Colors.ink)
                .textSelection(.enabled)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }
}

/// Settings row: label + hint on the left, control on the right, hairline on top.
struct SettingRow<Control: View>: View {
    let label: String
    var hint: String?
    @ViewBuilder var control: Control

    var body: some View {
        HStack(alignment: .center, spacing: DS.Space.s4) {
            VStack(alignment: .leading, spacing: 3) {
                Text(label).font(DS.Text.setting).foregroundStyle(DS.Colors.ink)
                if let hint {
                    Text(hint).font(DS.Text.caption).foregroundStyle(DS.Colors.textMuted)
                }
            }
            Spacer(minLength: 0)
            control
        }
        .padding(.vertical, 12)
        .overlay(alignment: .top) { Rectangle().fill(DS.Colors.borderHairline).frame(height: 1) }
    }
}

/// Text field on the control radius with the accent focus ring.
struct BoopTextField: View {
    let placeholder: String
    @Binding var text: String
    var mono = false
    @FocusState private var focused: Bool

    var body: some View {
        TextField(placeholder, text: $text)
            .font(mono ? DS.Text.code : DS.Text.meta)
            .foregroundStyle(DS.Colors.ink)
            .textInputAutocapitalization(.never)
            .autocorrectionDisabled()
            .focused($focused)
            .padding(.horizontal, 12)
            .frame(height: 40)
            .background(DS.Colors.bg, in: RoundedRectangle(cornerRadius: DS.Radius.control, style: .continuous))
            .overlay(RoundedRectangle(cornerRadius: DS.Radius.control, style: .continuous).stroke(focused ? DS.Colors.accentLine : DS.Colors.borderControl, lineWidth: 1))
            .animation(DS.Motion.gentle, value: focused)
    }
}

/// Small wordmark lockup: periwinkle mark + "Boop".
struct Wordmark: View {
    var body: some View {
        HStack(spacing: 8) {
            ZStack {
                Circle().fill(DS.Colors.accent)
                Circle().fill(DS.Colors.bg).frame(width: 8, height: 8)
            }
            .frame(width: 20, height: 20)
            Text("Boop").font(DS.Text.wordmark).tracking(0.4).foregroundStyle(DS.Colors.ink)
        }
    }
}
