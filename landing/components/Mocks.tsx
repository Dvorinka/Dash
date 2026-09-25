"use client";

import { useEffect, useState } from "react";

/* Small live mocks for feature rows - data-shaped divs, no images. */

function useReduced() {
  const [r, setR] = useState(false);
  useEffect(() => {
    const mq = window.matchMedia("(prefers-reduced-motion: reduce)");
    setR(mq.matches);
    const fn = (e: MediaQueryListEvent) => setR(e.matches);
    mq.addEventListener("change", fn);
    return () => mq.removeEventListener("change", fn);
  }, []);
  return r;
}

function usePing(base: number, spread: number, ms: number) {
  const [v, setV] = useState(base);
  const reduced = useReduced();
  useEffect(() => {
    if (reduced) return;
    const t = setInterval(() => setV(Math.round(base + (Math.random() - 0.4) * spread)), ms);
    return () => clearInterval(t);
  }, [base, spread, ms, reduced]);
  return v;
}

function HeartbeatStrip({ seed, ms }: { seed: number; ms: number }) {
  const [bars, setBars] = useState<number[]>(() =>
    Array.from({ length: 32 }, (_, k) => ((k * seed) % 23 === 0 ? 0 : 1)),
  );
  const reduced = useReduced();
  useEffect(() => {
    if (reduced) return;
    const t = setInterval(
      () => setBars((b) => [...b.slice(1), Math.random() > 0.045 ? 1 : 0]),
      ms,
    );
    return () => clearInterval(t);
  }, [seed, ms, reduced]);
  return (
    <div className="flex h-6 flex-1 items-end gap-[2px]">
      {bars.map((v, k) => (
        <span
          key={k}
          className={`flex-1 rounded-[1.5px] transition-colors duration-300 ${v ? "bg-up" : "bg-down"}`}
          style={{ height: v ? `${55 + ((k * 29) % 45)}%` : "100%", opacity: v ? 0.8 : 1 }}
        />
      ))}
    </div>
  );
}

const MONITORS = [
  { name: "gitea.tdvorak.dev", kind: "https · 30s", up: "99.98", base: 38, seed: 3 },
  { name: "api.internal", kind: "tcp · 60s", up: "100.00", base: 12, seed: 7 },
  { name: "pihole.local", kind: "keyword · 60s", up: "99.62", base: 24, seed: 11 },
];

function MonitorRow({ m, i }: { m: (typeof MONITORS)[number]; i: number }) {
  const ping = usePing(m.base, 18, 1700 + i * 300);
  return (
    <div className="flex items-center gap-4 px-4 py-3">
      <div className="w-36 shrink-0">
        <div className="truncate font-mono text-[12px]">{m.name}</div>
        <div className="font-mono text-[10px] text-faint">{m.kind}</div>
      </div>
      <HeartbeatStrip seed={m.seed} ms={1300 + i * 250} />
      <div className="w-14 text-right font-mono text-[11px] tabular-nums text-dim">{ping} ms</div>
      <div className="w-16 rounded border border-line px-1.5 py-0.5 text-center font-mono text-[10px] text-up">
        {m.up}%
      </div>
    </div>
  );
}

export function MonitorMock() {
  return (
    <div className="divide-y divide-line rounded-xl border border-line bg-surface/50">
      {MONITORS.map((m, i) => (
        <MonitorRow key={m.name} m={m} i={i} />
      ))}
    </div>
  );
}

const DOMAINS = [
  { name: "tdvorak.dev", days: 260, pct: 71, tags: ["porkbun", "cloudflare", "tls 81d"] },
  { name: "kan.tdvorak.dev", days: 260, pct: 71, tags: ["porkbun", "cloudflare", "tls 81d"] },
  { name: "old-legacy.net", days: 12, pct: 8, tags: ["godaddy", "ns1", "tls 9d"], warn: true },
];

export function DomainMock() {
  return (
    <div className="divide-y divide-line rounded-xl border border-line bg-surface/50">
      {DOMAINS.map((d) => (
        <div key={d.name} className="flex items-center gap-4 px-4 py-3.5">
          <div className="w-36 shrink-0">
            <div className="truncate font-mono text-[12px]">{d.name}</div>
            <div className="mt-1.5 h-1 w-full overflow-hidden rounded bg-iconbg">
              <div
                className={`h-full rounded ${d.warn ? "bg-down" : "bg-accent"}`}
                style={{ width: `${d.pct}%` }}
              />
            </div>
          </div>
          <span
            className={`font-mono text-[12px] tabular-nums ${d.warn ? "text-down" : "text-dim"}`}
          >
            {d.days}d
          </span>
          <div className="ml-auto hidden gap-1.5 sm:flex">
            {d.tags.map((t) => (
              <span
                key={t}
                className="rounded border border-line px-1.5 py-0.5 font-mono text-[10px] text-faint"
              >
                {t}
              </span>
            ))}
          </div>
        </div>
      ))}
    </div>
  );
}

export function SystemMock() {
  const cpu = usePing(37, 30, 1900);
  const mem = usePing(61, 6, 2600);
  const R = 44;
  const C = 2 * Math.PI * R;
  return (
    <div className="flex items-center gap-6 rounded-xl border border-line bg-surface/50 p-5">
      <div className="relative h-28 w-28 shrink-0">
        <svg viewBox="0 0 100 100" className="h-full w-full -rotate-90">
          <circle cx="50" cy="50" r={R} fill="none" stroke="var(--color-iconbg)" strokeWidth="7" />
          <circle
            cx="50"
            cy="50"
            r={R}
            fill="none"
            stroke="var(--color-accent)"
            strokeWidth="7"
            strokeLinecap="round"
            strokeDasharray={C}
            strokeDashoffset={C - (C * Math.min(cpu, 100)) / 100}
            style={{ transition: "stroke-dashoffset 0.9s cubic-bezier(0.2,0,0,1)" }}
          />
        </svg>
        <div className="absolute inset-0 flex flex-col items-center justify-center">
          <span className="text-xl font-medium tabular-nums">{cpu}%</span>
          <span className="font-mono text-[9px] uppercase tracking-[0.12em] text-faint">cpu</span>
        </div>
      </div>
      <div className="min-w-0 flex-1 space-y-3.5">
        <div>
          <div className="mb-1 flex justify-between font-mono text-[10px] text-faint">
            <span>memory</span>
            <span className="tabular-nums">{mem}%</span>
          </div>
          <div className="h-1.5 overflow-hidden rounded bg-iconbg">
            <div className="h-full rounded bg-up/80 transition-all duration-700" style={{ width: `${mem}%` }} />
          </div>
        </div>
        <div>
          <div className="mb-1 flex justify-between font-mono text-[10px] text-faint">
            <span>disk /data</span>
            <span>44%</span>
          </div>
          <div className="h-1.5 overflow-hidden rounded bg-iconbg">
            <div className="h-full rounded bg-accent/80" style={{ width: "44%" }} />
          </div>
        </div>
        <div className="flex items-center justify-between font-mono text-[10px] text-faint">
          <span>8 containers</span>
          <span className="tabular-nums">load 0.42 · up 14d</span>
        </div>
      </div>
    </div>
  );
}

const ALERTS: { at: string; text: string; kind: "ok" | "down" | "warn" }[] = [
  { at: "12:04:11", text: "pihole.local recovered", kind: "ok" },
  { at: "12:03:48", text: "pihole.local is down", kind: "down" },
  { at: "11:58:02", text: "old-legacy.net expires in 12d", kind: "warn" },
];

const ALERT_STYLE: Record<(typeof ALERTS)[number]["kind"], string> = {
  ok: "text-up",
  down: "text-destructive",
  warn: "text-down",
};

export function StatusBand() {
  const [i, setI] = useState(0);
  const reduced = useReduced();
  useEffect(() => {
    if (reduced) return;
    const t = setInterval(() => setI((v) => (v + 1) % ALERTS.length), 3000);
    return () => clearInterval(t);
  }, [reduced]);
  const a = ALERTS[i];

  return (
    <div className="rounded-xl border border-line bg-surface/50 p-5">
      <div className="flex items-center justify-between">
        <span className="font-mono text-[10px] uppercase tracking-[0.14em] text-faint">
          status page · public
        </span>
        <span className="rounded border border-line px-1.5 py-0.5 font-mono text-[10px] text-faint">
          /status/lab
        </span>
      </div>
      <div className="mt-4 rounded-md bg-up/90 px-3 py-2 text-center text-[12.5px] font-medium text-[#052e16]">
        All systems operational
      </div>
      <div className="mt-4 space-y-2">
        {["edge", "api", "media"].map((s) => (
          <div key={s} className="flex items-center gap-2.5 font-mono text-[11px] text-dim">
            <span className="h-1.5 w-1.5 rounded-full bg-up animate-pulse-dot" />
            {s}
            <span className="ml-auto text-faint">operational</span>
          </div>
        ))}
      </div>
      <div className="mt-5 border-t border-line pt-3.5">
        <div className="flex items-center gap-2 font-mono text-[10.5px]">
          <span className="text-faint">{a.at}</span>
          <span className={`truncate ${ALERT_STYLE[a.kind]}`}>{a.text}</span>
          <span className="ml-auto shrink-0 rounded border border-line px-1.5 py-0.5 text-[9.5px] text-faint">
            → slack · ntfy · smtp
          </span>
        </div>
      </div>
    </div>
  );
}
