import { useState, useEffect, useCallback } from "react";
import { api } from "../api/client";
import { sanitizeTasks, upsertTask } from "../utils/tasks";
import type { Task } from "../types";

export function useTasks(boardId: number | null) {
  const [tasks, setTasks] = useState<Task[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchTasks = useCallback(async () => {
    if (!boardId) return;
    try {
      const data = await api.tasks.listByBoard(boardId);
      setTasks(sanitizeTasks(data));
    } catch (e) {
      console.error("Failed to fetch tasks:", e);
    } finally {
      setLoading(false);
    }
  }, [boardId]);

  useEffect(() => {
    fetchTasks();
  }, [fetchTasks]);

  const createTask = useCallback(async (data: { title: string; description?: string; column_id?: number; priority?: string; deadline?: string | null }) => {
    if (!boardId) return;
    const task = await api.tasks.create(boardId, data);
    setTasks((prev) => upsertTask(prev, task));
    return task;
  }, [boardId]);

  const moveTask = useCallback(async (taskId: number, columnId: number, position: number) => {
    setTasks((prev) =>
      prev.map((t) =>
        t.id === taskId ? { ...t, column_id: columnId, position } : t
      )
    );

    try {
      await api.tasks.move(taskId, { column_id: columnId, position });
    } catch (e) {
      console.error("Move failed, refetching:", e);
      fetchTasks();
    }
  }, [fetchTasks]);

  const deleteTask = useCallback(async (taskId: number) => {
    setTasks((prev) => prev.filter((t) => t.id !== taskId));
    try {
      await api.tasks.delete(taskId);
    } catch (e) {
      fetchTasks();
    }
  }, [fetchTasks]);

  const updateTask = useCallback(async (taskId: number, data: { title?: string; description?: string; priority?: string; deadline?: string }) => {
    setTasks((prev) =>
      prev.map((t) => (t.id === taskId ? { ...t, ...data, priority: (data.priority || t.priority) as Task["priority"] } : t))
    );
    try {
      await api.tasks.update(taskId, data);
    } catch (e) {
      fetchTasks();
    }
  }, [fetchTasks]);

  return { tasks, loading, createTask, moveTask, deleteTask, updateTask, refetch: fetchTasks, setTasks };
}
