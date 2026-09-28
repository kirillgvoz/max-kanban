import type { Task } from "../types";

export function sanitizeTasks(tasks: Task[]): Task[] {
  const seen = new Map<number, Task>();
  for (const task of tasks) {
    seen.set(task.id, task);
  }
  const sanitized: Task[] = [];
  for (const task of tasks) {
    const merged = seen.get(task.id);
    if (merged && !sanitized.some((existing) => existing.id === task.id)) {
      sanitized.push(merged);
    }
  }
  return sanitized;
}

export interface TaskFilters {
  query?: string;
  priority?: string;
  assigneeId?: number | null;
  overdueOnly?: boolean;
  today?: string;
}

export function filterTasks(tasks: Task[], filters: TaskFilters = {}): Task[] {
  const query = (filters.query || "").trim().toLowerCase();
  const today = filters.today || new Date().toISOString().slice(0, 10);
  return tasks.filter((task) => {
    if (query && !`${task.title} ${task.description || ""}`.toLowerCase().includes(query)) return false;
    if (filters.priority && task.priority !== filters.priority) return false;
    if (filters.assigneeId && !(task.assignees || []).some((assignee) => assignee.user_id === filters.assigneeId)) return false;
    if (filters.overdueOnly && (!task.deadline || task.deadline >= today)) return false;
    return true;
  });
}

export function upsertTask(tasks: Task[], task: Task): Task[] {
  if (tasks.some((existing) => existing.id === task.id)) {
    return tasks.map((existing) =>
      existing.id === task.id ? { ...existing, ...task } : existing
    );
  }
  return [...tasks, task];
}
