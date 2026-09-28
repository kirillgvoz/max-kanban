import { ArrowLeft, Wifi, WifiOff } from "lucide-react";
import type { AuthUser } from "../types";

type View =
  | { type: "orgs" }
  | { type: "boards"; orgId: number; orgName: string }
  | { type: "board"; boardId: number; orgId: number; orgName: string; boardName: string };

interface HeaderProps {
  view: View;
  onNavigate: (v: View) => void;
  isMobile: boolean;
  user: AuthUser | null;
  connected: boolean;
}

export function Header({ view, onNavigate, isMobile, user, connected }: HeaderProps) {
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
            <span className="header-logo-icon">⬡</span>
          </div>
        )}

        <div className="header-title">
          {view.type === "orgs" && "Организации"}
          {view.type === "boards" && view.orgName}
          {view.type === "board" && view.boardName}
        </div>
      </div>

      <div className="header-right">
        <div className={`header-status ${connected ? "header-status--online" : ""}`}>
          {connected ? <Wifi size={14} /> : <WifiOff size={14} />}
        </div>
      </div>
    </header>
  );
}
