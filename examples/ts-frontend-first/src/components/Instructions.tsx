import { useState } from "react";

const STEPS = [
  { cmd: "docker compose stop upstream", label: "① Stop backend → L1 cache" },
  { cmd: "docker compose restart proxy", label: "② Clear L1 → L2 cache (redacted)" },
  { cmd: "docker compose start upstream", label: "③ Restore → live again" },
];

function Copy({ text }: { text: string }) {
  const [ok, setOk] = useState(false);
  return (
    <button
      className="copy-btn"
      onClick={() => { navigator.clipboard.writeText(text); setOk(true); setTimeout(() => setOk(false), 1200); }}
    >
      {ok ? "✓" : "⎘"}
    </button>
  );
}

export function Instructions() {
  return (
    <div className="instructions">
      <strong>Try it</strong>
      <p className="inst-hint">Run a command — the badge updates live within ~2s, no refresh needed</p>
      {STEPS.map((s) => (
        <div key={s.cmd} className="inst-row">
          <span className="inst-label">{s.label}</span>
          <div className="inst-cmd">
            <code>{s.cmd}</code>
            <Copy text={s.cmd} />
          </div>
        </div>
      ))}
    </div>
  );
}
