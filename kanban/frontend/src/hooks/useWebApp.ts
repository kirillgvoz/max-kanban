import { useCallback, useEffect, useState } from "react";
import { api } from "../api/client";
import type { AuthUser } from "../types";

type AuthState = "loading" | "authenticated" | "error";

interface WebAppState {
  user: AuthUser | null;
  platform: string | null;
  ready: boolean;
  state: AuthState;
  retry: () => void;
}

export function useWebApp(): WebAppState {
  const [user, setUser] = useState<AuthUser | null>(null);
  const [platform, setPlatform] = useState<string | null>(null);
  const [ready, setReady] = useState(false);
  const [state, setState] = useState<AuthState>("loading");
  const [attempt, setAttempt] = useState(0);

  useEffect(() => {
    let cancelled = false;
    const bridge = window.WebApp;
    const initData = bridge?.initData || "";

    const initialize = async () => {
      setReady(false);
      setState("loading");
      setPlatform(bridge?.platform || null);
      try {
        if (initData) {
          const response = await api.auth.validate(initData);
          if (!response.ok || !response.user) {
            throw new Error(response.error || "Authentication failed");
          }
          if (!cancelled) {
            setUser(response.user);
            setState("authenticated");
            setReady(true);
          }
          return;
        }

        const isLocal =
          window.location.hostname === "localhost" ||
          window.location.hostname === "127.0.0.1";
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

        throw new Error("Open Max Kanban from MAX to continue");
      } catch {
        if (!cancelled) {
          setUser(null);
          setState("error");
          setReady(true);
        }
      }
    };

    try {
      bridge?.ready?.();
    } catch {
      return;
    }
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

  return { user, platform, ready, state, retry };
}
