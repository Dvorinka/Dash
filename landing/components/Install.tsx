import { CopyButton } from "./CopyButton";
import { Reveal } from "./Reveal";

const CURL = "curl -fsSL https://raw.githubusercontent.com/Dvorinka/Dash/main/install.sh | sh";
const DOCKER = "docker run -d -v ./data:/data -p 3000:3000 ghcr.io/dvorinka/dash:latest";

export function Install() {
  return (
    <div className="grid gap-2.5 md:grid-cols-2">
      <Reveal className="overflow-hidden rounded-xl border border-line bg-surface/50">
        <div className="flex items-center justify-between border-b border-line px-4 py-2.5">
          <span className="font-mono text-[10px] uppercase tracking-[0.14em] text-faint">
            one-liner
          </span>
          <CopyButton text={CURL} label="Copy install command" />
        </div>
        <pre className="overflow-x-auto p-4 font-mono text-[12.5px] leading-7 text-dim">
          <span className="text-text">$</span> curl -fsSL https://raw.githubusercontent.com/
          {"\n"}    Dvorinka/Dash/main/install.sh | sh{"\n"}
          <span className="text-faint"># installs into ./dash, serves :3000</span>
        </pre>
      </Reveal>
      <Reveal delay={0.08} className="overflow-hidden rounded-xl border border-line bg-surface/50">
        <div className="flex items-center justify-between border-b border-line px-4 py-2.5">
          <span className="font-mono text-[10px] uppercase tracking-[0.14em] text-faint">
            docker
          </span>
          <CopyButton text={DOCKER} label="Copy docker command" />
        </div>
        <pre className="overflow-x-auto p-4 font-mono text-[12.5px] leading-7 text-dim">
          <span className="text-text">$</span> docker run -d -v ./data:/data \{"\n"}
          {"    "}-p 3000:3000 \{"\n"}
          {"    "}ghcr.io/dvorinka/dash:latest
        </pre>
      </Reveal>
    </div>
  );
}
