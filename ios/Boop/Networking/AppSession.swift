import Foundation
import Observation
import UIKit

/// Owns the server URL, device credential and connection state.
@MainActor
@Observable
final class AppSession {
    enum Connection: Equatable {
        case unknown, checking, connected, unreachable(String), unauthorized
    }

    private(set) var serverURL: URL?
    private(set) var deviceID: String?
    var deviceName: String
    private(set) var connection: Connection = .unknown
    private(set) var pushRegistered = false
    private(set) var lastAPNSToken: String?
    /// True between a successful pairing and the user tapping Continue on the success screen.
    var onboarding = false

    private let keychain: KeychainStore
    private let defaults: UserDefaults
    private var credential: String?

    private enum Keys {
        static let server = "boop.server"
        static let deviceID = "boop.deviceID"
        static let deviceName = "boop.deviceName"
        static let credential = "credential"
        static let apnsToken = "boop.apnsToken"
    }

    init(keychain: KeychainStore = KeychainStore(), defaults: UserDefaults = .standard) {
        self.keychain = keychain
        self.defaults = defaults
        serverURL = defaults.string(forKey: Keys.server).flatMap(URL.init(string:))
        deviceID = defaults.string(forKey: Keys.deviceID)
        deviceName = defaults.string(forKey: Keys.deviceName) ?? UIDevice.current.name
        credential = keychain.get(Keys.credential)
        lastAPNSToken = defaults.string(forKey: Keys.apnsToken)
        pushRegistered = lastAPNSToken != nil
    }

    var isPaired: Bool { serverURL != nil && credential != nil }

    var client: APIClient? {
        guard let serverURL else { return nil }
        return APIClient(baseURL: serverURL, credential: credential)
    }

    // MARK: Pairing

    func pair(_ payload: PairingPayload) async throws -> Device {
        guard let url = payload.serverURL else { throw PairingPayload.ParseError.badServer }
        let client = APIClient(baseURL: url)
        let result = try await client.exchangePairing(token: payload.token, name: deviceName)
        keychain.set(result.credential, for: Keys.credential)
        credential = result.credential
        serverURL = url
        deviceID = result.device.id
        defaults.set(url.absoluteString, forKey: Keys.server)
        defaults.set(result.device.id, forKey: Keys.deviceID)
        defaults.set(deviceName, forKey: Keys.deviceName)
        connection = .connected
        onboarding = true
        // Re-send a token we already hold (re-pairing after a reinstall keeps pushes working).
        if let token = lastAPNSToken {
            try? await registerAPNSToken(token)
        }
        return result.device
    }

    func disconnect() async {
        if let client, let deviceID {
            try? await client.deleteDevice(id: deviceID)
        }
        forget()
    }

    /// Drops local state without contacting the server (for re-pair, or when the credential was revoked).
    func forget() {
        keychain.delete(Keys.credential)
        credential = nil
        serverURL = nil
        deviceID = nil
        pushRegistered = false
        connection = .unknown
        defaults.removeObject(forKey: Keys.server)
        defaults.removeObject(forKey: Keys.deviceID)
    }

    // MARK: Device

    func registerAPNSToken(_ token: String) async throws {
        lastAPNSToken = token
        defaults.set(token, forKey: Keys.apnsToken)
        guard let client else { return }
        let device = try await client.registerDevice(token: token, name: deviceName, bundleID: Bundle.main.bundleIdentifier ?? "")
        pushRegistered = device.pushRegistered
        deviceID = device.id
    }

    func rename(_ name: String) async {
        let trimmed = name.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return }
        deviceName = trimmed
        defaults.set(trimmed, forKey: Keys.deviceName)
        if let client, let deviceID {
            _ = try? await client.updateDevice(id: deviceID, name: trimmed)
        }
    }

    // MARK: Connection

    @discardableResult
    func checkConnection() async -> Connection {
        guard let client else { connection = .unknown; return connection }
        connection = .checking
        do {
            _ = try await client.events(limit: 1)
            connection = .connected
        } catch let e as APIError where e.isUnauthorized {
            connection = .unauthorized
        } catch {
            connection = .unreachable((error as? APIError)?.message ?? error.localizedDescription)
        }
        return connection
    }
}
