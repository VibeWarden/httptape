import { useEffect, useState } from "react";
import type { DataSource } from "./api";

const API_URL = import.meta.env.VITE_API_URL || "http://localhost:3001";

interface HealthSnapshot {
  state: "live" | "l1-cache" | "l2-cache";
  upstream_url: string;
  last_probed_at?: string;
  probe_interval_ms: number;
  since: string;
}

function toDataSource(state: HealthSnapshot["state"]): DataSource {
  return state === "live" ? "upstream" : state;
}

export function useHealthStream(): DataSource | null {
  const [source, setSource] = useState<DataSource | null>(null);

  useEffect(() => {
    const es = new EventSource(`${API_URL}/__httptape/health/stream`);
    es.onmessage = (e) => {
      try {
        const snap: HealthSnapshot = JSON.parse(e.data);
        setSource(toDataSource(snap.state));
      } catch {
        // ignore malformed events
      }
    };
    es.onerror = () => {
      console.warn("httptape health stream disconnected; EventSource will auto-reconnect");
    };
    return () => es.close();
  }, []);

  return source;
}
