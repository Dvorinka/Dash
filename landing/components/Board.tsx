"use client";

import { useEffect, useState } from "react";
import { motion } from "motion/react";

/* A live miniature of the product - rendered, not screenshotted. */

type Service = {
  name: string;
  host: string;
  letter: string;
  tint: string;
};

const MEDIA: Service[] = [
  { name: "Plex", host: "plex.localhost:32400", letter: "P", tint: "#e5a00d" },
  { name: "Jellyfin", host: "jellyfin.localhost:8096", letter: "J", tint: "#aa5cc3" },
  { name: "Immich", host: "immich.localhost:2283", letter: "I", tint: "#f16d9a" },
];

const INFRA: Service[] = [
  { name: "Proxmox", host: "proxmox.localhost:8006", letter: "P", tint: "#e57000" },
  { name: "Pi-hole", host: "pihole.localhost", letter: "P", tint: "#f05454" },
  { name: "Gitea", host: "gitea.localhost:3000", letter: "G", tint: "#609926" },
];

function useReduced() {
  const [reduced, setReduced] = useState(false);
  useEffect(() => {
    const mq = window.matchMedia("(prefers-reduced-motion: reduce)");
    setReduced(mq.matches);
    const fn = (e: MediaQueryListEvent) => setReduced(e.matches);
    mq.addEventListener("change", fn);
    return () => mq.removeEventListener("change", fn);
  }, []);
  return reduced;
}

function useClock() {
  const [now, setNow] = useState<Date | null>(null);
  const reduced = useReduced();
  useEffect(() => {
    setNow(new Date());
    if (reduced) return;
    const t = setInterval(() => setNow(new Date()), 1000);
    return () => clearInterval(t);
  }, [reduced]);
  return now;
}

function useWander(base: number, spread: number, ms: number) {
  const [v, setV] = useState(base);
  const reduced = useReduced();
  useEffect(() => {
    if (reduced) return;
    const t = setInterval(
      () => setV(Math.round(base + (Math.random() - 0.35) * spread)),
      ms,
    );
    return () => clearInterval(t);
  }, [base, spread, ms, reduced]);
  return v;
}

function StatusDot({ down = false }: { down?: boolean }) {
  return (
    <span
      className={`inline-block h-1.5 w-1.5 rounded-full ${down ? "bg-down" : "bg-up animate-pulse-dot"}`}
    />
  );
}

function Tile({
  i,
  className = "",
  children,
}: {
  i: number;
  className?: string;
  children: React.ReactNode;
}) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 10 }}
      whileInView={{ opacity: 1, y: 0 }}
      viewport={{ once: true }}
      transition={{ duration: 0.45, delay: 0.05 * i, ease: [0.2, 0, 0, 1] }}
      className={`rounded-lg border border-line bg-surface p-3 transition-colors duration-150 hover:border-line-strong hover:bg-surface-hover ${className}`}
    >
      {children}
    </motion.div>
  );
}

function ServiceTile({ s, i, down }: { s: Service; i: number; down?: boolean }) {
  return (
    <Tile i={i} className="col-span-3 flex items-center gap-3 sm:col-span-2">
      <span
        className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-iconbg text-sm font-semibold"
        style={{ color: s.tint }}
      >
        {s.letter}
      </span>
      <span className="min-w-0">
        <span className="block truncate text-[13px] font-medium">{s.name}</span>
        <span className="block truncate font-mono text-[10.5px] text-faint">{s.host}</span>
      </span>
      <span className="ml-auto flex items-center gap-1.5 rounded border border-line px-1.5 py-0.5 font-mono text-[10px] text-dim">
        <StatusDot down={down} />
        {down ? "degraded" : "up"}
      </span>
    </Tile>
  );
}

function ClockTile({ i }: { i: number }) {
  const now = useClock();
  return (
    <Tile i={i} className="col-span-3 sm:col-span-2">
      <div className="font-mono text-[10px] uppercase tracking-[0.14em] text-faint">local time</div>
      <div className="mt-1.5 font-mono text-xl font-medium tabular-nums">
        {now ? now.toLocaleTimeString("en-GB") : "--:--:--"}
      </div>
      <div className="mt-1 font-mono text-[10.5px] text-dim">Europe/Prague</div>
    </Tile>
  );
}

function PiholeTile({ i }: { i: number }) {
  const blocked = useWander(24, 6, 2600);
  return (
    <Tile i={i} className="col-span-3 sm:col-span-2">
      <div className="font-mono text-[10px] uppercase tracking-[0.14em] text-faint">pi-hole</div>
      <div className="mt-1.5 text-xl font-medium tabular-nums">
        {blocked.toFixed(1)}
        <span className="text-sm text-dim">%</span>
      </div>
      <div className="mt-1 font-mono text-[10.5px] text-dim">queries blocked</div>
    </Tile>
  );
}

function HeartbeatTile({ i }: { i: number }) {
  const [bars, setBars] = useState<number[]>(() => Array.from({ length: 26 }, () => 1));
  const ping = useWander(42, 30, 1500);
  const reduced = useReduced();
  useEffect(() => {
    if (reduced) return;
    const t = setInterval(() => {
      setBars((b) => [...b.slice(1), Math.random() > 0.06 ? 1 : 0]);
    }, 1100);
    return () => clearInterval(t);
  }, [reduced]);
  return (
    <Tile i={i} className="col-span-3 sm:col-span-2">
      <div className="flex items-baseline justify-between">
        <span className="font-mono text-[10px] uppercase tracking-[0.14em] text-faint">api</span>
        <span className="font-mono text-[10.5px] tabular-nums text-dim">{ping} ms</span>
      </div>
      <div className="mt-2 flex h-7 items-end gap-[3px]">
        {bars.map((v, k) => (
          <span
            key={k}
            className={`flex-1 rounded-[2px] ${v ? "bg-up" : "bg-down"} transition-opacity duration-300`}
            style={{ height: `${v ? 60 + ((k * 37) % 40) : 100}%`, opacity: v ? 0.85 : 1 }}
          />
        ))}
      </div>
    </Tile>
  );
}

function SystemTile({ i }: { i: number }) {
  const cpu = useWander(34, 26, 2200);
  return (
    <Tile i={i} className="col-span-3 sm:col-span-2">
      <div className="font-mono text-[10px] uppercase tracking-[0.14em] text-faint">nas-01</div>
      <div className="mt-1.5 flex items-baseline gap-1">
        <span className="text-xl font-medium tabular-nums">{cpu}%</span>
        <span className="font-mono text-[10.5px] text-dim">cpu</span>
      </div>
      <div className="mt-2 space-y-1">
        <div className="h-1 w-full overflow-hidden rounded bg-iconbg">
          <div className="h-full rounded bg-accent transition-all duration-700" style={{ width: `${Math.min(cpu, 100)}%` }} />
        </div>
        <div className="h-1 w-full overflow-hidden rounded bg-iconbg">
          <div className="h-full rounded bg-up/70 transition-all duration-700" style={{ width: "61%" }} />
        </div>
      </div>
    </Tile>
  );
}

function DomainTile({ i }: { i: number }) {
  return (
    <Tile i={i} className="col-span-3 sm:col-span-2">
      <div className="font-mono text-[10px] uppercase tracking-[0.14em] text-faint">tdvorak.dev</div>
      <div className="mt-1.5 text-xl font-medium">
        260<span className="text-sm text-dim">d</span>
      </div>
      <div className="mt-1 font-mono text-[10.5px] text-dim">until expiry · tls 81d</div>
    </Tile>
  );
}

export function Board() {
  const [warn, setWarn] = useState(-1);
  const reduced = useReduced();
  useEffect(() => {
    if (reduced) return;
    const t = setInterval(() => {
      const pick = Math.floor(Math.random() * MEDIA.length);
      setWarn(pick);
      setTimeout(() => setWarn(-1), 2600);
    }, 11000);
    return () => clearInterval(t);
  }, [reduced]);

  return (
    <div className="overflow-hidden rounded-xl border border-line-strong bg-bg shadow-[0_24px_80px_-24px_rgba(0,0,0,0.7)]">
      {/* window chrome */}
      <div className="flex h-9 items-center gap-2 border-b border-line bg-surface px-3.5">
        <span className="h-2.5 w-2.5 rounded-full bg-line-strong" />
        <span className="h-2.5 w-2.5 rounded-full bg-line-strong" />
        <span className="h-2.5 w-2.5 rounded-full bg-line-strong" />
        <span className="ml-2 rounded-md border border-line bg-bg px-2 py-0.5 font-mono text-[10px] text-faint">
          localhost:3000
        </span>
        <span className="ml-auto hidden items-center gap-1 rounded-md border border-line px-2 py-0.5 font-mono text-[10px] text-faint sm:flex">
          ⌘K
        </span>
      </div>

      <div className="p-3.5 sm:p-5">
        <div className="mb-2.5 flex items-center gap-3">
          <span className="font-mono text-[10px] font-semibold uppercase tracking-[0.14em] text-dim">
            Media
          </span>
          <span className="font-mono text-[10px] text-faint">05</span>
          <span className="h-px flex-1 bg-line" />
        </div>
        <div className="grid grid-cols-6 gap-2.5">
          {MEDIA.map((s, k) => (
            <ServiceTile key={s.name} s={s} i={k} down={warn === k} />
          ))}
          <ClockTile i={3} />
          <PiholeTile i={4} />
        </div>

        <div className="mb-2.5 mt-5 flex items-center gap-3">
          <span className="font-mono text-[10px] font-semibold uppercase tracking-[0.14em] text-dim">
            Infrastructure
          </span>
          <span className="font-mono text-[10px] text-faint">06</span>
          <span className="h-px flex-1 bg-line" />
        </div>
        <div className="grid grid-cols-6 gap-2.5">
          {INFRA.map((s, k) => (
            <ServiceTile key={s.name} s={s} i={k + 5} />
          ))}
          <HeartbeatTile i={8} />
          <SystemTile i={9} />
          <DomainTile i={10} />
        </div>
      </div>
    </div>
  );
}
