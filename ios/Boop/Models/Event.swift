import Foundation

struct Event: Codable, Identifiable, Hashable, Sendable {
    let id: String
    var externalID: String?
    let projectID: String
    let projectName: String
    let projectSlug: String
    let projectIcon: String
    let source: String
    let type: String
    let level: Level
    let title: String
    let body: String
    let fingerprint: String
    let data: JSONValue
    let occurredAt: Date
    let createdAt: Date
    /// True when a silence rule stopped this event from being pushed.
    let silenced: Bool
    /// Buttons that open a URL, shown on the notification and in the detail.
    let actions: [EventAction]
    /// Present in grouped listings: this event is the latest of `group.count` occurrences.
    let group: GroupInfo?

    enum CodingKeys: String, CodingKey {
        case id
        case externalID = "external_id"
        case projectID = "project_id"
        case projectName = "project_name"
        case projectSlug = "project_slug"
        case projectIcon = "project_icon"
        case source, type, level, title, body, fingerprint, data
        case occurredAt = "occurred_at"
        case createdAt = "created_at"
        case silenced, actions, group
    }

    init(from decoder: any Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        id = try c.decode(String.self, forKey: .id)
        externalID = try c.decodeIfPresent(String.self, forKey: .externalID)
        projectID = try c.decode(String.self, forKey: .projectID)
        projectName = try c.decodeIfPresent(String.self, forKey: .projectName) ?? ""
        projectSlug = try c.decodeIfPresent(String.self, forKey: .projectSlug) ?? ""
        projectIcon = try c.decodeIfPresent(String.self, forKey: .projectIcon) ?? ""
        source = try c.decodeIfPresent(String.self, forKey: .source) ?? ""
        type = try c.decodeIfPresent(String.self, forKey: .type) ?? ""
        level = Level(lenient: try c.decodeIfPresent(String.self, forKey: .level) ?? "info")
        title = try c.decode(String.self, forKey: .title)
        body = try c.decodeIfPresent(String.self, forKey: .body) ?? ""
        fingerprint = try c.decodeIfPresent(String.self, forKey: .fingerprint) ?? ""
        data = try c.decodeIfPresent(JSONValue.self, forKey: .data) ?? .object([:])
        occurredAt = try c.decode(Date.self, forKey: .occurredAt)
        createdAt = try c.decode(Date.self, forKey: .createdAt)
        silenced = try c.decodeIfPresent(Bool.self, forKey: .silenced) ?? false
        actions = try c.decodeIfPresent([EventAction].self, forKey: .actions) ?? []
        group = try c.decodeIfPresent(GroupInfo.self, forKey: .group)
    }

    init(id: String, projectID: String, projectName: String, projectIcon: String = "", source: String = "", type: String = "",
         level: Level, title: String, body: String = "", fingerprint: String = "", data: JSONValue = .object([:]),
         occurredAt: Date, createdAt: Date, silenced: Bool = false, actions: [EventAction] = [], group: GroupInfo? = nil) {
        self.id = id
        self.externalID = nil
        self.projectID = projectID
        self.projectName = projectName
        self.projectSlug = projectName.lowercased()
        self.projectIcon = projectIcon
        self.source = source
        self.type = type
        self.level = level
        self.title = title
        self.body = body
        self.fingerprint = fingerprint
        self.data = data
        self.occurredAt = occurredAt
        self.createdAt = createdAt
        self.silenced = silenced
        self.actions = actions
        self.group = group
    }

    /// True when this row stands for more than one occurrence.
    var isRepeated: Bool { (group?.count ?? 1) > 1 }

    func hash(into hasher: inout Hasher) { hasher.combine(id) }
    static func == (a: Event, b: Event) -> Bool { a.id == b.id && a.createdAt == b.createdAt }

    /// Rich sections recognised from `data`; everything else is `rest`.
    var sections: EventSections { EventSections(data: data) }
}

struct GroupInfo: Codable, Hashable, Sendable {
    let count: Int
    let firstSeen: Date
    let lastSeen: Date

    enum CodingKeys: String, CodingKey {
        case count
        case firstSeen = "first_seen"
        case lastSeen = "last_seen"
    }
}

struct EventPage: Codable, Sendable {
    let events: [Event]
    let nextCursor: String?

    enum CodingKeys: String, CodingKey {
        case events
        case nextCursor = "next_cursor"
    }
}

/// Splits `data` into the sections the detail screen renders specially.
struct EventSections: Sendable {
    static let knownKeys: Set<String> = ["exception", "stacktrace", "tags", "context", "breadcrumbs"]

    var exception: [String: JSONValue]?
    var frames: [StackFrame]?
    var tags: [(String, JSONValue)]?
    var context: [(String, JSONValue)]?
    var breadcrumbs: [Breadcrumb]?
    var rest: [(String, JSONValue)]

    init(data: JSONValue) {
        let obj = data.objectValue ?? [:]
        exception = obj["exception"]?.objectValue
        frames = obj["stacktrace"]?.arrayValue?.map(StackFrame.init(json:))
        tags = obj["tags"]?.objectValue.map { $0.sorted { $0.key < $1.key } }
        context = obj["context"]?.objectValue.map { $0.sorted { $0.key < $1.key } }
        breadcrumbs = obj["breadcrumbs"]?.arrayValue?.map(Breadcrumb.init(json:))
        rest = obj.filter { !Self.knownKeys.contains($0.key) }.sorted { $0.key < $1.key }
    }

    var isEmpty: Bool {
        exception == nil && frames == nil && tags == nil && context == nil && breadcrumbs == nil && rest.isEmpty
    }
}

struct StackFrame: Identifiable, Sendable {
    let id = UUID()
    let function: String
    let file: String
    let line: String?
    let inApp: Bool

    init(json: JSONValue) {
        if let o = json.objectValue {
            function = o["function"]?.display ?? o["module"]?.display ?? "—"
            file = o["file"]?.display ?? o["filename"]?.display ?? ""
            line = o["line"]?.display
            inApp = o["in_app"] == .bool(true)
        } else {
            function = json.display
            file = ""
            line = nil
            inApp = false
        }
    }
}

struct Breadcrumb: Identifiable, Sendable {
    let id = UUID()
    let timestamp: String
    let category: String
    let message: String

    init(json: JSONValue) {
        if let o = json.objectValue {
            timestamp = o["timestamp"]?.display ?? o["time"]?.display ?? ""
            category = o["category"]?.display ?? o["type"]?.display ?? ""
            message = o["message"]?.display ?? o["msg"]?.display ?? json.pretty
        } else {
            timestamp = ""
            category = ""
            message = json.display
        }
    }
}
