import SwiftUI
import VisionKit

struct PairingView: View {
    @Environment(AppSession.self) private var session
    @Environment(PushManager.self) private var push

    enum Mode { case welcome, scan, manual, pairing, done }
    @State private var mode: Mode = .welcome
    @State private var server = ""
    @State private var token = ""
    @State private var error: String?
    @State private var device: Device?

    var body: some View {
        VStack(spacing: 0) {
            header
            Group {
                switch mode {
                case .welcome: welcome
                case .scan: scanner
                case .manual: manual
                case .pairing: pairing
                case .done: done
                }
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity)
            .transition(.asymmetric(insertion: .move(edge: .trailing).combined(with: .opacity), removal: .opacity))
            .id(mode)
        }
        .animation(DS.Motion.settle, value: mode)
        .background(DS.Colors.bg)
    }

    private var header: some View {
        HStack {
            Wordmark()
            Spacer()
            if mode == .scan || mode == .manual {
                Button("Cancel") { withAnimation { mode = .welcome; error = nil } }
                    .buttonStyle(.boopGhost)
            }
        }
        .padding(.horizontal, DS.Space.pagePad)
        .padding(.vertical, 14)
    }

    // MARK: Screens

    private var welcome: some View {
        VStack(alignment: .leading, spacing: DS.Space.s5) {
            Spacer()
            Text("Welcome to Boop")
                .font(DS.Text.title)
                .foregroundStyle(DS.Colors.ink)
            Text("Boop connects directly to your self-hosted server. Nothing passes through anyone else.")
                .font(DS.Text.statusLine)
                .foregroundStyle(DS.Colors.textSecondary)
                .fixedSize(horizontal: false, vertical: true)
            Spacer()
            if let error {
                Notice(tone: .bad, text: error)
            }
            VStack(spacing: 10) {
                Button {
                    withAnimation { mode = DataScannerViewController.isSupported ? .scan : .manual }
                } label: {
                    Label("Pair server", systemImage: "qrcode.viewfinder")
                }
                .buttonStyle(.boopPrimary)
                Button("Enter details manually") { withAnimation { mode = .manual } }
                    .buttonStyle(.boopGhost)
            }
            Text("Open your Boop web UI, go to Devices, and tap Pair iPhone to show a code.")
                .font(DS.Text.caption)
                .foregroundStyle(DS.Colors.textMuted)
                .frame(maxWidth: .infinity)
                .multilineTextAlignment(.center)
        }
        .padding(DS.Space.pagePad)
    }

    private var scanner: some View {
        VStack(spacing: DS.Space.s4) {
            QRScannerView { text in
                handle(scanned: text)
            }
            .clipShape(RoundedRectangle(cornerRadius: DS.Radius.card, style: .continuous))
            .overlay(RoundedRectangle(cornerRadius: DS.Radius.card, style: .continuous).stroke(DS.Colors.borderHairline))
            .overlay { ScanReticle() }
            Text("Point the camera at the pairing code")
                .font(DS.Text.meta)
                .foregroundStyle(DS.Colors.textSecondary)
            if let error {
                Notice(tone: .bad, text: error)
            }
            Button("Enter details manually") { withAnimation { mode = .manual } }
                .buttonStyle(.boopGhost)
        }
        .padding(DS.Space.pagePad)
    }

    private var manual: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: DS.Space.s4) {
                Text("Pair manually").font(DS.Text.metric).foregroundStyle(DS.Colors.ink)
                Text("Use Show payload under the QR code in the web UI to see these values.")
                    .font(DS.Text.meta).foregroundStyle(DS.Colors.textSecondary)
                VStack(alignment: .leading, spacing: 6) {
                    Text("Server").font(DS.Text.caption).foregroundStyle(DS.Colors.textMuted)
                    BoopTextField(placeholder: "https://boop.example.com", text: $server)
                        .keyboardType(.URL)
                }
                VStack(alignment: .leading, spacing: 6) {
                    Text("Pairing token").font(DS.Text.caption).foregroundStyle(DS.Colors.textMuted)
                    BoopTextField(placeholder: "pair_…", text: $token, mono: true)
                }
                if let error {
                    Notice(tone: .bad, text: error)
                }
                Button("Pair") {
                    do {
                        let p = try PairingPayload.manual(server: server, token: token)
                        start(p)
                    } catch {
                        self.error = error.localizedDescription
                    }
                }
                .buttonStyle(.boopPrimary)
                .disabled(server.isEmpty || token.isEmpty)
            }
            .padding(DS.Space.pagePad)
        }
        .scrollDismissesKeyboard(.interactively)
    }

    private var pairing: some View {
        VStack(spacing: DS.Space.s4) {
            ProgressView().tint(DS.Colors.accent).controlSize(.large)
            Text("Pairing with your server").font(DS.Text.statusLine).foregroundStyle(DS.Colors.textSecondary)
        }
    }

    private var done: some View {
        VStack(spacing: DS.Space.s4) {
            Spacer()
            Image(systemName: "checkmark.circle.fill")
                .font(.system(size: 56, weight: .medium))
                .foregroundStyle(DS.Colors.ok)
                .symbolEffect(.bounce, options: .nonRepeating)
            Text("Paired").font(DS.Text.metric).foregroundStyle(DS.Colors.ink)
            if let device {
                Text("\(device.name) is connected to \(session.serverURL?.host() ?? "your server").")
                    .font(DS.Text.meta).foregroundStyle(DS.Colors.textSecondary).multilineTextAlignment(.center)
            }
            Spacer()
            Text("Next, allow notifications so Boop can reach you.")
                .font(DS.Text.caption).foregroundStyle(DS.Colors.textMuted)
            Button("Continue") {
                Task {
                    await push.requestAndRegister()
                    withAnimation(DS.Motion.settle) { session.onboarding = false }
                    await session.checkConnection()
                }
            }
            .buttonStyle(.boopPrimary)
        }
        .padding(DS.Space.pagePad)
    }

    // MARK: Actions

    private func handle(scanned text: String) {
        guard mode == .scan else { return }
        do {
            start(try PairingPayload.parse(text))
        } catch {
            self.error = error.localizedDescription
        }
    }

    private func start(_ payload: PairingPayload) {
        error = nil
        withAnimation { mode = .pairing }
        Task {
            do {
                let d = try await session.pair(payload)
                device = d
                withAnimation { mode = .done }
                // Hold on the success screen; the root view swaps to the inbox when the user continues.
            } catch {
                self.error = (error as? APIError)?.message ?? error.localizedDescription
                withAnimation { mode = payload.token == token ? .manual : .welcome }
            }
        }
    }
}

/// Corner brackets over the camera preview.
private struct ScanReticle: View {
    @State private var breathe = false

    var body: some View {
        GeometryReader { geo in
            let side = min(geo.size.width, geo.size.height) * 0.62
            RoundedRectangle(cornerRadius: DS.Radius.card, style: .continuous)
                .strokeBorder(DS.Colors.bg.opacity(0.9), lineWidth: 2)
                .frame(width: side, height: side)
                .scaleEffect(breathe ? 1.02 : 0.98)
                .position(x: geo.size.width / 2, y: geo.size.height / 2)
                .animation(.easeInOut(duration: 1.6).repeatForever(autoreverses: true), value: breathe)
                .onAppear { breathe = true }
        }
        .allowsHitTesting(false)
    }
}
