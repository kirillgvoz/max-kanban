import { useState } from "react";
import { ClipboardList, Plus } from "lucide-react";
import { useBoards } from "../hooks/useBoards";
import { api } from "../api/client";
import { CreateBoardModal, type BoardStatusDraft } from "./CreateBoardModal";
import { OrgSettings } from "./OrgSettings";
import type { AuthUser } from "../types";

type View =
  | { type: "orgs" }
  | { type: "boards"; orgId: number; orgName: string }
  | { type: "board"; boardId: number; orgId: number; orgName: string; boardName: string };

interface BoardListProps {
  orgId: number;
  orgName: string;
  onNavigate: (v: View) => void;
  user: AuthUser | null;
}

export function BoardList({ orgId, orgName, onNavigate, user }: BoardListProps) {
  const { boards, loading, includeArchived, setIncludeArchived, createBoard, refetch } = useBoards(orgId);
  const [showSettings, setShowSettings] = useState(false);
  const [showCreate, setShowCreate] = useState(false);

  const handleRestore = async (id: number) => {
    await api.boards.update(id, { is_archived: false });
    await refetch();
  };

  const handleCreate = async (data: { name: string; description?: string; columns?: BoardStatusDraft[] }) => {
    const board = await createBoard(data.name, data.description, data.columns);
    if (board) {
      setShowCreate(false);
      onNavigate({
        type: "board",
        boardId: board.id,
        orgId,
        orgName,
        boardName: board.name,
      });
    }
  };

  if (loading) {
    return (
      <div className="content-loading">
        <div className="loading-spinner" />
      </div>
    );
  }

  return (
    <div className="board-list">
      <div className="content-header">
        <h1 className="content-title">{orgName}</h1>
        <div className="content-actions">
          <label className="filter-checkbox"><input type="checkbox" checked={includeArchived} onChange={(event) => setIncludeArchived(event.target.checked)} /> Архив</label>
          <button className="btn btn--ghost" onClick={() => setShowSettings(true)}>
            Настройки
          </button>
          <button className="btn btn--primary" onClick={() => setShowCreate(true)}>
            <Plus size={16} />
            Доска
          </button>
        </div>
      </div>

      {showSettings && <OrgSettings orgId={orgId} user={user} onClose={() => setShowSettings(false)} />}

      {showCreate && (
        <CreateBoardModal onSubmit={handleCreate} onClose={() => setShowCreate(false)} />
      )}

      <div className="board-grid">
        {boards.map((board) => (
          <div
            key={board.id}
            className={`board-card ${board.is_archived ? "board-card--archived" : ""}`}
            onClick={() => {
              if (board.is_archived) return;
              onNavigate({
                type: "board",
                boardId: board.id,
                orgId,
                orgName,
                boardName: board.name,
              });
            }}
          >
            <div className="board-card-name">{board.name} {board.is_archived && <span className="board-card-archived">Архив</span>}</div>
            {board.description && (
              <div className="board-card-desc">{board.description}</div>
            )}
            <div className="board-card-meta">
              {new Date(board.created_at).toLocaleDateString("ru-RU")}
              {board.is_archived && (
                <button className="btn btn--ghost btn--sm" onClick={(event) => {
                  event.stopPropagation();
                  void handleRestore(board.id);
                }}>Восстановить</button>
              )}
            </div>
          </div>
        ))}

        {boards.length === 0 && !showCreate && (
          <div className="empty-state">
            <ClipboardList size={48} className="empty-state-icon" aria-hidden="true" />
            <div className="empty-state-text">Нет досок</div>
            <div className="empty-state-hint">Создайте первую доску</div>
          </div>
        )}
      </div>
    </div>
  );
}
