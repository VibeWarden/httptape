import type { DataSource } from "../api";

interface Props {
  source: DataSource | null;
}

export function ArchitectureDiagram({ source }: Props) {
  const isLive = !source || source === "upstream";
  const isL1 = source === "l1-cache";
  const upstreamDown = !isLive;

  return (
    <div className="arch">
      <div className="arch-row">
        <span className="arch-node arch-on">App</span>
        <span className="arch-link arch-on">→</span>
        <span className="arch-node arch-proxy">
          <img src="/logo.png" alt="httptape" className="arch-logo" />
        </span>
        <span className={`arch-link ${upstreamDown ? "arch-off" : "arch-on"}`}>
          {upstreamDown ? "✕" : "→"}
        </span>
        <span className={`arch-node ${upstreamDown ? "arch-off" : "arch-on"}`}>
          API {upstreamDown && "⏸"}
        </span>
      </div>
      {upstreamDown && (
        <div className="arch-fallback">
          ↳ serving from {isL1 ? "memory (L1)" : "disk (L2, redacted)"}
        </div>
      )}
    </div>
  );
}
