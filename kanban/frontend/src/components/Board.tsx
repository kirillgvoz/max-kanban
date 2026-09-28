import { useCallback, useEffect, useMemo, useState } from "react";
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
import { useBoard } from "../hooks/useBoards";
import { useTasks } from "../hooks/useTasks";
import { filterTasks, upsertTask } from "../utils/tasks";
import { KanbanColumn } from "./KanbanColumn";
import { KanbanCard, KanbanCardOverlay } from "./KanbanCard";
import { MobileBoard } from "./MobileBoard";
import { BoardSettings } from "./BoardSettings";
import { TaskModal } from "./TaskModal";
import { CreateTaskModal } from "./CreateTaskModal";
import { Plus } from "lucide-react";
import type { AuthUser, Task } from "../types";

interface BoardProps {
  boardId: number;
  orgId: number;
  orgName: string;
  boardName: string;
  user: AuthUser | null;
  isMobile: boolean;
  subscribe: (type: string, handler: (data: any) => void) => () => void;
  onTaskUpdate: () => void;
}

export function Board({
  boardId,
  user,
  isMobile,
  subscribe,
}: BoardProps) {
  const { board, loading, refetch } = useBoard(boardId);
  const { tasks, loading: tasksLoading, createTask, moveTask, deleteTask, setTasks } = useTasks(boardId);
  const [activeTask, setActiveTask] = useState<Task | null>(null);
  const [selectedTaskId, setSelectedTaskId] = useState<number | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [showSettings, setShowSettings] = useState(false);
  const [query, setQuery] = useState("");
  const [priority, setPriority] = useState("");
  const [assignee, setAssignee] = useState("");
  const [overdueOnly, setOverdueOnly] = useState(false);

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 5 } }),
    useSensor(TouchSensor, { activationConstraint: { delay: 150, tolerance: 5 } })
  );

  useEffect(() => {
    const created = subscribe("task:created", (data: Task) => {
      if (data.board_id !== boardId) return;
      setTasks((previous) => upsertTask(previous, data));
    });
    const moved = subscribe("task:moved", (data: Task) => {
      if (data.board_id !== boardId) return;
      setTasks((previous) => previous.map((task) => task.id === data.id ? { ...task, ...data } : task));
    });
    const deleted = subscribe("task:deleted", (data: { id: number }) => {
      setTasks((previous) => previous.filter((task) => task.id !== data.id));
      setSelectedTaskId((current) => current === data.id ? null : current);
    });
    const updated = subscribe("task:updated", (data: Task) => {
      if (data.board_id !== boardId) return;
      setTasks((previous) => previous.map((task) => task.id === data.id ? { ...task, ...data } : task));
    });
    return () => {
      created();
      moved();
      deleted();
      updated();
    };
  }, [boardId, setTasks, subscribe]);

  const columns = (board?.columns || []).sort((a, b) => a.position - b.position);
  const visibleTasks = useMemo(() => filterTasks(tasks, {
    query,
    priority: priority || undefined,
    assigneeId: assignee ? Number(assignee) : null,
    overdueOnly,
  }), [tasks, query, priority, assignee, overdueOnly]);
  const getColumnTasks = useCallback(
    (columnId: number) => visibleTasks.filter((task) => task.column_id === columnId).sort((a, b) => a.position - b.position),
    [visibleTasks]
  );
  const assigneeOptions = useMemo(() => {
    const options = new Map<number, string>();
    for (const task of tasks) {
      for (const person of task.assignees || []) {
        options.set(person.user_id, person.display_name || person.username || `User ${person.user_id}`);
      }
    }
    return [...options.entries()];
  }, [tasks]);
  const canManage = Boolean(board?.members.some((member) => member.user_id === user?.user_id && (member.role === "owner" || member.role === "admin")));

  const handleDragStart = (event: DragStartEvent) => {
    const id = Number(String(event.active.id).replace("task-", ""));
    setActiveTask(tasks.find((task) => task.id === id) || null);
  };

  const handleDragEnd = (event: DragEndEvent) => {
    setActiveTask(null);
    const { active, over } = event;
    if (!over) return;
    const activeId = Number(String(active.id).replace("task-", ""));
    const overValue = String(over.id);
    const currentTask = tasks.find((task) => task.id === activeId);
    if (!currentTask || overValue === active.id) return;

    let targetColumnId: number;
    let targetPosition: number;
    if (overValue.startsWith("column-")) {
      targetColumnId = Number(overValue.replace("column-", ""));
      targetPosition = getColumnTasks(targetColumnId).filter((task) => task.id !== activeId).length;
    } else {
      const overTaskId = Number(overValue.replace("task-", ""));
      const overTask = tasks.find((task) => task.id === overTaskId);
      if (!overTask) return;
      targetColumnId = overTask.column_id;
      const siblings = getColumnTasks(targetColumnId).filter((task) => task.id !== activeId);
      const overIndex = siblings.findIndex((task) => task.id === overTaskId);
      targetPosition = overIndex >= 0 ? overIndex : siblings.length;
    }
    if (currentTask.column_id !== targetColumnId || currentTask.position !== targetPosition) {
      void moveTask(activeId, targetColumnId, targetPosition);
    }
  };

  const handleCreateTask = async (data: { title: string; description?: string; column_id?: number; priority?: string; deadline?: string | null }) => {
    await createTask(data);
    setShowCreate(false);
  };

  const handleArchived = async () => {
    setShowSettings(false);
    await refetch();
  };

  if (loading || tasksLoading) {
    return <div className="content-loading"><div className="loading-spinner" /></div>;
  }
  if (!board) return <div className="empty-state">Доска не найдена</div>;

  const filters = (
    <div className="board-filters">
      <input className="input input--sm" placeholder="Поиск задач" aria-label="Поиск задач" value={query} onChange={(event) => setQuery(event.target.value)} />
      <select className="select select--sm" value={priority} aria-label="Фильтр по приоритету" onChange={(event) => setPriority(event.target.value)}>
        <option value="">Все приоритеты</option>
        <option value="low">Низкий</option>
        <option value="medium">Средний</option>
        <option value="high">Высокий</option>
        <option value="urgent">Срочный</option>
      </select>
      <select className="select select--sm" value={assignee} aria-label="Фильтр по исполнителю" onChange={(event) => setAssignee(event.target.value)}>
        <option value="">Все исполнители</option>
        {assigneeOptions.map(([id, label]) => <option key={id} value={id}>{label}</option>)}
      </select>
      <label className="filter-checkbox"><input type="checkbox" checked={overdueOnly} onChange={(event) => setOverdueOnly(event.target.checked)} /> Просроченные</label>
    </div>
  );

  const content = isMobile ? (
    <>
      {filters}
      <MobileBoard
        board={board}
        tasks={visibleTasks}
        allTasks={tasks}
        columns={columns}
        onMoveTask={moveTask}
        onSelectTask={setSelectedTaskId}
        onCreateTask={() => setShowCreate(true)}
      />
    </>
  ) : (
    <div className="kanban">
      <div className="kanban-toolbar">
        {filters}
        <div className="content-actions">
          {canManage && <button className="btn btn--ghost btn--sm" onClick={() => setShowSettings(true)}>Настройки</button>}
          <button className="btn btn--primary btn--sm" onClick={() => setShowCreate(true)}>
            <Plus size={14} /> Задача
          </button>
        </div>
      </div>
      <DndContext
        sensors={sensors}
        collisionDetection={closestCorners}
        onDragStart={handleDragStart}
        onDragCancel={() => setActiveTask(null)}
        onDragEnd={handleDragEnd}
      >
        <div className="kanban-columns">
          {columns.map((column) => {
            const columnTasks = getColumnTasks(column.id);
            return (
              <KanbanColumn key={column.id} column={column} taskCount={columnTasks.length}>
                <SortableContext items={columnTasks.map((task) => `task-${task.id}`)} strategy={verticalListSortingStrategy}>
                  {columnTasks.map((task) => (
                    <KanbanCard key={task.id} task={task} onClick={() => setSelectedTaskId(task.id)} />
                  ))}
                </SortableContext>
              </KanbanColumn>
            );
          })}
        </div>
        <DragOverlay>
          {activeTask ? <div className="drag-overlay"><KanbanCardOverlay task={activeTask} /></div> : null}
        </DragOverlay>
      </DndContext>
    </div>
  );

  return (
    <>
      {content}
      {selectedTaskId && (
        <TaskModal
          taskId={selectedTaskId}
          columns={columns}
          members={board.members}
          user={user}
          onClose={() => setSelectedTaskId(null)}
          onDeleted={() => {
            if (selectedTaskId) void deleteTask(selectedTaskId);
            setSelectedTaskId(null);
          }}
          onChanged={(task) => setTasks((previous) => previous.map((item) => item.id === task.id ? { ...item, ...task } : item))}
        />
      )}
      {showSettings && (
        <BoardSettings
          board={board}
          tasks={tasks}
          user={user}
          canManage={canManage}
          onChanged={() => void refetch()}
          onArchived={() => void handleArchived()}
          onClose={() => setShowSettings(false)}
        />
      )}
      {showCreate && (
        <CreateTaskModal columns={columns} onSubmit={handleCreateTask} onClose={() => setShowCreate(false)} />
      )}
    </>
  );
}
