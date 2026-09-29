import { useState } from "react";

export type StartTarget =
  | { kind: "board"; id: number }
  | { kind: "task"; id: number };

const START_PATTERN = /^(board|task)_(\d{1,19})$/;

export function parseStartParam(raw: unknown): StartTarget | null {
  if (typeof raw !== "string") {
    return null;
  }
  const match = START_PATTERN.exec(raw.trim());
  if (!match) {
    return null;
  }
  const id = Number(match[2]);
  if (!Number.isSafeInteger(id) || id <= 0) {
    return null;
  }
  return { kind: match[1] as "board" | "task", id };
}

function readRawStartParam(): unknown {
  const bridgeParam = window.WebApp?.initDataUnsafe?.start_param;
  if (typeof bridgeParam === "string" && bridgeParam !== "") {
    return bridgeParam;
  }
  try {
    const fromUrl = new URLSearchParams(window.location.search).get("startapp");
    if (fromUrl) {
      return fromUrl;
    }
  } catch {
    // URL parsing is best-effort; the bridge remains the source of truth.
  }
  return null;
}

export function useStartParam(): StartTarget | null {
  const [target] = useState<StartTarget | null>(() => parseStartParam(readRawStartParam()));
  return target;
}
