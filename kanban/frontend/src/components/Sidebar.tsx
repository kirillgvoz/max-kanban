import { Building2, ChevronLeft } from "lucide-react";

import type { AuthUser } from "../types";

type View =
  | { type: "orgs" }
  | { type: "boards"; orgId: number; orgName: string }
  | { type: "board"; boardId: number; orgId: number; orgName: string; boardName: string };

interface SidebarProps {
  currentView: View;
  onNavigate: (v: View) => void;
  user: AuthUser | null;
  connected: boolean;
}

export function Sidebar({ currentView, onNavigate, user, connected }: SidebarProps) {
  return (
    <aside className="sidebar">
      <div className="sidebar-header">
        <div className="sidebar-logo">
          <span className="brand-mark" aria-hidden="true">T</span>
          <span className="sidebar-logo-text">Max Канбан</span>
        </div>
      </div>

      <nav className="sidebar-nav">
        <button
          className={`sidebar-item ${currentView.type === "orgs" ? "sidebar-item--active" : ""}`}
          onClick={() => onNavigate({ type: "orgs" })}
        >
          <Building2 size={18} className="sidebar-item-icon" aria-hidden="true" />
          Организации
        </button>

        {currentView.type !== "orgs" && (
          <button
            className="sidebar-item sidebar-item--back"
            onClick={() => onNavigate({ type: "orgs" })}
          >
            <ChevronLeft size={16} aria-hidden="true" /> Все организации
          </button>
        )}

        {currentView.type === "boards" && (
          <div className="sidebar-section">
            <div className="sidebar-section-title">{currentView.orgName}</div>
          </div>
        )}

        {currentView.type === "board" && (
          <div className="sidebar-section">
            <button
              className="sidebar-item sidebar-item--back"
              onClick={() => onNavigate({ type: "boards", orgId: currentView.orgId, orgName: currentView.orgName })}
            >
              <ChevronLeft size={16} aria-hidden="true" /> {currentView.orgName}
            </button>
            <div className="sidebar-section-title">{currentView.boardName}</div>
          </div>
        )}
      </nav>

      <div className="sidebar-footer">
        <div className="sidebar-status">
          <span className={`status-dot ${connected ? "status-dot--online" : "status-dot--offline"}`} />
          {connected ? "Онлайн" : "Офлайн"}
        </div>
        {user && (
          <div className="sidebar-user">
            <span className="sidebar-user-name">{user.display_name}</span>
          </div>
        )}
      </div>
    </aside>
  );
}
