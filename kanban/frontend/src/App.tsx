import { useState, useCallback } from "react";
import { useWebApp } from "./hooks/useWebApp";
import { useIsMobile } from "./hooks/useIsMobile";
import { useWebSocket } from "./hooks/useWebSocket";
import { Sidebar } from "./components/Sidebar";
import { OrgList } from "./components/OrgList";
import { BoardList } from "./components/BoardList";
import { Board } from "./components/Board";
import { Header } from "./components/Header";
import type { AuthUser, Task, Board as BoardType, Column } from "./types";
import "./styles/index.scss";

type View =
  | { type: "orgs" }
  | { type: "boards"; orgId: number; orgName: string }
  | { type: "board"; boardId: number; orgId: number; orgName: string; boardName: string };

export default function App() {
  const { user, ready, state, retry } = useWebApp();
  const isMobile = useIsMobile();
  const [view, setView] = useState<View>({ type: "orgs" });
  const [refreshKey, setRefreshKey] = useState(0);

  const handleTaskUpdate = useCallback(() => {
    setRefreshKey((key) => key + 1);
  }, []);
  const currentOrgId = view.type === "boards" ? view.orgId : view.type === "board" ? view.orgId : null;
  const { connected, subscribe } = useWebSocket(currentOrgId, handleTaskUpdate);

  const navigate = useCallback((v: View) => {
    setView(v);
  }, []);

  if (!ready) {
    return (
      <div className="app-loading">
        <div className="loading-spinner" />
      </div>
    );
  }

  if (state === "error") {
    return (
      <div className="auth-error">
        <div className="auth-error-title">Нужен вход через MAX</div>
        <div className="auth-error-text">Откройте TaskFlow из бота в мессенджере.</div>
        <button className="btn btn--primary" onClick={retry}>Повторить</button>
      </div>
    );
  }

  return (
    <div className={`app ${isMobile ? "app--mobile" : ""}`}>
      {!isMobile && (
        <Sidebar
          currentView={view}
          onNavigate={navigate}
          user={user}
          connected={connected}
        />
      )}

      <div className="app-main">
        <Header
          view={view}
          onNavigate={navigate}
          isMobile={isMobile}
          user={user}
          connected={connected}
        />

        <div className="app-content">
          {view.type === "orgs" && (
            <OrgList user={user} onNavigate={navigate} />
          )}

          {view.type === "boards" && (
            <BoardList
              orgId={view.orgId}
              orgName={view.orgName}
              onNavigate={navigate}
              user={user}
            />
          )}

          {view.type === "board" && (
            <Board
              key={`${view.boardId}-${refreshKey}`}
              boardId={view.boardId}
              orgId={view.orgId}
              orgName={view.orgName}
              boardName={view.boardName}
              user={user}
              isMobile={isMobile}
              subscribe={subscribe}
              onTaskUpdate={handleTaskUpdate}
            />
          )}
        </div>
      </div>
    </div>
  );
}
