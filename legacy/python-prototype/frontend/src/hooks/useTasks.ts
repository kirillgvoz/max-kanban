import { useState, useEffect, useCallback } from "react";
import { api } from "../api/client";
import type { Task } from "../types";

export function useTasks(boardId: number | null) {
  const [tasks, setTasks] = useState<Task[]>([]);
  const [loading, setLoading] = useState(false);

  const fetchTasks = useCallback(async () => {
    if (!boardId) return;
    setLoading(true);
    try {
      const data = await api.tasks.listByBoard(boardId);
      setTasks(data);
    } catch {
    } finally {
      setLoading(false);
    }
  }, [boardId]);

  useEffect(() => {
    fetchTasks();
  }, [fetchTasks]);

  const moveTask = useCallback(
    async (taskId: number, newColumnId: number, newPosition: number) => {
      setTasks((prev) =>
        prev.map((t) =>
          t.id === taskId ? { ...t, column_id: newColumnId, position: newPosition } : t
        )
      );
      try {
        await api.tasks.move(taskId, newColumnId, newPosition);
      } catch {
        fetchTasks();
      }
    },
    [fetchTasks]
  );

  const createTask = useCallback(
    async (data: {
      title: string;
      description?: string;
      column_id: number;
      deadline?: string;
      priority?: string;
      assignee_ids?: number[];
    }, createdBy: number) => {
      if (!boardId) return;
      const task = await api.tasks.create(boardId, data, createdBy);
      setTasks((prev) => [...prev, task]);
      return task;
    },
    [boardId]
  );

  const deleteTask = useCallback(
    async (taskId: number) => {
      await api.tasks.delete(taskId);
      setTasks((prev) => prev.filter((t) => t.id !== taskId));
    },
    []
  );

  return { tasks, loading, moveTask, createTask, deleteTask, refetch: fetchTasks };
}
