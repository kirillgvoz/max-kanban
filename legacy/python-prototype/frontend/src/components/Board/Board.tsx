import { useState, useCallback } from "react";
import {
  DndContext,
  DragOverlay,
  closestCorners,
  PointerSensor,
  TouchSensor,
  useSensor,
  useSensors,
  type DragStartEvent,
  type DragEndEvent,
} from "@dnd-kit/core";
import { SortableContext, verticalListSortingStrategy } from "@dnd-kit/sortable";
import { Column } from "./Column";
import { TaskCard } from "./TaskCard";
import { MobileBoard } from "./MobileBoard";
import { CreateTask } from "../Task/CreateTask";
import { useIsMobile } from "../../hooks/useIsMobile";
import type { Board, Task } from "../../types";
import { Button, Typography } from "@maxhub/max-ui";

interface BoardViewProps {
  board: Board;
  tasks: Task[];
  onMoveTask: (taskId: number, columnId: number, position: number) => Promise<void>;
  onCreateTask: (data: any, userId: number) => Promise<Task | undefined>;
  onDeleteTask: (taskId: number) => Promise<void>;
  currentUserId: number;
}

export function BoardView({
  board,
  tasks,
  onMoveTask,
  onCreateTask,
  onDeleteTask,
  currentUserId,
}: BoardViewProps) {
  const isMobile = useIsMobile();

  if (isMobile) {
    return (
      <MobileBoard
        board={board}
        tasks={tasks}
        onMoveTask={onMoveTask}
        onCreateTask={onCreateTask}
        onDeleteTask={onDeleteTask}
        currentUserId={currentUserId}
      />
    );
  }

  return (
    <DesktopBoard
      board={board}
      tasks={tasks}
      onMoveTask={onMoveTask}
      onCreateTask={onCreateTask}
      onDeleteTask={onDeleteTask}
      currentUserId={currentUserId}
    />
  );
}

function DesktopBoard({
  board,
  tasks,
  onMoveTask,
  onCreateTask,
  onDeleteTask,
  currentUserId,
}: BoardViewProps) {
  const [activeTask, setActiveTask] = useState<Task | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [createColumnId, setCreateColumnId] = useState<number | null>(null);

  const columns = board.columns || [];

  const sensors = useSensors(
    useSensor(PointerSensor, {
      activationConstraint: { distance: 5 },
    }),
    useSensor(TouchSensor, {
      activationConstraint: { delay: 150, tolerance: 5 },
    })
  );

  const getColumnTasks = useCallback(
    (columnId: number) =>
      tasks
        .filter((t) => t.column_id === columnId)
        .sort((a, b) => a.position - b.position),
    [tasks]
  );

  const handleDragStart = (event: DragStartEvent) => {
    const task = tasks.find((t) => t.id === Number(event.active.id));
    if (task) setActiveTask(task);
  };

  const handleDragEnd = (event: DragEndEvent) => {
    setActiveTask(null);
    const { active, over } = event;
    if (!over) return;

    const activeId = Number(active.id);
    const overId = Number(over.id);

    const activeTask = tasks.find((t) => t.id === activeId);
    if (!activeTask) return;

    let targetColumnId: number;
    let targetPosition: number;

    const overTask = tasks.find((t) => t.id === overId);
    if (overTask) {
      targetColumnId = overTask.column_id;
      const colTasks = getColumnTasks(targetColumnId).filter((t) => t.id !== activeId);
      const overIndex = colTasks.findIndex((t) => t.id === overId);
      targetPosition = overIndex >= 0 ? overIndex : colTasks.length;
    } else {
      targetColumnId = overId;
      targetPosition = getColumnTasks(targetColumnId).filter((t) => t.id !== activeId).length;
    }

    if (activeTask.column_id !== targetColumnId || activeTask.position !== targetPosition) {
      onMoveTask(activeId, targetColumnId, targetPosition);
    }
  };

  const handleCreate = (columnId: number) => {
    setCreateColumnId(columnId);
    setShowCreate(true);
  };

  const handleCreateSubmit = async (data: any) => {
    if (createColumnId !== null) {
      await onCreateTask({ ...data, column_id: createColumnId }, currentUserId);
    }
    setShowCreate(false);
    setCreateColumnId(null);
  };

  return (
    <div className="board-container">
      <div className="board-toolbar">
        <Typography.Title variant="large-strong">{board.name}</Typography.Title>
        <Button variant="primary" onClick={() => handleCreate(columns[0]?.id)}>
          + Задача
        </Button>
      </div>

      <DndContext
        sensors={sensors}
        collisionDetection={closestCorners}
        onDragStart={handleDragStart}
        onDragEnd={handleDragEnd}
      >
        <div className="board-columns">
          {columns.map((col) => {
            const colTasks = getColumnTasks(col.id);
            return (
              <Column
                key={col.id}
                column={col}
                taskCount={colTasks.length}
                onAddTask={() => handleCreate(col.id)}
              >
                <SortableContext
                  items={colTasks.map((t) => t.id)}
                  strategy={verticalListSortingStrategy}
                >
                  {colTasks.map((task) => (
                    <TaskCard
                      key={task.id}
                      task={task}
                      onDelete={() => onDeleteTask(task.id)}
                    />
                  ))}
                </SortableContext>
              </Column>
            );
          })}
        </div>

        <DragOverlay>
          {activeTask ? (
            <div className="drag-overlay">
              <TaskCard task={activeTask} isDragOverlay />
            </div>
          ) : null}
        </DragOverlay>
      </DndContext>

      {showCreate && (
        <CreateTask
          columns={columns}
          initialColumnId={createColumnId}
          onSubmit={handleCreateSubmit}
          onClose={() => {
            setShowCreate(false);
            setCreateColumnId(null);
          }}
        />
      )}
    </div>
  );
}
