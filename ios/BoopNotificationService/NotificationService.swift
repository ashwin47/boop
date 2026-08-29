import UserNotifications

/// Runs before a push with `mutable-content` is shown. Boop uses it for one
/// thing: turning the `actions` in the payload into real notification buttons.
/// iOS only shows buttons for categories registered ahead of time, and the
/// labels are per-event, so the category is registered here, just in time.
final class NotificationService: UNNotificationServiceExtension {
    private var handler: ((UNNotificationContent) -> Void)?
    private var content: UNMutableNotificationContent?

    override func didReceive(_ request: UNNotificationRequest, withContentHandler contentHandler: @escaping (UNNotificationContent) -> Void) {
        handler = contentHandler
        let mutable = request.content.mutableCopy() as? UNMutableNotificationContent
        content = mutable
        guard let mutable else { return contentHandler(request.content) }

        let info = request.content.userInfo
        let actions = NotificationActions.parse(info)
        guard !actions.isEmpty, let eventID = info["event_id"] as? String else { return contentHandler(mutable) }

        let categoryID = NotificationActions.categoryID(eventID: eventID)
        let buttons = actions.enumerated().map { i, a in
            UNNotificationAction(identifier: NotificationActions.actionID(index: i), title: a.label, options: [.foreground])
        }
        let category = UNNotificationCategory(identifier: categoryID, actions: buttons, intentIdentifiers: [], options: [])
        let center = UNUserNotificationCenter.current()
        center.getNotificationCategories { existing in
            // setNotificationCategories replaces the whole set, so keep what is there
            // (minus stale per-event categories, which pile up otherwise).
            var keep = existing.filter { !$0.identifier.hasPrefix(NotificationActions.categoryPrefix) }
            keep.insert(category)
            center.setNotificationCategories(keep)
            mutable.categoryIdentifier = categoryID
            contentHandler(mutable)
        }
    }

    override func serviceExtensionTimeWillExpire() {
        if let handler, let content { handler(content) }
    }
}
