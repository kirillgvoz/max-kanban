import { useSortable } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { Typography, Avatar } from "@maxhub/max-ui";
import type { Task } from "../../types";

interface TaskCardProps {
  task: Task;
  onDelete?: () => void;
  isDragOverlay?: boolean;
}

const PRIORITY_COLORS: Record<string, string> = {
  low: "#6B7280",
  medium: "#3B82F6",
  high: "#F59E0B",
  urgent: "#EF4444",
};

const PRIORITY_LABELS: Record<string, string> = {
  low: "Низкий",
  medium: "Средний",
  high: "Высокий",
  urgent: "Срочно",
};

export function TaskCard({ task, onDelete, isDragOverlay }: TaskCardProps) {
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id: task.id });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.3 : 1,
  };

  const assignee = task.assignees?.[0];
  const initials = assignee?.display_name
    ? assignee.display_name
        .split(" ")
        .map((n) => n[0])
        .join("")
        .slice(0, 2)
        .toUpperCase()
    : "?";

  return (
    <div
      ref={setNodeRef}
      style={style}
      className={`task-card ${isDragOverlay ? "task-card--overlay" : ""}`}
    >
      <div className="task-card-row">
        <div
          className="task-drag-handle"
          {...attributes}
          {...listeners}
        >
          ⠿
        </div>
        <div className="task-card-body">
          <div className="task-card-header">
            <div
              className="task-priority-dot"
              style={{ backgroundColor: PRIORITY_COLORS[task.priority] }}
              title={PRIORITY_LABELS[task.priority]}
            />
            <Typography.Label>#{task.id}</Typography.Label>
            {onDelete && (
              <button
                className="task-delete-btn"
                onClick={(e) => {
                  e.stopPropagation();
                  onDelete();
                }}
                onPointerDown={(e) => e.stopPropagation()}
              >
                ×
              </button>
            )}
          </div>
          <Typography.Body className="task-title">{task.title}</Typography.Body>
          {task.description && (
            <Typography.Body className="task-description">
              {task.description.length > 80
                ? task.description.slice(0, 80) + "…"
                : task.description}
            </Typography.Body>
          )}
          <div className="task-card-footer">
            {task.deadline && (
              <span className="task-deadline">📅 {task.deadline}</span>
            )}
            {assignee && (
              <Avatar.Container size={24} form="circle">
                <Avatar.Text>{initials}</Avatar.Text>
              </Avatar.Container>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
