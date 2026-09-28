import { useEffect, useState } from "react";
import { X, Check, Trash2 } from "lucide-react";
import { api, ApiError } from "../api/client";
import type { AuthUser, Column, OrgMember, Task, TaskDetail } from "../types";

interface TaskModalProps {
  taskId: number;
  columns: Column[];
  members: OrgMember[];
  onClose: () => void;
  user: AuthUser | null;
  onDeleted: () => void;
  onChanged: (task: Task) => void;
}

const PRIORITY_OPTIONS = [
  { value: "low", label: "Низкий" },
  { value: "medium", label: "Средний" },
  { value: "high", label: "Высокий" },
  { value: "urgent", label: "Срочный" },
];

export function TaskModal({ taskId, columns, members, onClose, user, onDeleted, onChanged }: TaskModalProps) {
  const [task, setTask] = useState<TaskDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);
  const [newComment, setNewComment] = useState("");
  const [newItems, setNewItems] = useState<Record<number, string>>({});
  const [editingTitle, setEditingTitle] = useState(false);
  const [title, setTitle] = useState("");
  const [editingDescription, setEditingDescription] = useState(false);
  const [description, setDescription] = useState("");
  const [assigneeId, setAssigneeId] = useState("");

  const load = async () => {
    setLoading(true);
    try {
      const data = await api.tasks.get(taskId);
      setTask(data);
      setTitle(data.title);
      setDescription(data.description || "");
      setError("");
    } catch (reason) {
      setError(reason instanceof ApiError && reason.status === 404 ? "Задача не найдена" : "Не удалось загрузить задачу");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void load();
  }, [taskId]);

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") onClose();
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [onClose]);

  const mutate = async (action: () => Promise<void>) => {
    setSaving(true);
    setError("");
    try {
      await action();
    } catch {
      setError("Не удалось сохранить изменения");
    } finally {
      setSaving(false);
    }
  };

  const saveTitle = () => mutate(async () => {
    if (!task || !title.trim()) return;
    const updated = await api.tasks.update(task.id, { title: title.trim() });
    setTask({ ...task, ...updated });
    onChanged(updated);
    setEditingTitle(false);
  });

  const saveDescription = () => mutate(async () => {
    if (!task) return;
    const updated = await api.tasks.update(task.id, { description });
    setTask({ ...task, ...updated });
    onChanged(updated);
    setEditingDescription(false);
  });

  const changePriority = (priority: string) => mutate(async () => {
    if (!task) return;
    const updated = await api.tasks.update(task.id, { priority });
    setTask({ ...task, ...updated });
    onChanged(updated);
  });

  const changeDeadline = (deadline: string) => mutate(async () => {
    if (!task) return;
    const updated = await api.tasks.update(task.id, { deadline: deadline || null });
    setTask({ ...task, ...updated });
    onChanged(updated);
  });

  const changeStatus = (columnId: number) => mutate(async () => {
    if (!task || task.column_id === columnId) return;
    const updated = await api.tasks.move(task.id, { column_id: columnId, position: 0 });
    const detail = await api.tasks.get(task.id);
    setTask(detail);
    onChanged(updated);
  });

  const assignMember = () => mutate(async () => {
    if (!task || !assigneeId) return;
    const assignee = await api.tasks.assign(task.id, { user_id: Number(assigneeId) });
    const next = { ...task, assignees: [...task.assignees.filter((item) => item.user_id !== assignee.user_id), assignee] };
    setTask(next);
    onChanged(next);
    setAssigneeId("");
  });

  const unassignMember = (userId: number) => mutate(async () => {
    if (!task) return;
    await api.tasks.unassign(task.id, userId);
    const next = { ...task, assignees: task.assignees.filter((item) => item.user_id !== userId) };
    setTask(next);
    onChanged(next);
  });

  const addChecklist = () => mutate(async () => {
    if (!task) return;
    const checklist = await api.checklists.create(task.id);
    setTask({ ...task, checklists: [...task.checklists, checklist] });
  });

  const addChecklistItem = (checklistId: number) => mutate(async () => {
    if (!task) return;
    const text = (newItems[checklistId] || "").trim();
    if (!text) return;
    const item = await api.checklists.createItem(checklistId, text);
    setTask({
      ...task,
      checklists: task.checklists.map((checklist) => checklist.id === checklistId ? { ...checklist, items: [...checklist.items, item] } : checklist),
    });
    setNewItems((current) => ({ ...current, [checklistId]: "" }));
  });

  const toggleItem = (checklistId: number, itemId: number, done: boolean) => mutate(async () => {
    if (!task) return;
    await api.checklists.updateItem(itemId, { is_done: done });
    setTask({
      ...task,
      checklists: task.checklists.map((checklist) => checklist.id === checklistId ? {
        ...checklist,
        items: checklist.items.map((item) => item.id === itemId ? { ...item, is_done: done } : item),
      } : checklist),
    });
  });

  const removeItem = (checklistId: number, itemId: number) => mutate(async () => {
    if (!task) return;
    await api.checklists.deleteItem(itemId);
    setTask({
      ...task,
      checklists: task.checklists.map((checklist) => checklist.id === checklistId ? { ...checklist, items: checklist.items.filter((item) => item.id !== itemId) } : checklist),
    });
  });

  const addComment = () => mutate(async () => {
    if (!task || !newComment.trim()) return;
    const comment = await api.comments.create(task.id, newComment.trim());
    setTask({ ...task, comments: [...task.comments, comment] });
    setNewComment("");
  });

  const removeComment = (commentId: number) => mutate(async () => {
    if (!task) return;
    await api.comments.delete(commentId);
    setTask({ ...task, comments: task.comments.filter((comment) => comment.id !== commentId) });
  });

  const removeTask = () => {
    if (!task || !window.confirm("Удалить задачу?")) return;
    void mutate(async () => {
      await api.tasks.delete(task.id);
      onDeleted();
    });
  };

  if (loading) {
    return <div className="modal-overlay" onClick={onClose}><div className="modal-content" onClick={(event) => event.stopPropagation()}><div className="loading-spinner" /></div></div>;
  }
  if (!task) {
    return <div className="modal-overlay" onClick={onClose}><div className="modal-content" onClick={(event) => event.stopPropagation()}><div className="modal-empty">{error || "Задача не найдена"}</div></div></div>;
  }

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-content modal-content--task" role="dialog" aria-modal="true" aria-label="Задача" onClick={(event) => event.stopPropagation()}>
        <div className="modal-header">
          <div className="modal-header-left"><span className="modal-task-id">#{task.id}</span><span className="modal-priority">{task.priority}</span></div>
          <div className="modal-header-actions">
            <button className="modal-delete" onClick={removeTask} disabled={saving} aria-label="Удалить задачу"><Trash2 size={17} /></button>
            <button className="modal-close" onClick={onClose} aria-label="Закрыть"><X size={20} /></button>
          </div>
        </div>
        <div className="modal-body">
          {error && <div className="form-error">{error}</div>}
          {editingTitle ? (
            <input className="input input--lg" value={title} onChange={(event) => setTitle(event.target.value)} onBlur={saveTitle} onKeyDown={(event) => event.key === "Enter" && saveTitle()} autoFocus />
          ) : <h2 className="modal-title" onClick={() => setEditingTitle(true)}>{task.title}</h2>}
          <div className="modal-props">
            <label className="modal-prop"><span className="modal-prop-label">Статус</span><select className="select" value={task.column_id} onChange={(event) => changeStatus(Number(event.target.value))}>{columns.map((column) => <option key={column.id} value={column.id}>{column.name}</option>)}</select></label>
            <label className="modal-prop"><span className="modal-prop-label">Приоритет</span><select className="select" value={task.priority} onChange={(event) => changePriority(event.target.value)}>{PRIORITY_OPTIONS.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}</select></label>
            <label className="modal-prop"><span className="modal-prop-label">Дедлайн</span><input type="date" className="input" value={task.deadline || ""} onChange={(event) => changeDeadline(event.target.value)} /></label>
          </div>
          <section className="modal-section">
            <div className="modal-section-header"><span className="modal-section-title">Исполнители ({task.assignees.length})</span></div>
            <div className="assignees">
              {task.assignees.map((assignee) => (
                <span className="assignee-chip" key={assignee.user_id}>
                  {assignee.display_name || assignee.username || `User ${assignee.user_id}`}
                  <button className="assignee-remove" onClick={() => unassignMember(assignee.user_id)} aria-label={`Снять исполнителя ${assignee.user_id}`}><X size={14} aria-hidden="true" /></button>
                </span>
              ))}
              {task.assignees.length === 0 && <div className="muted-text">Исполнители не назначены</div>}
            </div>
            <div className="form-row">
              <div className="form-group form-group--half">
                <label className="form-label" htmlFor="task-assignee">Добавить исполнителя</label>
                <select id="task-assignee" className="select" value={assigneeId} onChange={(event) => setAssigneeId(event.target.value)}>
                  <option value="">Выберите участника</option>
                  {members.filter((member) => !task.assignees.some((assignee) => assignee.user_id === member.user_id)).map((member) => (
                    <option key={member.user_id} value={member.user_id}>{member.display_name || member.username || `User ${member.user_id}`}</option>
                  ))}
                </select>
              </div>
              <div className="form-group form-group--half form-group--end">
                <button className="btn btn--primary btn--sm" onClick={assignMember} disabled={!assigneeId || saving}>Назначить</button>
              </div>
            </div>
          </section>
          <section className="modal-section">
            <div className="modal-section-header"><span className="modal-section-title">Описание</span>{!editingDescription && <button className="btn btn--ghost btn--sm" onClick={() => setEditingDescription(true)}>Изменить</button>}</div>
            {editingDescription ? <><textarea className="textarea" value={description} onChange={(event) => setDescription(event.target.value)} rows={4} /><div className="modal-section-actions"><button className="btn btn--secondary btn--sm" onClick={() => setEditingDescription(false)}>Отмена</button><button className="btn btn--primary btn--sm" onClick={saveDescription}>Сохранить</button></div></> : <div className="modal-desc" onClick={() => setEditingDescription(true)}>{task.description || "Добавить описание..."}</div>}
          </section>
          <section className="modal-section">
            <div className="modal-section-header"><span className="modal-section-title">Чеклисты</span><button className="btn btn--ghost btn--sm" onClick={addChecklist} disabled={saving}>+ Чеклист</button></div>
            {task.checklists.map((checklist) => {
              const done = checklist.items.filter((item) => item.is_done).length;
              return <div className="checklist" key={checklist.id}>
                <div className="checklist-header"><span className="checklist-title">{checklist.title}</span><span className="checklist-progress">{done}/{checklist.items.length}</span></div>
                {checklist.items.map((item) => <div className="checklist-item" key={item.id}><button className={`checklist-check ${item.is_done ? "checklist-check--done" : ""}`} onClick={() => toggleItem(checklist.id, item.id, !item.is_done)} aria-label="Отметить пункт">{item.is_done && <Check size={12} />}</button><span className={`checklist-text ${item.is_done ? "checklist-text--done" : ""}`}>{item.text}</span><button className="checklist-delete" onClick={() => removeItem(checklist.id, item.id)} aria-label="Удалить пункт"><Trash2 size={12} /></button></div>)}
                <input className="input input--sm" placeholder="Добавить пункт" value={newItems[checklist.id] || ""} onChange={(event) => setNewItems((current) => ({ ...current, [checklist.id]: event.target.value }))} onKeyDown={(event) => event.key === "Enter" && addChecklistItem(checklist.id)} />
              </div>;
            })}
            {task.checklists.length === 0 && <div className="muted-text">Чеклисты пока не добавлены</div>}
          </section>
          <section className="modal-section">
            <div className="modal-section-header"><span className="modal-section-title">Комментарии ({task.comments.length})</span></div>
            <div className="comments">{task.comments.map((comment) => <div className="comment" key={comment.id}><div className="comment-header"><span className="comment-author">{comment.display_name || comment.username || "Пользователь"}</span><span className="comment-date">{new Date(comment.created_at).toLocaleDateString("ru-RU")}</span>{(user?.user_id === comment.user_id) && <button className="comment-delete" onClick={() => removeComment(comment.id)} aria-label="Удалить комментарий"><Trash2 size={12} /></button>}</div><div className="comment-text">{comment.text}</div></div>)}</div>
            <input className="input" placeholder="Написать комментарий..." value={newComment} onChange={(event) => setNewComment(event.target.value)} onKeyDown={(event) => event.key === "Enter" && addComment()} />
          </section>
        </div>
      </div>
    </div>
  );
}
