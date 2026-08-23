import SwiftUI

enum Level: String, Codable, CaseIterable, Sendable, Identifiable {
    case info, success, warning, error, critical

    var id: String { rawValue }

    var label: String {
        switch self {
        case .info: "Info"
        case .success: "Success"
        case .warning: "Warning"
        case .error: "Error"
        case .critical: "Critical"
        }
    }

    var fill: Color {
        switch self {
        case .info: DS.Colors.levelInfo
        case .success: DS.Colors.levelSuccess
        case .warning: DS.Colors.levelWarning
        case .error: DS.Colors.levelError
        case .critical: DS.Colors.levelCritical
        }
    }

    var strong: Color {
        switch self {
        case .info: DS.Colors.levelInfoStrong
        case .success: DS.Colors.levelSuccessStrong
        case .warning: DS.Colors.levelWarningStrong
        case .error: DS.Colors.levelErrorStrong
        case .critical: DS.Colors.levelCriticalStrong
        }
    }

    /// Unknown strings from the server decode as `.info` rather than failing.
    init(lenient raw: String) {
        self = Level(rawValue: raw.lowercased()) ?? .info
    }
}
