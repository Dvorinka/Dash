import { Nav } from "../components/Nav";
import { Board } from "../components/Board";
import { RendererDemo } from "../components/RendererDemo";
import { MonitorMock, DomainMock, SystemMock, StatusBand } from "../components/Mocks";
import { Install } from "../components/Install";
import { CopyButton } from "../components/CopyButton";
import { Reveal } from "../components/Reveal";

function SectionHead({ label, tag }: { label: string; tag: string }) {
  return (
    <div className="mb-7 flex items-center gap-3">
      <h2 className="font-mono text-[11px] font-semibold uppercase tracking-[0.14em] text-dim">
        {label}
      </h2>
      <span className="h-px flex-1 bg-line" />
      <span className="font-mono text-[11px] text-faint">{tag}</span>
    </div>
  );
}

function Feature({
  flip = false,
  title,
  children,
  mock,
}: {
  flip?: boolean;
  title: string;
  children: React.ReactNode;
  mock: React.ReactNode;
}) {
  return (
    <Reveal>
      <div className="grid items-center gap-8 md:grid-cols-[5fr_7fr]">
        <div className={flip ? "md:order-2" : ""}>
          <h3 className="text-[22px] font-medium tracking-tight">{title}</h3>
          <p className="mt-2.5 max-w-[46ch] text-[14.5px] leading-relaxed text-dim">{children}</p>
        </div>
        <div className={flip ? "md:order-1" : ""}>{mock}</div>
      </div>
    </Reveal>
  );
}

export default function Page() {
  return (
    <>
      <Nav />

      {/* ---------- hero ---------- */}
      <header className="mx-auto max-w-5xl px-5 pt-20 sm:pt-24">
        <Reveal>
          <span className="inline-flex items-center gap-2 rounded-full border border-line bg-surface px-3 py-1 font-mono text-[11.5px] text-dim">
            <span className="h-1.5 w-1.5 rounded-full bg-up animate-pulse-dot" />
            self-hosted · open source · MIT
          </span>
        </Reveal>
        <Reveal delay={0.06}>
          <h1 className="mt-6 max-w-[16ch] text-[42px] font-medium leading-[1.04] tracking-[-0.035em] sm:text-[64px]">
            Your homelab, on one calm screen.
          </h1>
        </Reveal>
        <Reveal delay={0.12}>
          <p className="mt-5 max-w-[58ch] text-[16.5px] leading-relaxed text-dim">
            Service boards, uptime monitors, domain watch, and system metrics - a
            single static Go binary with the UI baked in. No YAML, no external
            database, no mandatory accounts.
          </p>
        </Reveal>
        <Reveal delay={0.18} className="mt-8 flex flex-wrap items-center gap-2.5">
          <a
            href="#install"
            className="flex h-10 items-center rounded-lg bg-text px-4 text-[14px] font-medium text-bg transition-transform duration-100 active:scale-[0.97]"
          >
            Get started
          </a>
          <a
            href="https://github.com/Dvorinka/Dash"
            className="flex h-10 items-center rounded-lg border border-line bg-surface px-4 text-[14px] font-medium text-dim transition-colors hover:border-line-strong hover:text-text"
          >
            GitHub
          </a>
          <span className="flex h-10 items-center gap-1 rounded-lg border border-line bg-surface pl-3.5 pr-1.5 font-mono text-[12.5px]">
            <span className="text-faint">$</span>
            <code>docker run -p 3000:3000 ghcr.io/dvorinka/dash</code>
            <CopyButton
              text="docker run -d -v ./data:/data -p 3000:3000 ghcr.io/dvorinka/dash:latest"
              label="Copy docker command"
            />
          </span>
        </Reveal>

        {/* the product, rendered live */}
        <Reveal delay={0.24} className="relative mt-14">
          <div
            aria-hidden="true"
            className="pointer-events-none absolute -inset-x-8 -top-10 h-40 bg-[radial-gradient(60%_100%_at_50%_0%,rgba(59,130,246,0.10),transparent)]"
          />
          <Board />
        </Reveal>

        <Reveal className="mt-14 grid grid-cols-2 overflow-hidden rounded-xl border border-line sm:grid-cols-4">
          {[
            ["Deploy", "one static binary"],
            ["Database", "SQLite, embedded"],
            ["Config", "UI-managed, no YAML"],
            ["License", "MIT"],
          ].map(([k, v], i) => (
            <div key={k} className={`px-5 py-4 ${i > 0 ? "border-l border-line" : ""} ${i >= 2 ? "max-sm:border-t max-sm:border-line" : ""} ${i === 2 ? "max-sm:border-l-0" : ""}`}>
              <div className="font-mono text-[10px] uppercase tracking-[0.12em] text-faint">{k}</div>
              <div className="mt-1.5 text-[14.5px] font-medium">{v}</div>
            </div>
          ))}
        </Reveal>
      </header>

      <main className="mx-auto max-w-5xl px-5">
        {/* ---------- renderers ---------- */}
        <section id="renderers" className="scroll-mt-24 pt-24">
          <SectionHead label="Boards" tag="the differentiator" />
          <div className="grid items-center gap-8 md:grid-cols-[4fr_8fr]">
            <Reveal>
              <h3 className="text-[22px] font-medium tracking-tight">
                One board state. Four skins.
              </h3>
              <p className="mt-2.5 max-w-[42ch] text-[14.5px] leading-relaxed text-dim">
                Bento, Cards, Index, and Console renderers share the same drag-drop,
                popovers, and state. Switching the look is a setting, not a migration.
                Try it - this demo is live.
              </p>
            </Reveal>
            <Reveal delay={0.1}>
              <RendererDemo />
            </Reveal>
          </div>
        </section>

        {/* ---------- monitors ---------- */}
        <section id="monitors" className="scroll-mt-24 pt-24">
          <SectionHead label="Monitors" tag="checks on a schedule" />
          <Feature title="Uptime that watches itself" mock={<MonitorMock />}>
            HTTP, TCP, ping, DNS, keyword, JSON-path, and push checks on an
            in-process scheduler. Heartbeat history, latency graphs, and alert
            rules per monitor - down means you hear about it, not discover it.
          </Feature>
        </section>

        {/* ---------- domains ---------- */}
        <section className="pt-24">
          <SectionHead label="Domains" tag="rdap + whois + tls" />
          <Feature
            flip
            title="Every domain, accounted for"
            mock={<DomainMock />}
          >
            Expiry countdowns, TLS certificate chains, DNS records attributed to
            their provider, and CT-log subdomain discovery. Field-level history on
            every change.
          </Feature>
        </section>

        {/* ---------- systems ---------- */}
        <section className="pt-24">
          <SectionHead label="Systems" tag="push-based agent" />
          <Feature title="Hosts that report in" mock={<SystemMock />}>
            A tiny <code className="rounded border border-line bg-surface px-1.5 py-0.5 font-mono text-[12.5px]">dash-agent</code>{" "}
            pushes CPU, memory, disk, network, temperatures, and per-container
            stats - SMART, ZFS, and GPU when the tooling exists. One binary per
            host, nothing else.
          </Feature>
        </section>

        {/* ---------- status + alerts ---------- */}
        <section className="pt-24">
          <SectionHead label="Status & alerts" tag="public pages, real transports" />
          <Feature
            flip
            title="Tell users before they ask"
            mock={<StatusBand />}
          >
            Automatic incidents on monitor down, maintenance windows, public
            status pages, and SVG badges. Alerts dispatch through webhook, Slack,
            Discord, Telegram, Gotify, ntfy, or SMTP.
          </Feature>
        </section>

        {/* ---------- stack ---------- */}
        <section id="stack" className="scroll-mt-24 pt-24">
          <SectionHead label="Stack" tag="boring, on purpose" />
          <Reveal>
            <div className="flex flex-wrap gap-2">
              {[
                "go 1.24",
                "gin",
                "sqlite (modernc)",
                "embed.FS",
                "openapi contract",
                "react 18",
                "docker · ~34 MB",
                "goreleaser binaries",
                "prometheus /api/metrics",
                "pwa",
              ].map((t) => (
                <span
                  key={t}
                  className="rounded-md border border-line bg-surface px-3 py-1.5 font-mono text-[12px] text-dim"
                >
                  {t}
                </span>
              ))}
            </div>
          </Reveal>
          <Reveal delay={0.08}>
            <p className="mt-5 max-w-[62ch] text-[14.5px] leading-relaxed text-dim">
              It needs a persistent scheduler, WHOIS on port 43, ICMP, and SMTP
              egress - none of which exist on serverless. So the deploy story is{" "}
              <code className="rounded border border-line bg-surface px-1.5 py-0.5 font-mono text-[12.5px]">docker run</code>
              , and this page is the only part of Dash hosted here.
            </p>
          </Reveal>
        </section>

        {/* ---------- install ---------- */}
        <section id="install" className="scroll-mt-24 pt-24">
          <SectionHead label="Install" tag="~1 min" />
          <Install />
          <Reveal delay={0.1}>
            <p className="mt-5 max-w-[62ch] text-[14.5px] leading-relaxed text-dim">
              Everything is configured from the UI at{" "}
              <code className="rounded border border-line bg-surface px-1.5 py-0.5 font-mono text-[12.5px]">localhost:3000</code>
              . All state lives in one directory. Importers for Homepage, Homarr,
              and Dashy configs are built in.
            </p>
          </Reveal>
        </section>

        {/* ---------- bottom cta ---------- */}
        <Reveal className="mt-24">
          <div className="flex flex-wrap items-center justify-between gap-6 rounded-2xl border border-line bg-gradient-to-b from-surface to-bg p-10">
            <div>
              <h2 className="text-[26px] font-medium tracking-tight">Run it in one command.</h2>
              <p className="mt-2 text-[14.5px] text-dim">MIT licensed. Self-hosted. Yours.</p>
            </div>
            <a
              href="https://github.com/Dvorinka/Dash"
              className="flex h-11 items-center rounded-lg bg-text px-5 text-[14px] font-medium text-bg transition-transform duration-100 active:scale-[0.97]"
            >
              Star on GitHub
            </a>
          </div>
        </Reveal>
      </main>

      <footer className="mt-24 border-t border-line">
        <div className="mx-auto flex max-w-5xl flex-wrap items-center gap-5 px-5 py-8 text-[13px] text-faint">
          <span className="flex items-center gap-2">
            <img src="/dash.svg" alt="" width={18} height={18} className="rounded-[4px]" />
            Dash
          </span>
          <span>© 2026 · MIT License</span>
          <div className="ml-auto flex gap-5">
            <a className="text-dim transition-colors hover:text-text" href="https://github.com/Dvorinka/Dash">GitHub</a>
            <a className="text-dim transition-colors hover:text-text" href="https://github.com/Dvorinka/Dash/tree/main/docs">Docs</a>
            <a className="text-dim transition-colors hover:text-text" href="https://github.com/Dvorinka/Dash/releases">Releases</a>
          </div>
        </div>
      </footer>
    </>
  );
}
