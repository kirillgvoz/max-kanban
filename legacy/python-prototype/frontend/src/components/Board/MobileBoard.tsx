import { useState, useCallback } from "react";
import { Typography, Button, Counter } from "@maxhub/max-ui";
import { SwipeableCard } from "./SwipeableCard";
import { CreateTask } from "../Task/CreateTask";
import type { Board, Column, Task } from "../../types";

interface MobileBoardProps {
  board: Board;
  tasks: Task[];
  onMoveTask: (taskId: number, columnId: number, position: number) => Promise<void>;
  onCreateTask: (data: any, userId: number) => Promise<Task | undefined>;
  onDeleteTask: (taskId: number) => Promise<void>;
  currentUserId: number;
}

export function MobileBoard({
  board,
  tasks,
  onMoveTask,
  onCreateTask,
  onDeleteTask,
  currentUserId,
}: MobileBoardProps) {
  const columns = (board.columns || []).sort((a, b) => a.position - b.position);
  const [activeTab, setActiveTab] = useState(0);
  const [showCreate, setShowCreate] = useState(false);

  const getColumnTasks = useCallback(
    (columnId: number) =>
      tasks
        .filter((t) => t.column_id === columnId)
        .sort((a, b) => a.position - b.position),
    [tasks]
  );

  const currentColumn = columns[activeTab];
  const prevColumn = activeTab > 0 ? columns[activeTab - 1] : null;
  const nextColumn = activeTab < columns.length - 1 ? columns[activeTab + 1] : null;
  const currentTasks = currentColumn ? getColumnTasks(currentColumn.id) : [];

  const handleSwipeRight = (taskId: number) => {
    if (nextColumn) {
      const nextTasks = getColumnTasks(nextColumn.id);
      onMoveTask(taskId, nextColumn.id, nextTasks.length);
    }
  };

  const handleSwipeLeft = (taskId: number) => {
    if (prevColumn) {
      const prevTasks = getColumnTasks(prevColumn.id);
      onMoveTask(taskId, prevColumn.id, prevTasks.length);
    }
  };

  const handleCreateSubmit = async (data: any) => {
    if (currentColumn) {
      await onCreateTask({ ...data, column_id: currentColumn.id }, currentUserId);
    }
    setShowCreate(false);
  };

  return (
    <div className="mobile-board">
      <div className="mobile-tabs">
        {columns.map((col, i) => {
          const count = getColumnTasks(col.id).length;
          return (
            <button
              key={col.id}
              className={`mobile-tab ${i === activeTab ? "mobile-tab--active" : ""}`}
              onClick={() => setActiveTab(i)}
              style={
                i === activeTab
                  ? ({ "--tab-color": col.color } as React.CSSProperties)
                  : undefined
              }
            >
              <span className="mobile-tab-dot" style={{ backgroundColor: col.color }} />
              <span className="mobile-tab-label">{col.name}</span>
              {count > 0 && (
                <Counter value={count} appearance={i === activeTab ? "themed" : "neutral"} />
              )}
            </button>
          );
        })}
      </div>

      <div className="mobile-swipe-hint">
        {prevColumn && <span>← {prevColumn.name}</span>}
        <span className="mobile-swipe-hint-label">{currentColumn?.name}</span>
        {nextColumn && <span>{nextColumn.name} →</span>}
      </div>

      <div className="mobile-task-list">
        {currentTasks.length === 0 ? (
          <div className="mobile-empty">
            <Typography.Body>Нет задач</Typography.Body>
          </div>
        ) : (
          currentTasks.map((task) => (
            <SwipeableCard
              key={task.id}
              task={task}
              nextLabel={nextColumn?.name || ""}
              prevLabel={prevColumn?.name || ""}
              onSwipeRight={() => handleSwipeRight(task.id)}
              onSwipeLeft={() => handleSwipeLeft(task.id)}
              onDelete={() => onDeleteTask(task.id)}
            />
          ))
        )}
      </div>

      <button className="mobile-fab" onClick={() => setShowCreate(true)}>
        +
      </button>

      {showCreate && (
        <CreateTask
          columns={columns}
          initialColumnId={currentColumn?.id || null}
          onSubmit={handleCreateSubmit}
          onClose={() => setShowCreate(false)}
        />
      )}
    </div>
  );
}
