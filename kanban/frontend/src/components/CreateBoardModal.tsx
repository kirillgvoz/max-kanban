import { useEffect, useState } from "react";
import { X } from "lucide-react";

export interface BoardStatusDraft {
  name: string;
  color: string;
}

interface CreateBoardModalProps {
  onSubmit: (data: { name: string; description?: string; columns?: BoardStatusDraft[] }) => Promise<void> | void;
  onClose: () => void;
}

const DEFAULT_STATUSES: BoardStatusDraft[] = [
  { name: "К выполнению", color: "#6366F1" },
  { name: "В работе", color: "#F59E0B" },
  { name: "Готово", color: "#10B981" },
];

export function CreateBoardModal({ onSubmit, onClose }: CreateBoardModalProps) {
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [statuses, setStatuses] = useState<BoardStatusDraft[]>(DEFAULT_STATUSES);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => event.key === "Escape" && onClose();
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [onClose]);

  const validStatuses = statuses.map((status) => ({ ...status, name: status.name.trim() })).filter((status) => status.name !== "");

  const handleSubmit = async () => {
    if (!name.trim() || pending || validStatuses.length === 0 || validStatuses.length > 30) return;
    setPending(true);
    setError("");
    try {
      await onSubmit({ name: name.trim(), description: description.trim() || undefined, columns: validStatuses });
    } catch {
      setError("Не удалось создать доску");
    } finally {
      setPending(false);
    }
  };

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-content" role="dialog" aria-modal="true" aria-label="Новая доска" onClick={(event) => event.stopPropagation()}>
        <div className="modal-header"><h3 className="modal-header-title">Новая доска</h3><button className="modal-close" onClick={onClose} aria-label="Закрыть"><X size={20} /></button></div>
        <div className="modal-body">
          {error && <div className="form-error">{error}</div>}
          <div className="form-group"><label className="form-label" htmlFor="board-name">Название *</label><input id="board-name" className="input" value={name} onChange={(event) => setName(event.target.value)} autoFocus /></div>
          <div className="form-group"><label className="form-label" htmlFor="board-description">Описание</label><textarea id="board-description" className="textarea" value={description} onChange={(event) => setDescription(event.target.value)} rows={3} /></div>
          <div className="modal-section-header"><span className="modal-section-title">Статусы ({validStatuses.length})</span></div>
          {statuses.map((status, index) => (
            <div className="form-row" key={index}>
              <div className="form-group form-group--half"><label className="form-label" htmlFor={`status-name-${index}`}>Название</label><input id={`status-name-${index}`} className="input" value={status.name} onChange={(event) => setStatuses((previous) => previous.map((item, position) => position === index ? { ...item, name: event.target.value } : item))} /></div>
              <div className="form-group"><label className="form-label" htmlFor={`status-color-${index}`}>Цвет</label><input id={`status-color-${index}`} type="color" value={status.color} onChange={(event) => setStatuses((previous) => previous.map((item, position) => position === index ? { ...item, color: event.target.value } : item))} /></div>
              <div className="form-group form-group--end"><button className="btn btn--ghost" onClick={() => setStatuses((previous) => previous.filter((_, position) => position !== index))} disabled={statuses.length <= 1} aria-label={`Удалить статус ${index + 1}`}>Удалить</button></div>
            </div>
          ))}
          {statuses.length < 30 && <button className="btn btn--ghost" onClick={() => setStatuses((previous) => [...previous, { name: "", color: "#6B7280" }])}>+ Статус</button>}
        </div>
        <div className="modal-footer"><button className="btn btn--ghost" onClick={onClose}>Отмена</button><button className="btn btn--primary" onClick={() => void handleSubmit()} disabled={!name.trim() || pending || validStatuses.length === 0}>Создать</button></div>
      </div>
    </div>
  );
}
