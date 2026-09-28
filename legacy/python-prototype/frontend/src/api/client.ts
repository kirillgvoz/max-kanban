import type { Board, Task, AuthUser } from "../types";

const BASE = `${import.meta.env.BASE_URL}api`;

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    headers: { "Content-Type": "application/json", ...options?.headers },
    ...options,
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ detail: res.statusText }));
    throw new Error(err.detail || "Request failed");
  }
  return res.json();
}

export const api = {
  auth: {
    validate: (initData: string) =>
      request<AuthUser>("/auth/validate", {
        method: "POST",
        body: JSON.stringify({ initData }),
      }),
  },

  boards: {
    list: (userId?: number) =>
      request<Board[]>(`/boards${userId ? `?user_id=${userId}` : ""}`),
    get: (id: number) => request<Board>(`/boards/${id}`),
    create: (name: string, createdBy: number, chatId?: number) =>
      request<Board>(`/boards?created_by=${createdBy}`, {
        method: "POST",
        body: JSON.stringify({ name, chat_id: chatId }),
      }),
  },

  tasks: {
    listByBoard: (boardId: number) =>
      request<Task[]>(`/tasks/board/${boardId}`),
    get: (id: number) => request<Task>(`/tasks/${id}`),
    create: (boardId: number, data: {
      title: string;
      description?: string;
      column_id: number;
      deadline?: string;
      priority?: string;
      assignee_ids?: number[];
    }, createdBy: number) =>
      request<Task>(`/tasks/create-on-board?board_id=${boardId}&created_by=${createdBy}`, {
        method: "POST",
        body: JSON.stringify(data),
      }),
    update: (id: number, data: Partial<Task>) =>
      request<Task>(`/tasks/${id}`, {
        method: "PATCH",
        body: JSON.stringify(data),
      }),
    move: (id: number, columnId: number, position: number) =>
      request<Task>(`/tasks/${id}/move`, {
        method: "PUT",
        body: JSON.stringify({ column_id: columnId, position }),
      }),
    delete: (id: number) =>
      request<{ ok: boolean }>(`/tasks/${id}`, { method: "DELETE" }),
  },
};
