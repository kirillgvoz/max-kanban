import { useEffect, useState } from "react";
import { X } from "lucide-react";
import { api, ApiError } from "../api/client";
import type { AuthUser, BoardDetail, BoardChat, Column, Task } from "../types";

interface BoardSettingsProps {
  board: BoardDetail;
  tasks: Task[];
  user: AuthUser | null;
  canManage: boolean;
  onChanged: () => Promise<void> | void;
  onArchived: () => void;
  onClose: () => void;
}

type Tab = "board" | "statuses" | "chats";

export function BoardSettings({ board, tasks, user, canManage, onChanged, onArchived, onClose }: BoardSettingsProps) {
  const [tab, setTab] = useState<Tab>("board");
  const [name, setName] = useState(board.name);
  const [description, setDescription] = useState(board.description || "");
  const [statusName, setStatusName] = useState("");
  const [statusColor, setStatusColor] = useState("#6B7280");
  const [chatId, setChatId] = useState("");
  const [chatTitle, setChatTitle] = useState("");
  const [chats, setChats] = useState<BoardChat[]>([]);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => event.key === "Escape" && onClose();
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [onClose]);

  useEffect(() => {
    if (tab !== "chats" || !canManage) return;
    void (async () => {
      try {
        setChats(await api.boards.chats.list(board.id));
      } catch {
        setError("Не удалось загрузить чаты");
      }
    })();
  }, [tab, canManage, board.id]);

  const mutate = async (action: () => Promise<void>) => {
    setPending(true);
    setError("");
    try {
      await action();
      await onChanged();
    } catch (reason) {
      setError(reason instanceof ApiError ? reason.message : "Не удалось сохранить изменения");
    } finally {
      setPending(false);
    }
  };

  const saveBoard = () => mutate(async () => {
    if (!name.trim()) return;
    await api.boards.update(board.id, { name: name.trim(), description: description.trim() });
  });

  const archiveBoard = () => {
    if (!window.confirm("Архивировать доску?")) return;
    void mutate(async () => {
      await api.boards.update(board.id, { is_archived: true });
      onArchived();
    });
  };

  const addStatus = () => mutate(async () => {
    if (!statusName.trim() || board.columns.length >= 30) return;
    await api.columns.create(board.id, { name: statusName.trim(), color: statusColor });
    setStatusName("");
  });

  const saveStatus = (column: Column, updates: { name?: string; color?: string }) => mutate(async () => {
    await api.columns.update(column.id, updates);
  });

  const moveStatus = (index: number, direction: -1 | 1) => mutate(async () => {
    const ordered = [...board.columns].sort((a, b) => a.position - b.position);
    const next = index + direction;
    if (next < 0 || next >= ordered.length) return;
    [ordered[index], ordered[next]] = [ordered[next], ordered[index]];
    await api.columns.reorder(board.id, ordered.map((column) => column.id));
  });

  const removeStatus = (column: Column) => {
    const taskCount = tasks.filter((task) => task.column_id === column.id).length;
    if (taskCount > 0) {
      setError("Сначала переместите все задачи из этого статуса");
      return;
    }
    if (!window.confirm(`Удалить статус «${column.name}»?`)) return;
    void mutate(async () => {
      await api.columns.delete(column.id);
    });
  };

  const addChat = () => mutate(async () => {
    const parsed = Number(chatId);
    if (!Number.isInteger(parsed) || parsed <= 0) {
      throw new Error("Invalid chat");
    }
    const chat = await api.boards.chats.create(board.id, { chat_id: parsed, title: chatTitle.trim() || undefined });
    setChats((previous) => [...previous, chat]);
    setChatId("");
    setChatTitle("");
  });

  const removeChat = (id: number) => {
    if (!window.confirm("Отвязать чат?")) return;
    void mutate(async () => {
      await api.boards.chats.remove(id);
      setChats((previous) => previous.filter((chat) => chat.id !== id));
    });
  };

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-content" role="dialog" aria-modal="true" aria-label="Настройки доски" onClick={(event) => event.stopPropagation()}>
        <div className="modal-header">
          <h3 className="modal-header-title">Настройки доски</h3>
          <button className="modal-close" onClick={onClose} aria-label="Закрыть"><X size={20} /></button>
        </div>
        <div className="modal-tabs" role="tablist" aria-label="Разделы настроек">
          {([["board", "Доска"], ["statuses", "Статусы"], ["chats", "Чаты"]] as Array<[Tab, string]>).map(([value, label]) => (
            <button key={value} role="tab" aria-selected={tab === value} className={`modal-tab ${tab === value ? "modal-tab--active" : ""}`} onClick={() => setTab(value)}>{label}</button>
          ))}
        </div>
        <div className="modal-body">
          {error && <div className="form-error">{error}</div>}
          {tab === "board" && (
            <>
              <div className="form-group"><label className="form-label" htmlFor="board-name">Название</label><input id="board-name" className="input" value={name} onChange={(event) => setName(event.target.value)} disabled={!canManage} /></div>
              <div className="form-group"><label className="form-label" htmlFor="board-description">Описание</label><textarea id="board-description" className="textarea" value={description} onChange={(event) => setDescription(event.target.value)} rows={3} disabled={!canManage} /></div>
              {canManage && (
                <div className="modal-section-actions">
                  <button className="btn btn--primary" onClick={saveBoard} disabled={pending}>Сохранить</button>
                  <button className="btn btn--ghost" onClick={archiveBoard} disabled={pending}>Архивировать</button>
                </div>
              )}
            </>
          )}
          {tab === "statuses" && (
            <>
              {[...board.columns].sort((a, b) => a.position - b.position).map((column, index, ordered) => (
                <div className="board-status-row" key={column.id}>
                  <div className="comment">
                    <div className="comment-header">
                    <input className="input input--sm" defaultValue={column.name} aria-label={`Название статуса ${column.id}`} disabled={!canManage} onBlur={(event) => {
                      if (event.target.value.trim() && event.target.value !== column.name) saveStatus(column, { name: event.target.value.trim() });
                    }} />
                    <input type="color" value={column.color} aria-label={`Цвет статуса ${column.id}`} disabled={!canManage} onChange={(event) => saveStatus(column, { color: event.target.value })} />
                    {canManage && (
                      <>
                        <button className="btn btn--ghost btn--sm" onClick={() => moveStatus(index, -1)} disabled={pending || index === 0} aria-label={`Переместить ${column.name} влево`}>←</button>
                        <button className="btn btn--ghost btn--sm" onClick={() => moveStatus(index, 1)} disabled={pending || index === ordered.length - 1} aria-label={`Переместить ${column.name} вправо`}>→</button>
                        <button className="btn btn--ghost btn--sm" onClick={() => removeStatus(column)} disabled={pending}>Удалить</button>
                      </>
                    )}
                  </div>
                  </div>
                </div>
              ))}
              {canManage && board.columns.length < 30 && (
                <div className="form-row">
                  <div className="form-group form-group--half"><label className="form-label" htmlFor="status-name">Новый статус</label><input id="status-name" className="input" value={statusName} onChange={(event) => setStatusName(event.target.value)} /></div>
                  <div className="form-group"><label className="form-label" htmlFor="status-color">Цвет</label><input id="status-color" type="color" value={statusColor} onChange={(event) => setStatusColor(event.target.value)} /></div>
                  <div className="form-group form-group--end"><button className="btn btn--primary" onClick={addStatus} disabled={pending || !statusName.trim()}>Добавить</button></div>
                </div>
              )}
            </>
          )}
          {tab === "chats" && (
            <>
              {chats.map((chat) => (
                <div className="comment" key={chat.id}>
                  <div className="comment-header">
                    <span className="comment-author">{chat.title || `Чат ${chat.chat_id}`}</span>
                    <span className="comment-date">{chat.chat_id}</span>
                    {canManage && <button className="comment-delete" onClick={() => removeChat(chat.id)} aria-label={`Отвязать чат ${chat.chat_id}`}>Отвязать</button>}
                  </div>
                </div>
              ))}
              {chats.length === 0 && <div className="muted-text">Чаты пока не привязаны</div>}
              {canManage && (
                <div className="form-row">
                  <div className="form-group form-group--half"><label className="form-label" htmlFor="chat-id">Chat ID</label><input id="chat-id" className="input" value={chatId} onChange={(event) => setChatId(event.target.value)} /></div>
                  <div className="form-group form-group--half"><label className="form-label" htmlFor="chat-title">Название</label><input id="chat-title" className="input" value={chatTitle} onChange={(event) => setChatTitle(event.target.value)} /></div>
                  <div className="form-group form-group--end"><button className="btn btn--primary" onClick={addChat} disabled={pending}>Привязать</button></div>
                </div>
              )}
            </>
          )}
        </div>
      </div>
    </div>
  );
}
