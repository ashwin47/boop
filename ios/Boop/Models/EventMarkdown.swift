import Foundation

/// Text renderings of an event for copying and sharing. `agentMarkdown` is
/// meant to be pasted straight into an AI assistant: everything the server
/// knows, in a shape a model reads well.
extension Event {
    /// Short plain text: what happened, where, when.
    var plainText: String {
        var lines = [title]
        if !body.isEmpty { lines.append(body) }
        lines.append("")
        var meta = [projectName, level.label]
        if !source.isEmpty { meta.append(source) }
        meta.append(Formatting.full(occurredAt))
        lines.append(meta.joined(separator: " · "))
        for a in actions { lines.append("\(a.label): \(a.url)") }
        return lines.joined(separator: "\n")
    }

    /// Markdown with a section per recognised part of `data`.
    var agentMarkdown: String {
        var out: [String] = ["# \(title)"]
        if !body.isEmpty { out.append(""); out.append(body) }

        var facts: [String] = ["- Project: \(projectName)", "- Level: \(level.rawValue)"]
        if !source.isEmpty { facts.append("- Source: \(source)") }
        if !type.isEmpty { facts.append("- Type: \(type)") }
        facts.append("- Occurred: \(Self.iso.string(from: occurredAt))")
        if let g = group, g.count > 1 {
            facts.append("- Occurrences: \(g.count) (first \(Self.iso.string(from: g.firstSeen)), last \(Self.iso.string(from: g.lastSeen)))")
        }
        if !fingerprint.isEmpty { facts.append("- Fingerprint: \(fingerprint)") }
        if let ext = externalID, !ext.isEmpty { facts.append("- External id: \(ext)") }
        facts.append("- Event id: \(id)")
        out.append(contentsOf: ["", "## Event", ""] + facts)

        let s = sections
        if let exc = s.exception {
            out.append(contentsOf: ["", "## Exception", ""])
            let t = exc["type"]?.display ?? ""
            let m = exc["message"]?.display ?? ""
            switch (t.isEmpty, m.isEmpty) {
            case (false, false): out.append("**\(t)**: \(m)")
            case (false, true): out.append("**\(t)**")
            case (true, false): out.append(m)
            default: break
            }
            for k in exc.keys.filter({ $0 != "type" && $0 != "message" }).sorted() {
                out.append("- \(k): \(exc[k]!.inline)")
            }
        }

        if let tags = s.tags, !tags.isEmpty {
            if let env = tags.first(where: { $0.0 == "environment" || $0.0 == "env" }) {
                out.append(contentsOf: ["", "## Environment", "", env.1.display])
            }
            let others = tags.filter { $0.0 != "environment" && $0.0 != "env" }
            if !others.isEmpty {
                out.append(contentsOf: ["", "## Tags", ""])
                out.append(contentsOf: others.map { "- \($0.0): \($0.1.inline)" })
            }
        }

        if let frames = s.frames, !frames.isEmpty {
            out.append(contentsOf: ["", "## Stack trace", "", "```"])
            for f in frames {
                var loc = f.file
                if let line = f.line, !loc.isEmpty { loc += ":\(line)" }
                var line = f.inApp ? "* " : "  "
                line += f.function
                if !loc.isEmpty { line += "  (\(loc))" }
                out.append(line)
            }
            out.append("```")
            if frames.contains(where: \.inApp) { out.append("`*` marks frames in the application's own code.") }
        }

        if let ctx = s.context, !ctx.isEmpty {
            out.append(contentsOf: ["", "## Context", ""])
            out.append(contentsOf: ctx.map { "- \($0.0): \($0.1.inline)" })
        }

        if let bcs = s.breadcrumbs, !bcs.isEmpty {
            out.append(contentsOf: ["", "## Breadcrumbs", ""])
            for b in bcs {
                var parts: [String] = []
                if !b.timestamp.isEmpty { parts.append(b.timestamp) }
                if !b.category.isEmpty { parts.append("[\(b.category)]") }
                parts.append(b.message)
                out.append("- " + parts.joined(separator: " "))
            }
        }

        if !s.rest.isEmpty {
            out.append(contentsOf: ["", "## Data", "", "```json", JSONValue.object(Dictionary(uniqueKeysWithValues: s.rest)).pretty, "```"])
        }

        if !actions.isEmpty {
            out.append(contentsOf: ["", "## Links", ""])
            out.append(contentsOf: actions.map { "- [\($0.label)](\($0.url))" })
        }
        return out.joined(separator: "\n") + "\n"
    }

    nonisolated(unsafe) private static let iso: ISO8601DateFormatter = {
        let f = ISO8601DateFormatter()
        f.formatOptions = [.withInternetDateTime]
        return f
    }()
}

extension JSONValue {
    /// Scalars as-is; containers as compact JSON on one line.
    var inline: String {
        if isContainer {
            let enc = JSONEncoder()
            enc.outputFormatting = [.sortedKeys, .withoutEscapingSlashes]
            if let d = try? enc.encode(self), let s = String(data: d, encoding: .utf8) { return s }
        }
        return display
    }
}
