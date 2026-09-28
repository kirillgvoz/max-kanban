import { useEffect, useState } from "react";

interface WebApp {
  initData: string;
  initDataUnsafe: Record<string, any>;
  platform: string;
  version: string;
  ready: () => void;
  expand: () => void;
  close: () => void;
  switchInlineQuery: (query: string, chat_types?: string[]) => void;
  openLink: (url: string) => void;
}

declare global {
  interface Window {
    WebApp?: WebApp;
  }
}

export function useWebApp() {
  const [webApp, setWebApp] = useState<WebApp | null>(null);
  const [user, setUser] = useState<any>(null);

  useEffect(() => {
    if (typeof window !== "undefined" && window.WebApp) {
      const wa = window.WebApp;
      setWebApp(wa);
      wa.ready();
      if (wa.initDataUnsafe?.user) {
        setUser(wa.initDataUnsafe.user);
      }
    }
  }, []);

  return { webApp, user };
}
