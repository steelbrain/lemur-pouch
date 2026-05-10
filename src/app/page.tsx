import Image from "next/image";
import { InstallCommand } from "@/components/InstallCommand";
import { PeerListMockup } from "@/components/PeerListMockup";

const TRUST_SIGNALS = [
  "MIT licensed",
  "macOS, Linux, Windows",
  "No accounts",
  "No telemetry",
  "Session-only",
];

export default function Home() {
  return (
    <div className="flex flex-1 flex-col bg-background text-foreground">
      <main className="flex-1">
        <section className="relative overflow-hidden">
          <div aria-hidden className="hero-glow" />
          <div className="relative mx-auto grid w-full max-w-6xl items-center gap-12 px-6 pt-20 pb-24 lg:grid-cols-[1.05fr_1fr] lg:gap-16 lg:pt-28">
            <div className="flex flex-col items-start">
              <div className="flex items-center gap-3">
                <Image
                  src="/logo.png"
                  alt=""
                  width={56}
                  height={56}
                  priority
                  className="h-12 w-12 rounded-2xl shadow-sm sm:h-14 sm:w-14"
                />
                <span className="text-base font-semibold tracking-tight text-foreground">
                  LemurPouch
                </span>
              </div>

              <h1 className="mt-8 max-w-xl text-4xl font-semibold leading-tight tracking-tight text-foreground sm:text-5xl lg:text-6xl">
                End-to-end encrypted file sharing that{" "}
                <span className="text-pouch-strong">never leaves your network</span>.
              </h1>

              <p className="mt-6 max-w-xl text-lg leading-relaxed text-foreground-muted">
                Run the relay on any LAN machine. Anyone on your network opens
                it in a browser, verifies a six-word fingerprint, and sends.
                Files never touch the cloud — and the relay routes opaque
                ciphertext it can&rsquo;t read.
              </p>

              <div className="mt-10 w-full max-w-xl">
                <InstallCommand command="curl -fsSL https://lemurpouch.com/install.sh | sh" />
                <div className="mt-3 flex flex-wrap items-center gap-x-4 gap-y-1 text-sm text-foreground-muted">
                  <span>Installs the relay. Clients just open a URL.</span>
                  <a
                    href="https://github.com/steelbrain/lemur-pouch"
                    className="font-medium text-accent hover:text-accent-strong hover:underline"
                  >
                    View on GitHub →
                  </a>
                </div>
              </div>

              <ul className="mt-8 flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-foreground-subtle">
                {TRUST_SIGNALS.map((signal, idx) => (
                  <li key={signal} className="flex items-center gap-2">
                    {idx > 0 ? (
                      <span aria-hidden className="text-foreground-subtle/60">
                        ·
                      </span>
                    ) : null}
                    <span>{signal}</span>
                  </li>
                ))}
              </ul>
            </div>

            <div className="relative w-full lg:justify-self-end">
              <div
                aria-hidden
                className="absolute -inset-4 rounded-3xl bg-pouch-soft blur-2xl"
              />
              <div className="relative">
                <PeerListMockup />
              </div>
            </div>
          </div>
        </section>
      </main>

      <footer className="border-t border-border py-6 text-center text-sm text-foreground-muted">
        <p>
          © {new Date().getFullYear()} LemurPouch ·{" "}
          <a
            className="text-accent hover:text-accent-strong hover:underline"
            href="https://github.com/steelbrain/lemur-pouch"
          >
            GitHub
          </a>
        </p>
      </footer>
    </div>
  );
}
