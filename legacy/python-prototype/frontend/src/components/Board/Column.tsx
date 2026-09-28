import { useDroppable } from "@dnd-kit/core";
import { Typography, Counter } from "@maxhub/max-ui";
import type { Column as ColumnType } from "../../types";

interface ColumnProps {
  column: ColumnType;
  taskCount: number;
  onAddTask: () => void;
  children: React.ReactNode;
}

export function Column({ column, taskCount, onAddTask, children }: ColumnProps) {
  const { setNodeRef, isOver } = useDroppable({ id: column.id });

  return (
    <div
      ref={setNodeRef}
      className={`column ${isOver ? "column--over" : ""}`}
      style={{ borderTopColor: column.color }}
    >
      <div className="column-header">
        <div className="column-header-left">
          <div className="column-dot" style={{ backgroundColor: column.color }} />
          <Typography.Label>{column.name}</Typography.Label>
        </div>
        <Counter value={taskCount} appearance="neutral" />
      </div>
      <div className="column-body">{children}</div>
      <button className="column-add-btn" onClick={onAddTask}>
        + Добавить
      </button>
    </div>
  );
}
