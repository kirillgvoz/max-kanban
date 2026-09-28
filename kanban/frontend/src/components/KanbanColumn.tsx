import { useDroppable } from "@dnd-kit/core";
import type { ReactNode } from "react";
import type { Column } from "../types";

interface KanbanColumnProps {
  column: Column;
  taskCount: number;
  children: ReactNode;
}

export function KanbanColumn({ column, taskCount, children }: KanbanColumnProps) {
  const { setNodeRef, isOver } = useDroppable({ id: `column-${column.id}` });

  return (
    <div
      ref={setNodeRef}
      className={`kanban-col ${isOver ? "kanban-col--over" : ""}`}
    >
      <div className="kanban-col-header">
        <div className="kanban-col-title">
          <span className="kanban-col-dot" style={{ backgroundColor: column.color }} />
          {column.name}
        </div>
        <span className="kanban-col-count">{taskCount}</span>
      </div>
      <div className="kanban-col-body">{children}</div>
    </div>
  );
}
