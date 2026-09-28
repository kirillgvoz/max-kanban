import { useEffect, useRef, useState } from "react";
import { X } from "lucide-react";
import type { Column } from "../types";

interface CreateTaskModalProps {
  columns: Column[];
  onSubmit: (data: {
    title: string;
    description?: string;
    column_id?: number;
    priority?: string;
    deadline?: string | null;
  }) => Promise<void> | void;
  onClose: () => void;
}

const PRIORITY_OPTIONS = [
  { value: "low", label: "Низкий" },
  { value: "medium", label: "Средний" },
  { value: "high", label: "Высокий" },
  { value: "urgent", label: "Срочный" },
];

export function CreateTaskModal({ columns, onSubmit, onClose }: CreateTaskModalProps) {
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [columnId, setColumnId] = useState(columns[0]?.id || 0);
  const [priority, setPriority] = useState("medium");
  const [deadline, setDeadline] = useState("");
  const [pending, setPending] = useState(false);
  const submittingRef = useRef(false);
  const [error, setError] = useState("");

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => event.key === "Escape" && onClose();
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [onClose]);

  const handleSubmit = async () => {
    if (!title.trim() || submittingRef.current) return;
    if (columns.length === 0) {
      setError("Сначала добавьте статус");
      return;
    }
    submittingRef.current = true;
    setPending(true);
    setError("");
    try {
      await onSubmit({
        title: title.trim(),
        description: description.trim() || undefined,
        column_id: columnId,
        priority,
        deadline: deadline || null,
      });
    } catch {
      setError("Не удалось создать задачу");
    } finally {
      submittingRef.current = false;
      setPending(false);
    }
  };

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-content" role="dialog" aria-modal="true" aria-label="Новая задача" onClick={(event) => event.stopPropagation()}>
        <div className="modal-header"><h3 className="modal-header-title">Новая задача</h3><button className="modal-close" onClick={onClose} aria-label="Закрыть"><X size={20} /></button></div>
        <div className="modal-body">
          {error && <div className="form-error">{error}</div>}
          <div className="form-group"><label className="form-label" htmlFor="new-task-title">Заголовок *</label><input id="new-task-title" className="input" placeholder="Что нужно сделать?" value={title} onChange={(event) => setTitle(event.target.value)} onKeyDown={(event) => event.key === "Enter" && void handleSubmit()} autoFocus /></div>
          <div className="form-group"><label className="form-label" htmlFor="new-task-description">Описание</label><textarea id="new-task-description" className="textarea" placeholder="Детали задачи..." value={description} onChange={(event) => setDescription(event.target.value)} rows={3} /></div>
          <div className="form-row">
            <div className="form-group form-group--half"><label className="form-label" htmlFor="new-task-status">Статус</label><select id="new-task-status" className="select" value={columnId} onChange={(event) => setColumnId(Number(event.target.value))}>{columns.map((column) => <option key={column.id} value={column.id}>{column.name}</option>)}</select></div>
            <div className="form-group form-group--half"><label className="form-label" htmlFor="new-task-priority">Приоритет</label><select id="new-task-priority" className="select" value={priority} onChange={(event) => setPriority(event.target.value)}>{PRIORITY_OPTIONS.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}</select></div>
          </div>
          <div className="form-group"><label className="form-label" htmlFor="new-task-deadline">Дедлайн</label><input id="new-task-deadline" type="date" className="input" value={deadline} onChange={(event) => setDeadline(event.target.value)} /></div>
        </div>
        <div className="modal-footer"><button className="btn btn--ghost" onClick={onClose}>Отмена</button><button className="btn btn--primary" onClick={() => void handleSubmit()} disabled={!title.trim() || pending}>{pending ? "Создание…" : "Создать"}</button></div>
      </div>
    </div>
  );
}
