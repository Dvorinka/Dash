"use client";

import { useEffect, useRef, useState } from "react";
import { motion } from "motion/react";

/* Same state, four skins - the product's renderer seam, playable in-page. */

type Mode = "bento" | "cards" | "index" | "console";
const MODES: Mode[] = ["bento", "cards", "index", "console"];

const ITEMS = [
  { name: "Plex", host: "plex.local", letter: "P", tint: "#e5a00d" },
  { name: "Jellyfin", host: "jf.local", letter: "J", tint: "#aa5cc3" },
  { name: "Pi-hole", host: "pihole.local", letter: "P", tint: "#f05454" },
  { name: "Gitea", host: "git.local", letter: "G", tint: "#609926" },
  { name: "Grafana", host: "graf.local", letter: "G", tint: "#f46800" },
];

const BENTO_SPAN = ["sm:col-span-2", "sm:col-span-2", "sm:col-span-2", "sm:col-span-3", "sm:col-span-3"];

function Chip({ up = true }: { up?: boolean }) {
  return (
    <span className="flex items-center gap-1.5 rounded border border-line px-1.5 py-0.5 font-mono text-[10px] text-dim">
      <span className={`h-1.5 w-1.5 rounded-full ${up ? "bg-up animate-pulse-dot" : "bg-down"}`} />
      up
    </span>
  );
}

export function RendererDemo() {
  const [mode, setMode] = useState<Mode>("bento");
  const locked = useRef(false);

  useEffect(() => {
    const t = setInterval(() => {
      if (!locked.current) {
        setMode((m) => MODES[(MODES.indexOf(m) + 1) % MODES.length]);
      }
    }, 3400);
    return () => clearInterval(t);
  }, []);

  const pick = (m: Mode) => {
    locked.current = true;
    setMode(m);
  };

  const gridCls =
    mode === "bento"
      ? "grid grid-cols-2 gap-2 sm:grid-cols-6"
      : mode === "cards"
        ? "grid grid-cols-2 gap-2 sm:grid-cols-5"
        : mode === "console"
          ? "grid grid-cols-1 gap-2 sm:grid-cols-3"
          : "flex flex-col divide-y divide-line";

  return (
    <div className="rounded-xl border border-line bg-bg">
      <div className="flex items-center gap-2 border-b border-line px-4 py-3">
        <span className="font-mono text-[10px] uppercase tracking-[0.14em] text-faint">
          same board, four renderers
        </span>
        <div className="ml-auto flex gap-1">
          {MODES.map((m) => (
            <button
              key={m}
              type="button"
              aria-pressed={mode === m}
              onClick={() => pick(m)}
              className={`rounded-md border px-2.5 py-1 font-mono text-[11px] transition-colors duration-150 ${
                mode === m
                  ? "border-line-strong bg-surface-hover text-text"
                  : "border-transparent text-faint hover:text-dim"
              }`}
            >
              {m}
            </button>
          ))}
        </div>
      </div>

      <div className="p-4">
        <div className={gridCls}>
          {ITEMS.map((it, k) => (
            <motion.div
              key={it.name}
              layout
              transition={{ duration: 0.5, ease: [0.2, 0, 0, 1] }}
              className={mode === "index" ? "" : mode === "bento" ? `col-span-1 ${BENTO_SPAN[k]}` : "col-span-1"}
            >
              <motion.div
                key={mode}
                initial={{ opacity: 0.4 }}
                animate={{ opacity: 1 }}
                transition={{ duration: 0.3 }}
                className={
                  mode === "index"
                    ? "flex items-center gap-4 py-3"
                    : mode === "console"
                      ? "flex items-center gap-3 rounded-[4px] border border-line px-3 py-2.5 font-mono"
                      : "flex items-center gap-3 rounded-lg border border-line bg-surface p-3"
                }
              >
                {mode === "index" ? (
                  <>
                    <span className="font-mono text-[11px] text-faint">0{k + 1}</span>
                    <span className="text-[14px] font-medium tracking-tight">{it.name}</span>
                    <span className="hidden font-mono text-[11px] text-faint sm:block">{it.host}</span>
                    <span className="ml-auto"><Chip /></span>
                  </>
                ) : mode === "console" ? (
                  <>
                    <span className="text-[11px] text-text">{it.name.toLowerCase()}</span>
                    <span className="text-[10px] text-faint">{it.host}</span>
                    <span className="ml-auto text-[10px] text-up">[ up ]</span>
                  </>
                ) : (
                  <>
                    <span
                      className={`flex shrink-0 items-center justify-center rounded-md bg-iconbg font-semibold ${mode === "cards" ? "h-7 w-7 text-[11px]" : "h-9 w-9 text-sm"}`}
                      style={{ color: it.tint }}
                    >
                      {it.letter}
                    </span>
                    <span className="min-w-0">
                      <span className={`block truncate font-medium ${mode === "cards" ? "text-[12px]" : "text-[13px]"}`}>
                        {it.name}
                      </span>
                      {mode === "bento" && (
                        <span className="block truncate font-mono text-[10.5px] text-faint">{it.host}</span>
                      )}
                    </span>
                    <span className="ml-auto"><Chip /></span>
                  </>
                )}
              </motion.div>
            </motion.div>
          ))}
        </div>
      </div>
    </div>
  );
}
