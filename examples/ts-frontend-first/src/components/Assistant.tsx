import { useAssistantStream } from "../useAssistantStream";

interface Query {
  label: string;
  path: string;
}

const QUERIES: Query[] = [
  { label: "Best headphones for office use?", path: "/api/assist/headphones" },
  { label: "Which keyboard for a developer?", path: "/api/assist/keyboard" },
  { label: "Hub for 4K monitor + USB-C laptop?", path: "/api/assist/hub" },
];

export function Assistant() {
  const { text, status, start } = useAssistantStream();

  return (
    <section className="assistant">
      <h2 className="section-title">Ask the assistant</h2>
      <div className="assistant-buttons">
        {QUERIES.map((q) => (
          <button
            key={q.path}
            className="assistant-query-btn"
            disabled={status === "streaming"}
            onClick={() => start(q.path)}
          >
            {q.label}
          </button>
        ))}
      </div>
      <div className="assistant-output">
        {status === "idle" && (
          <p className="assistant-empty">Click a question above to see SSE streaming in action.</p>
        )}
        {(status === "streaming" || status === "done") && (
          <p className="assistant-text">
            {text}
            {status === "streaming" && <span className="assistant-cursor" />}
          </p>
        )}
      </div>
    </section>
  );
}
