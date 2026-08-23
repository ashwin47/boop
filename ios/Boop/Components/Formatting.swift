import Foundation

enum Formatting {
    /// "just now", "2m", "3h", "2d", or "Aug 12" — terse, abbreviated.
    static func relative(_ date: Date, now: Date = .now) -> String {
        let sec = now.timeIntervalSince(date)
        if sec < 45 { return "now" }
        let min = Int((sec / 60).rounded())
        if min < 60 { return "\(min)m" }
        let hr = Int((Double(min) / 60).rounded())
        if hr < 24 { return "\(hr)h" }
        let day = Int((Double(hr) / 24).rounded())
        if day < 7 { return "\(day)d" }
        return shortDate(date)
    }

    static func shortDate(_ date: Date) -> String {
        date.formatted(.dateTime.month(.abbreviated).day())
    }

    /// "Wed, Aug 12 · 14:51:44"
    static func full(_ date: Date) -> String {
        let d = date.formatted(.dateTime.weekday(.abbreviated).month(.abbreviated).day())
        let t = date.formatted(.dateTime.hour(.twoDigits(amPM: .omitted)).minute(.twoDigits).second(.twoDigits))
        return "\(d) · \(t)"
    }

    /// "Today", "Yesterday", or a short date.
    static func dayGroup(_ date: Date, now: Date = .now, calendar: Calendar = .current) -> String {
        if calendar.isDate(date, inSameDayAs: now) { return "Today" }
        if let y = calendar.date(byAdding: .day, value: -1, to: now), calendar.isDate(date, inSameDayAs: y) { return "Yesterday" }
        return date.formatted(.dateTime.weekday(.abbreviated).month(.abbreviated).day())
    }
}
