import { useCallback, useEffect, useRef, useState } from "react";
import { api } from "../api/client";
import type { WSEvent } from "../types";

export function useWebSocket(orgId: number | null, onReconnect?: () => void) {
  const socketRef = useRef<WebSocket | null>(null);
  const reconnectTimer = useRef<number | null>(null);
  const reconnectAttempt = useRef(0);
  const mounted = useRef(true);
  const [connected, setConnected] = useState(false);
  const listenersRef = useRef<Map<string, Set<(data: any) => void>>>(new Map());

  useEffect(() => {
    mounted.current = true;
    if (!orgId) {
      setConnected(false);
      return () => {
        mounted.current = false;
      };
    }

    let cancelled = false;
    const connect = async () => {
      if (cancelled) return;
      try {
        const { ticket } = await api.ws.ticket(orgId);
        if (cancelled) return;
        const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
        const url = new URL(`${import.meta.env.BASE_URL}ws`, window.location.origin);
        url.protocol = protocol;
        url.searchParams.set("ticket", ticket);
        const socket = new WebSocket(url.toString());
        socketRef.current = socket;

        socket.onopen = () => {
          if (cancelled) return;
          reconnectAttempt.current = 0;
          setConnected(true);
          onReconnect?.();
        };
        socket.onmessage = (event) => {
          try {
            const message: WSEvent = JSON.parse(event.data);
            listenersRef.current.get(message.type)?.forEach((handler) => handler(message.data));
            listenersRef.current.get("*")?.forEach((handler) => handler(message));
          } catch {
            return;
          }
        };
        socket.onerror = () => socket.close();
        socket.onclose = () => {
          if (cancelled) return;
          setConnected(false);
          socketRef.current = null;
          const delay = Math.min(1000 * 2 ** reconnectAttempt.current, 15000);
          reconnectAttempt.current += 1;
          reconnectTimer.current = window.setTimeout(connect, delay);
        };
      } catch {
        if (cancelled) return;
        const delay = Math.min(1000 * 2 ** reconnectAttempt.current, 15000);
        reconnectAttempt.current += 1;
        reconnectTimer.current = window.setTimeout(connect, delay);
      }
    };

    void connect();
    return () => {
      cancelled = true;
      if (reconnectTimer.current) window.clearTimeout(reconnectTimer.current);
      socketRef.current?.close();
      socketRef.current = null;
      setConnected(false);
    };
  }, [orgId, onReconnect]);

  const subscribe = useCallback((type: string, handler: (data: any) => void) => {
    if (!listenersRef.current.has(type)) {
      listenersRef.current.set(type, new Set());
    }
    listenersRef.current.get(type)!.add(handler);
    return () => {
      listenersRef.current.get(type)?.delete(handler);
    };
  }, []);

  return { connected, subscribe };
}
