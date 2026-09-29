import { useRef, useState } from "react";
import { ChevronLeft, ChevronRight, ClipboardList, Plus, Settings } from "lucide-react";
import type { Board, Column, Task } from "../types";
import { KanbanCardStatic } from "./KanbanCard";

interface MobileBoardProps {
  board: Board;
  tasks: Task[];
  allTasks: Task[];
  columns: Column[];
  onMoveTask: (taskId: number, columnId: number, position: number) => Promise<void>;
  onSelectTask: (id: number) => void;
  onCreateTask: () => void;
  canManage?: boolean;
  onOpenSettings?: () => void;
}

const SWIPE_THRESHOLD = 60;

export function MobileBoard({ board, tasks, allTasks, columns, onMoveTask, onSelectTask, onCreateTask, canManage, onOpenSettings }: MobileBoardProps) {
  const [activeTab, setActiveTab] = useState(0);
  const currentColumn = columns[activeTab] || columns[0];
  const currentTasks = currentColumn
    ? tasks.filter((task) => task.column_id === currentColumn.id).sort((a, b) => a.position - b.position)
    : [];
  const previousColumn = columns[activeTab - 1];
  const nextColumn = columns[activeTab + 1];

  const moveToAdjacent = (task: Task, direction: 1 | -1) => {
    const target = direction === 1 ? nextColumn : previousColumn;
    if (!target) return;
    const position = allTasks.filter((item) => item.column_id === target.id).length;
    void onMoveTask(task.id, target.id, position);
  };

  return (
    <div className="mobile-board">
      <div className="mobile-board-bar">
        <span className="mobile-board-id">ID #{board.id}</span>
        {canManage && onOpenSettings && (
          <button className="mobile-settings" onClick={onOpenSettings} aria-label="Настройки доски">
            <Settings size={18} aria-hidden="true" />
          </button>
        )}
      </div>
      <div className="mobile-tabs" role="tablist" aria-label="Статусы">
        {columns.map((column, index) => {
          const count = tasks.filter((task) => task.column_id === column.id).length;
          return (
            <button
              key={column.id}
              role="tab"
              aria-selected={index === activeTab}
              className={`mobile-tab ${index === activeTab ? "mobile-tab--active" : ""}`}
              onClick={() => setActiveTab(index)}
              style={index === activeTab ? { borderBottomColor: column.color } : undefined}
            >
              <span className="mobile-tab-dot" style={{ backgroundColor: column.color }} />
              <span className="mobile-tab-label">{column.name}</span>
              <span className="mobile-tab-count">{count}</span>
            </button>
          );
        })}
      </div>
      <div className="mobile-swipe-hint">
        {previousColumn ? <span><ChevronLeft size={14} /> {previousColumn.name}</span> : <span />}
        <strong>{currentColumn?.name}</strong>
        {nextColumn ? <span>{nextColumn.name} <ChevronRight size={14} /></span> : <span />}
      </div>
      <div className="mobile-task-list">
        {currentTasks.length === 0 ? (
          <div className="mobile-empty"><ClipboardList size={48} className="mobile-empty-icon" aria-hidden="true" /><div className="mobile-empty-text">Нет задач</div></div>
        ) : currentTasks.map((task) => (
          <SwipeableTaskCard
            key={task.id}
            task={task}
            previousLabel={previousColumn?.name}
            nextLabel={nextColumn?.name}
            onClick={() => onSelectTask(task.id)}
            onSwipePrevious={() => moveToAdjacent(task, -1)}
            onSwipeNext={() => moveToAdjacent(task, 1)}
          />
        ))}
      </div>
      <button className="mobile-fab" onClick={onCreateTask} aria-label="Новая задача"><Plus size={24} /></button>
    </div>
  );
}

interface SwipeableTaskCardProps {
  task: Task;
  previousLabel?: string;
  nextLabel?: string;
  onClick: () => void;
  onSwipePrevious: () => void;
  onSwipeNext: () => void;
}

function SwipeableTaskCard({ task, previousLabel, nextLabel, onClick, onSwipePrevious, onSwipeNext }: SwipeableTaskCardProps) {
  const [offset, setOffset] = useState(0);
  const [animating, setAnimating] = useState(false);
  const start = useRef({ x: 0, y: 0 });
  const current = useRef({ x: 0, y: 0 });
  const horizontal = useRef(false);

  const finish = (action: "previous" | "next" | null) => {
    setAnimating(true);
    setOffset(action === "next" ? 120 : action === "previous" ? -120 : 0);
    window.setTimeout(() => {
      if (action === "next") onSwipeNext();
      if (action === "previous") onSwipePrevious();
      setOffset(0);
      setAnimating(false);
    }, 160);
  };

  return (
    <div className={`swipeable-task ${animating ? "swipeable-task--animating" : ""}`}>
      <div className={`swipe-task-hint swipe-task-hint--previous ${offset < -20 ? "swipe-task-hint--visible" : ""}`}>
        {previousLabel || "Назад"}
      </div>
      <div className={`swipe-task-hint swipe-task-hint--next ${offset > 20 ? "swipe-task-hint--visible" : ""}`}>
        {nextLabel || "Вперёд"}
      </div>
      <div
        className="swipe-task-card"
        style={{ transform: `translateX(${offset}px)`, transition: animating ? "transform .16s ease" : "none" }}
        onClick={onClick}
        onTouchStart={(event) => {
          start.current = { x: event.touches[0].clientX, y: event.touches[0].clientY };
          current.current = { ...start.current };
          horizontal.current = false;
        }}
        onTouchMove={(event) => {
          const dx = event.touches[0].clientX - start.current.x;
          const dy = event.touches[0].clientY - start.current.y;
          if (!horizontal.current && Math.abs(dx) > 8) {
            if (Math.abs(dx) > Math.abs(dy)) horizontal.current = true;
            else return;
          }
          if (horizontal.current) {
            current.current = { x: event.touches[0].clientX, y: event.touches[0].clientY };
            setOffset(dx);
          }
        }}
        onTouchEnd={() => {
          if (!horizontal.current) return;
          const delta = current.current.x - start.current.x;
          finish(delta >= SWIPE_THRESHOLD ? "next" : delta <= -SWIPE_THRESHOLD ? "previous" : null);
          horizontal.current = false;
        }}
        onTouchCancel={() => {
          horizontal.current = false;
          setOffset(0);
        }}
      >
        <KanbanCardStatic task={task} />
      </div>
    </div>
  );
}
