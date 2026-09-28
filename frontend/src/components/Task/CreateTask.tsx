import { useState } from "react";
import { Typography, Button, Input, Textarea } from "@maxhub/max-ui";
import type { Column } from "../../types";

interface CreateTaskProps {
  columns: Column[];
  initialColumnId: number | null;
  onSubmit: (data: {
    title: string;
    description?: string;
    priority: string;
    deadline?: string;
  }) => void;
  onClose: () => void;
}

export function CreateTask({ columns, initialColumnId, onSubmit, onClose }: CreateTaskProps) {
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [priority, setPriority] = useState("medium");
  const [deadline, setDeadline] = useState("");

  const handleSubmit = () => {
    if (!title.trim()) return;
    onSubmit({
      title: title.trim(),
      description: description.trim() || undefined,
      priority,
      deadline: deadline || undefined,
    });
  };

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-content" onClick={(e) => e.stopPropagation()}>
        <Typography.Title variant="medium-strong">Новая задача</Typography.Title>

        <div className="form-group">
          <Typography.Label>Название *</Typography.Label>
          <Input
            value={title}
            onChange={(e: React.ChangeEvent<HTMLInputElement>) => setTitle(e.target.value)}
            placeholder="Что нужно сделать?"
          />
        </div>

        <div className="form-group">
          <Typography.Label>Описание</Typography.Label>
          <Textarea
            value={description}
            onChange={(e: React.ChangeEvent<HTMLTextAreaElement>) => setDescription(e.target.value)}
            placeholder="Подробности задачи..."
            rows={3}
          />
        </div>

        <div className="form-group">
          <Typography.Label>Приоритет</Typography.Label>
          <div className="priority-selector">
            {[
              { value: "low", label: "Низкий", color: "#6B7280" },
              { value: "medium", label: "Средний", color: "#3B82F6" },
              { value: "high", label: "Высокий", color: "#F59E0B" },
              { value: "urgent", label: "Срочно", color: "#EF4444" },
            ].map((p) => (
              <button
                key={p.value}
                className={`priority-btn ${priority === p.value ? "priority-btn--active" : ""}`}
                style={{ borderColor: priority === p.value ? p.color : "transparent" }}
                onClick={() => setPriority(p.value)}
              >
                <span className="priority-dot" style={{ backgroundColor: p.color }} />
                {p.label}
              </button>
            ))}
          </div>
        </div>

        <div className="form-group">
          <Typography.Label>Дедлайн</Typography.Label>
          <Input
            type="date"
            value={deadline}
            onChange={(e: React.ChangeEvent<HTMLInputElement>) => setDeadline(e.target.value)}
          />
        </div>

        <div className="modal-actions">
          <Button variant="secondary" onClick={onClose}>
            Отмена
          </Button>
          <Button variant="primary" onClick={handleSubmit} disabled={!title.trim()}>
            Создать
          </Button>
        </div>
      </div>
    </div>
  );
}
