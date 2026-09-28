import { useSortable } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { Calendar, GripVertical } from "lucide-react";
import type { Task } from "../types";

interface KanbanCardProps {
  task: Task;
  onClick?: () => void;
  showDragHandle?: boolean;
}

interface CardViewProps extends KanbanCardProps {
  className?: string;
  style?: React.CSSProperties;
  dragAttributes?: any;
  dragListeners?: any;
  isDragging?: boolean;
  isOverlay?: boolean;
}

const PRIORITY_COLORS: Record<string, string> = {
  low: "#9CA3AF",
  medium: "#007AFF",
  high: "#FF9500",
  urgent: "#FF3B30",
};

function CardView({
  task,
  onClick,
  showDragHandle = true,
  className = "",
  style,
  dragAttributes,
  dragListeners,
  isOverlay = false,
}: CardViewProps) {
  return (
    <div
      style={style}
      className={`kanban-card ${className}`}
      onClick={onClick}
      role={onClick ? "button" : undefined}
      tabIndex={onClick ? 0 : undefined}
      onKeyDown={(event) => {
        if (onClick && (event.key === "Enter" || event.key === " ")) {
          event.preventDefault();
          onClick();
        }
      }}
    >
      {showDragHandle && (
        <div
          className="kanban-card-drag"
          {...dragAttributes}
          {...dragListeners}
          onClick={(event) => event.stopPropagation()}
        >
          <GripVertical size={14} aria-hidden="true" />
        </div>
      )}
      <div className="kanban-card-content">
        <div className="kanban-card-header">
          <span
            className="kanban-card-priority"
            style={{ backgroundColor: PRIORITY_COLORS[task.priority] }}
          />
          <span className="kanban-card-id">#{task.id}</span>
        </div>
        <div className="kanban-card-title">{task.title}</div>
        {task.description && (
          <div className="kanban-card-desc">
            {task.description.length > 80
              ? task.description.slice(0, 80) + "…"
              : task.description}
          </div>
        )}
        <div className="kanban-card-footer">
          {task.deadline && (
            <span className="kanban-card-meta">
              <Calendar size={12} />
              {task.deadline}
            </span>
          )}
          {task.assignees && task.assignees.length > 0 && (
            <div className="kanban-card-avatars">
              {task.assignees.slice(0, 3).map((assignee) => (
                <span key={assignee.user_id} className="avatar-sm">
                  {assignee.display_name?.[0] || "?"}
                </span>
              ))}
            </div>
          )}
        </div>
      </div>
      {isOverlay && <span className="sr-only">Перетаскиваемая карточка</span>}
    </div>
  );
}

export function KanbanCard({ task, onClick, showDragHandle = true }: KanbanCardProps) {
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id: `task-${task.id}` });

  return (
    <div ref={setNodeRef} className="kanban-card-sortable">
      <CardView
        task={task}
        onClick={onClick}
        showDragHandle={showDragHandle}
        dragAttributes={attributes}
        dragListeners={listeners}
        isDragging={isDragging}
        style={{
          transform: CSS.Transform.toString(transform),
          transition,
          opacity: isDragging ? 0.3 : 1,
        }}
      />
    </div>
  );
}

export function KanbanCardOverlay({ task }: { task: Task }) {
  return <CardView task={task} showDragHandle={false} isOverlay className="kanban-card--overlay" />;
}

export function KanbanCardStatic({ task, onClick }: KanbanCardProps) {
  return <CardView task={task} onClick={onClick} showDragHandle={false} />;
}
