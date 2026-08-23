import SwiftUI
import UIKit

/// Project icon: "<shape>:<color>" rendered as an abstract shape from the palette; other text shown as-is.
/// Mirrors server/internal/projects/icons.go and the web ProjectIcon component.
struct ProjectIcon: View {
    let icon: String
    var size: CGFloat = 16

    static let colors: [String: Color] = [
        "periwinkle": DS.Colors.accent,
        "mint": DS.Colors.altMint,
        "blush": DS.Colors.altBlush,
        "amber": DS.Colors.altAmber,
        "violet": DS.Colors.outage,
        "slate": DS.Colors.textMuted,
    ]

    static func parse(_ icon: String) -> (shape: IconShape, color: Color)? {
        guard let i = icon.firstIndex(of: ":") else { return nil }
        guard let shape = IconShape(rawValue: String(icon[..<i])), let color = colors[String(icon[icon.index(after: i)...])] else { return nil }
        return (shape, color)
    }

    var body: some View {
        if let p = Self.parse(icon) {
            p.shape.fill(p.color).frame(width: size, height: size)
        } else if !icon.isEmpty {
            Text(icon).font(.system(size: size * 0.9))
        }
    }
}

enum IconShape: String, CaseIterable {
    case circle, ring, square, diamond, triangle, hexagon, pill, blob

    @ViewBuilder
    func fill(_ color: Color) -> some View {
        switch self {
        case .circle: Circle().fill(color).padding(1)
        case .ring: Circle().strokeBorder(color, lineWidth: 3.2).padding(1.5)
        case .square: RoundedRectangle(cornerRadius: 3, style: .continuous).fill(color).padding(1.5)
        case .diamond: RoundedRectangle(cornerRadius: 2, style: .continuous).fill(color).rotationEffect(.degrees(45)).padding(3.5)
        case .triangle: Triangle().fill(color).padding(1)
        case .hexagon: Polygon(sides: 6).fill(color).padding(0.5)
        case .pill: Capsule().fill(color).padding(.vertical, 4.5).padding(.horizontal, 0.5)
        case .blob: Blob().fill(color).padding(1)
        }
    }

    /// Small rendered image for places SwiftUI cannot colour shapes (menus).
    @MainActor
    func image(_ color: Color, size: CGFloat = 20) -> UIImage {
        let renderer = ImageRenderer(content: fill(color).frame(width: size, height: size))
        renderer.scale = 3
        return renderer.uiImage?.withRenderingMode(.alwaysOriginal) ?? UIImage()
    }
}

struct Triangle: Shape {
    func path(in r: CGRect) -> Path {
        var p = Path()
        p.move(to: CGPoint(x: r.midX, y: r.minY + r.height * 0.06))
        p.addLine(to: CGPoint(x: r.maxX - r.width * 0.04, y: r.maxY - r.height * 0.1))
        p.addLine(to: CGPoint(x: r.minX + r.width * 0.04, y: r.maxY - r.height * 0.1))
        p.closeSubpath()
        return p.strokedPath(StrokeStyle(lineWidth: r.width * 0.12, lineJoin: .round)).union(p)
    }
}

struct Polygon: Shape {
    let sides: Int
    func path(in r: CGRect) -> Path {
        var p = Path()
        let c = CGPoint(x: r.midX, y: r.midY)
        let radius = min(r.width, r.height) / 2
        for i in 0..<sides {
            let a = (Double(i) / Double(sides)) * 2 * .pi - .pi / 2
            let pt = CGPoint(x: c.x + radius * cos(a), y: c.y + radius * sin(a))
            if i == 0 { p.move(to: pt) } else { p.addLine(to: pt) }
        }
        p.closeSubpath()
        return p
    }
}

struct Blob: Shape {
    func path(in r: CGRect) -> Path {
        let w = r.width, h = r.height
        func pt(_ x: CGFloat, _ y: CGFloat) -> CGPoint { CGPoint(x: r.minX + x / 20 * w, y: r.minY + y / 20 * h) }
        var p = Path()
        p.move(to: pt(10.5, 2))
        p.addCurve(to: pt(18, 9.5), control1: pt(15, 2), control2: pt(18.5, 5))
        p.addCurve(to: pt(10, 18), control1: pt(17.6, 13.5), control2: pt(15, 18))
        p.addCurve(to: pt(2, 10.5), control1: pt(5.5, 18), control2: pt(2, 15))
        p.addCurve(to: pt(10.5, 2), control1: pt(2, 6), control2: pt(6, 2))
        p.closeSubpath()
        return p
    }
}
