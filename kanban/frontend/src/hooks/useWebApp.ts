import { useCallback, useEffect, useState } from "react";
import { api, ApiError } from "../api/client";
import type { AuthUser } from "../types";

type AuthState = "loading" | "authenticated" | "error";

export type AuthFailReason =
  | "NO_BRIDGE"
  | "NO_INITDATA"
  | "AUTH_REJECTED"
  | "AUTH_ERROR";

interface WebAppState {
  user: AuthUser | null;
  platform: string | null;
  ready: boolean;
  state: AuthState;
  reason: AuthFailReason | null;
  retry: () => void;
}

// MAX Bridge can populate window.WebApp.initData asynchronously after the
// WebView opens the page, so a single synchronous read is not enough.
const BRIDGE_WAIT_MS = 6000;
const BRIDGE_POLL_MS = 250;

function sleep(ms: number): Promise<void> {
  return new Promise<void>((resolve) => {
    window.setTimeout(resolve, ms);
  });
}

function notifyBridgeReady(): void {
  try {
    window.WebApp?.ready?.();
  } catch {
    // Telling the bridge we are ready is best-effort and must never
    // block authentication.
  }
}

export function useWebApp(): WebAppState {
  const [user, setUser] = useState<AuthUser | null>(null);
  const [platform, setPlatform] = useState<string | null>(null);
  const [ready, setReady] = useState(false);
  const [state, setState] = useState<AuthState>("loading");
  const [reason, setReason] = useState<AuthFailReason | null>(null);
  const [attempt, setAttempt] = useState(0);

  useEffect(() => {
    let cancelled = false;

    const fail = (failReason: AuthFailReason, detail: string) => {
      if (cancelled) {
        return;
      }
      // Never log initData itself: only the failure stage and HTTP status.
      console.warn(`[auth] ${failReason}: ${detail}`);
      setUser(null);
      setReason(failReason);
      setState("error");
      setReady(true);
    };

    const initialize = async () => {
      setReady(false);
      setState("loading");
      setReason(null);
      setPlatform(window.WebApp?.platform || null);
      notifyBridgeReady();

      const isLocal =
        window.location.hostname === "localhost" ||
        window.location.hostname === "127.0.0.1";

      let initData = window.WebApp?.initData || "";
      let bridgeSeen = !!window.WebApp;

      if (!initData && !isLocal) {
        const deadline = Date.now() + BRIDGE_WAIT_MS;
        while (!cancelled && !initData && Date.now() < deadline) {
          await sleep(BRIDGE_POLL_MS);
          const current = window.WebApp;
          if (current) {
            bridgeSeen = true;
          }
          initData = current?.initData || "";
        }
        notifyBridgeReady();
      }
      if (cancelled) {
        return;
      }

      if (initData) {
        setPlatform(window.WebApp?.platform || null);
        let response;
        try {
          response = await api.auth.validate(initData);
        } catch (err) {
          const status =
            err instanceof ApiError ? `http-${err.status}` : "network";
          fail(
            err instanceof ApiError && (err.status === 401 || err.status === 403)
              ? "AUTH_REJECTED"
              : "AUTH_ERROR",
            `validate request failed (${status})`
          );
          return;
        }
        if (cancelled) {
          return;
        }
        if (!response.ok || !response.user) {
          fail("AUTH_REJECTED", "server did not accept initData");
          return;
        }
        setUser(response.user);
        setState("authenticated");
        setReady(true);
        return;
      }

      if (isLocal) {
        const localUser: AuthUser = {
          user_id: 1,
          username: "dev",
          display_name: "Dev User",
        };
        if (!cancelled) {
          setUser(localUser);
          setState("authenticated");
          setReady(true);
        }
        return;
      }

      fail(
        bridgeSeen ? "NO_INITDATA" : "NO_BRIDGE",
        bridgeSeen
          ? "bridge present but initData is empty"
          : "MAX bridge (window.WebApp) not detected"
      );
    };

    void initialize();
    return () => {
      cancelled = true;
    };
  }, [attempt]);

  useEffect(() => {
    if (platform) {
      document.documentElement.dataset.maxPlatform = platform;
    } else {
      delete document.documentElement.dataset.maxPlatform;
    }
    return () => {
      delete document.documentElement.dataset.maxPlatform;
    };
  }, [platform]);

  const retry = useCallback(() => setAttempt((value) => value + 1), []);

  return { user, platform, ready, state, reason, retry };
}
