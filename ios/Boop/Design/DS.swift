import SwiftUI

/// Design tokens, mirroring the CSS variables in `server/web/src/app.css`.
enum DS {
    // MARK: Colors
    enum Colors {
        static let bg = Color(hex: 0xFDFDFE)
        static let bgHover = Color(hex: 0xFAFAFC)
        static let ink = Color(hex: 0x1A1B25)
        static let surfaceDark = Color(hex: 0x17181F)

        static let text = ink
        static let textSecondary = Color(hex: 0x6B6D7C)
        static let textMuted = Color(hex: 0x9C9EAB)
        static let textInactive = Color(hex: 0xB4B6C2)
        static let textFaint = Color(hex: 0xC2C4CE)
        static let textOnDark = Color(hex: 0xFDFDFE)
        static let textOnDarkMuted = Color(hex: 0xFDFDFE).opacity(0.55)

        static let borderHairline = Color(hex: 0xF0F1F5)
        static let borderControl = Color(hex: 0xE4E6EE)
        static let dividerOnDark = Color(hex: 0xFDFDFE).opacity(0.12)

        static let accent = Color(hex: 0x7C83E8)
        static let accentHover = Color(hex: 0x5A62D4)
        static let accentLine = Color(hex: 0x8B92EC)
        static let accentTint = Color(hex: 0xEEF0FB)

        static let operational = Color(hex: 0xDBDFF8)
        static let operationalStrong = Color(hex: 0xA9ADD9)
        static let degraded = Color(hex: 0xF5B266)
        static let degradedStrong = Color(hex: 0xE0912F)
        static let outage = Color(hex: 0x9B7BEA)
        static let outageStrong = Color(hex: 0x8A63E0)

        static let altMint = Color(hex: 0x5FBF9F)
        static let altBlush = Color(hex: 0xE88CB0)
        static let altAmber = Color(hex: 0xE8B34C)

        // Boop event levels, mapped onto the pastel + strong pattern (same as the web UI).
        static let levelInfo = operational
        static let levelInfoStrong = operationalStrong
        static let levelSuccess = Color(hex: 0xDDF3EA)
        static let levelSuccessStrong = altMint
        static let levelWarning = Color(hex: 0xFCE8D0)
        static let levelWarningStrong = degradedStrong
        static let levelError = Color(hex: 0xE8DFFA)
        static let levelErrorStrong = outageStrong
        static let levelCritical = outage
        static let levelCriticalStrong = Color(hex: 0x6A45C4)

        /// Healthy/ok status uses mint, per the maintainer's preference.
        static let ok = altMint
    }

    // MARK: Radii
    enum Radius {
        static let bar: CGFloat = 2
        static let cell: CGFloat = 3
        static let row: CGFloat = 6
        static let control: CGFloat = 8
        static let tooltip: CGFloat = 10
        static let card: CGFloat = 14
        static let pill: CGFloat = 999
    }

    // MARK: Spacing
    enum Space {
        static let s1: CGFloat = 3
        static let s2: CGFloat = 8
        static let s3: CGFloat = 10
        static let s4: CGFloat = 16
        static let s5: CGFloat = 24
        static let s6: CGFloat = 40
        static let s7: CGFloat = 56
        static let pagePad: CGFloat = 20
    }

    // MARK: Type scale (Figtree, hierarchy by weight + gray)
    enum Text {
        static var wordmark: Font { .figtree(20, .bold) }
        static var metric: Font { .figtree(24, .bold) }
        static var rowTitle: Font { .figtree(15, .semibold) }
        static var statusLine: Font { .figtree(15, .medium) }
        static var setting: Font { .figtree(14, .semibold) }
        static var ui: Font { .figtree(13, .semibold) }
        static var meta: Font { .figtree(13, .medium) }
        static var small: Font { .figtree(12, .semibold) }
        static var caption: Font { .figtree(11, .medium) }
        static var code: Font { .system(size: 12, design: .monospaced) }
        static var title: Font { .figtree(28, .bold) }
    }

    /// Restrained motion: state changes should feel near-instant.
    enum Motion {
        static let snappy = Animation.spring(response: 0.32, dampingFraction: 0.86)
        static let gentle = Animation.easeOut(duration: 0.18)
        static let settle = Animation.spring(response: 0.45, dampingFraction: 0.8)
    }
}

extension Color {
    init(hex: UInt32) {
        self.init(
            .sRGB,
            red: Double((hex >> 16) & 0xFF) / 255,
            green: Double((hex >> 8) & 0xFF) / 255,
            blue: Double(hex & 0xFF) / 255,
            opacity: 1
        )
    }
}

extension Font {
    /// Figtree when the bundled font loaded, system font otherwise.
    static func figtree(_ size: CGFloat, _ weight: Font.Weight = .regular) -> Font {
        let name: String
        switch weight {
        case .bold, .heavy, .black: name = "Figtree-Bold"
        case .semibold: name = "Figtree-SemiBold"
        case .medium: name = "Figtree-Medium"
        default: name = "Figtree-Regular"
        }
        if UIFont(name: name, size: size) != nil {
            return .custom(name, size: size)
        }
        return .system(size: size, weight: weight)
    }
}
