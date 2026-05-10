import Image from "next/image";

const features = [
  {
    title: "End-to-end encrypted",
    body: "Files are sealed with XChaCha20-Poly1305 on your device. The relay only sees opaque bytes — never your content, never your keys.",
  },
  {
    title: "Works on locked-down networks",
    body: "All traffic is plain outbound TCP over WebSockets. No WebRTC, no UDP, no STUN — runs through corporate firewalls and captive portals that break everything else.",
  },
  {
    title: "Nothing to install",
    body: "Open the page in two browsers, confirm the BIP-39 fingerprint, and send. No accounts, no extensions, no persistence.",
  },
];

export default function Home() {
  return (
    <div className="flex flex-1 flex-col bg-background text-foreground">
      <main className="mx-auto flex w-full max-w-3xl flex-1 flex-col items-center px-6 pt-16 pb-24 text-center sm:pt-24">
        <span className="inline-flex items-center gap-2 rounded-full border border-pouch/40 bg-pouch-soft px-3 py-1 text-xs font-medium tracking-wide text-pouch uppercase">
          <span className="h-1.5 w-1.5 rounded-full bg-pouch" />
          Coming soon
        </span>

        <Image
          src="/logo.png"
          alt="LemurPouch logo"
          width={140}
          height={140}
          priority
          className="mt-8 h-32 w-32 rounded-3xl shadow-lg sm:h-36 sm:w-36"
        />

        <h1 className="mt-8 text-4xl font-semibold tracking-tight sm:text-5xl">
          LemurPouch
        </h1>

        <p className="mt-4 max-w-xl text-lg text-foreground-muted sm:text-xl">
          LAN file sharing, simplified. End-to-end encrypted transfers that
          punch through the most restrictive networks — no setup, no accounts,
          no servers that can read your data.
        </p>

        <div className="mt-12 grid w-full grid-cols-1 gap-4 text-left sm:grid-cols-3">
          {features.map((f) => (
            <div
              key={f.title}
              className="rounded-2xl border border-border bg-background-elevated p-5"
            >
              <h2 className="text-sm font-semibold text-foreground">
                {f.title}
              </h2>
              <p className="mt-2 text-sm leading-relaxed text-foreground-muted">
                {f.body}
              </p>
            </div>
          ))}
        </div>

        <div className="mt-12 inline-flex items-center gap-3 rounded-full border border-border bg-background-elevated px-5 py-2.5 text-sm text-foreground-muted">
          <span
            className="h-2 w-2 animate-pulse rounded-full bg-accent"
            aria-hidden
          />
          We&rsquo;re putting the finishing touches on it. Check back soon.
        </div>
      </main>

      <footer className="border-t border-border py-6 text-center text-sm text-foreground-muted">
        <p>
          © {new Date().getFullYear()} LemurPouch ·{" "}
          <a
            className="text-accent hover:underline"
            href="https://github.com/steelbrain/lemur-pouch"
          >
            GitHub
          </a>
        </p>
      </footer>
    </div>
  );
}
