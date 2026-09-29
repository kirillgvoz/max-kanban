import { ArrowLeft, CircleHelp, Wifi, WifiOff } from "lucide-react";
import type { AuthUser } from "../types";

type View =
  | { type: "orgs" }
  | { type: "boards"; orgId: number; orgName: string }
  | { type: "board"; boardId: number; orgId: number; orgName: string; boardName: string };

interface HeaderProps {
  view: View;
  onNavigate: (v: View) => void;
  onOpenGuide: () => void;
  isMobile: boolean;
  user: AuthUser | null;
  connected: boolean;
}

export function Header({ view, onNavigate, onOpenGuide, isMobile, user, connected }: HeaderProps) {
  return (
    <header className="header">
      <div className="header-left">
        {isMobile && view.type !== "orgs" && (
          <button
            className="header-back"
            onClick={() => {
              if (view.type === "board") {
                onNavigate({ type: "boards", orgId: view.orgId, orgName: view.orgName });
              } else {
                onNavigate({ type: "orgs" });
              }
            }}
          >
            <ArrowLeft size={20} />
          </button>
        )}

        {isMobile && (
          <div className="header-logo">
            <span className="brand-mark brand-mark--sm" aria-hidden="true">T</span>
          </div>
        )}

        <div className="header-title">
          {view.type === "orgs" && "Организации"}
          {view.type === "boards" && view.orgName}
          {view.type === "board" && view.boardName}
        </div>
      </div>

      <div className="header-right">
        <button className="header-back" onClick={onOpenGuide} aria-label="Подключение чата">
          <CircleHelp size={20} />
        </button>
        <div className={`header-status ${connected ? "header-status--online" : ""}`}>
          {connected ? <Wifi size={14} /> : <WifiOff size={14} />}
        </div>
      </div>
    </header>
  );
}
