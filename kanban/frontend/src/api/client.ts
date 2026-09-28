import type {
  Organization,
  OrgDetail,
  Assignee,
  Board,
  BoardChat,
  BoardDetail,
  Column,
  Task,
  TaskDetail,
  Comment,
  Checklist,
  ChecklistItem,
  AuthUser,
} from "../types";

const BASE = `${import.meta.env.BASE_URL}api`;

export class ApiError extends Error {
  status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const initData = window.WebApp?.initData || "";
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...(options?.headers as Record<string, string>),
  };
  if (initData) {
    headers["X-Max-InitData"] = initData;
  }

  const controller = new AbortController();
  const timeout = window.setTimeout(() => controller.abort(), 15000);
  try {
    const res = await fetch(`${BASE}${path}`, {
      ...options,
      headers,
      signal: controller.signal,
    });
    const body = await res.json().catch(() => ({}));
    if (!res.ok) {
      throw new ApiError(res.status, body.error || `HTTP ${res.status}`);
    }
    return body as T;
  } finally {
    window.clearTimeout(timeout);
  }
}

function qs(params: Record<string, string | number | undefined>): string {
  const entries = Object.entries(params).filter(
    ([, v]) => v !== undefined && v !== ""
  );
  return entries.length ? "?" + new URLSearchParams(entries.map(([k, v]) => [k, String(v)])).toString() : "";
}

export const api = {
  auth: {
    validate: (initData: string) =>
      request<{ ok: boolean; user: AuthUser; error?: string }>(
        "/auth/validate",
        {
          method: "POST",
          body: JSON.stringify({ initData }),
        }
      ),
  },

  ws: {
    ticket: (orgId: number) =>
      request<{ ticket: string }>("/ws-ticket", {
        method: "POST",
        body: JSON.stringify({ org_id: orgId }),
      }),
  },

  orgs: {
    list: () => request<Organization[]>("/orgs"),
    create: (name: string) =>
      request<Organization>("/orgs", {
        method: "POST",
        body: JSON.stringify({ name }),
      }),
    get: (id: number) => request<OrgDetail>(`/orgs/${id}`),
    update: (id: number, data: { name?: string; avatar_url?: string }) =>
      request<{ ok: boolean }>(`/orgs/${id}`, {
        method: "PATCH",
        body: JSON.stringify(data),
      }),
    addMember: (
      orgId: number,
      data: { user_id: number; username?: string; display_name?: string; role?: string }
    ) =>
      request<{ ok: boolean }>(`/orgs/${orgId}/members`, {
        method: "POST",
        body: JSON.stringify(data),
      }),
    removeMember: (orgId: number, userId: number) =>
      request<{ ok: boolean }>(`/orgs/${orgId}/members/${userId}`, {
        method: "DELETE",
      }),
  },

  boards: {
    listByOrg: (orgId: number, includeArchived = false) =>
      request<Board[]>(`/orgs/${orgId}/boards${includeArchived ? "?include_archived=true" : ""}`),
    create: (orgId: number, data: { name: string; description?: string; columns?: Array<{ name: string; color?: string }> }) =>
      request<Board>(`/orgs/${orgId}/boards`, {
        method: "POST",
        body: JSON.stringify(data),
      }),
    get: (id: number) => request<BoardDetail>(`/boards/${id}`),
    update: (id: number, data: { name?: string; description?: string; is_archived?: boolean }) =>
      request<{ ok: boolean }>(`/boards/${id}`, {
        method: "PATCH",
        body: JSON.stringify(data),
      }),
    delete: (id: number) =>
      request<{ ok: boolean }>(`/boards/${id}`, { method: "DELETE" }),
    chats: {
      list: (boardId: number) => request<BoardChat[]>(`/boards/${boardId}/chats`),
      create: (boardId: number, data: { chat_id: number; title?: string }) =>
        request<BoardChat>(`/boards/${boardId}/chats`, {
          method: "POST",
          body: JSON.stringify(data),
        }),
      remove: (id: number) =>
        request<{ ok: boolean }>(`/board-chats/${id}`, { method: "DELETE" }),
    },
  },

  columns: {
    create: (boardId: number, data: { name: string; color?: string }) =>
      request<Column>(`/boards/${boardId}/columns`, {
        method: "POST",
        body: JSON.stringify(data),
      }),
    update: (id: number, data: { name?: string; color?: string }) =>
      request<{ ok: boolean }>(`/columns/${id}`, {
        method: "PATCH",
        body: JSON.stringify(data),
      }),
    delete: (id: number) =>
      request<{ ok: boolean }>(`/columns/${id}`, { method: "DELETE" }),
    reorder: (boardId: number, columnIds: number[]) =>
      request<{ ok: boolean }>(`/boards/${boardId}/columns/reorder`, {
        method: "PATCH",
        body: JSON.stringify({ column_ids: columnIds }),
      }),
  },

  tasks: {
    listByBoard: (boardId: number) => request<Task[]>(`/boards/${boardId}/tasks`),
    get: (id: number) => request<TaskDetail>(`/tasks/${id}`),
    create: (boardId: number, data: { title: string; description?: string; column_id?: number; priority?: string; deadline?: string | null }) =>
      request<Task>(`/boards/${boardId}/tasks`, {
        method: "POST",
        body: JSON.stringify(data),
      }),
    update: (id: number, data: { title?: string; description?: string; priority?: string; deadline?: string | null }) =>
      request<Task>(`/tasks/${id}`, {
        method: "PATCH",
        body: JSON.stringify(data),
      }),
    move: (id: number, data: { column_id: number; position: number }) =>
      request<Task>(`/tasks/${id}/move`, {
        method: "PUT",
        body: JSON.stringify(data),
      }),
    delete: (id: number) =>
      request<{ ok: boolean }>(`/tasks/${id}`, { method: "DELETE" }),
    assign: (id: number, data: { user_id: number; username?: string; display_name?: string }) =>
      request<Assignee>(`/tasks/${id}/assign`, {
        method: "POST",
        body: JSON.stringify(data),
      }),
    unassign: (id: number, userId: number) =>
      request<{ ok: boolean }>(`/tasks/${id}/assign/${userId}`, {
        method: "DELETE",
      }),
  },

  checklists: {
    create: (taskId: number, title?: string) =>
      request<Checklist>(`/tasks/${taskId}/checklists`, {
        method: "POST",
        body: JSON.stringify({ title: title || "Чеклист" }),
      }),
    update: (id: number, data: { title?: string }) =>
      request<{ ok: boolean }>(`/checklists/${id}`, {
        method: "PATCH",
        body: JSON.stringify(data),
      }),
    delete: (id: number) =>
      request<{ ok: boolean }>(`/checklists/${id}`, { method: "DELETE" }),
    createItem: (checklistId: number, text: string) =>
      request<ChecklistItem>(`/checklists/${checklistId}/items`, {
        method: "POST",
        body: JSON.stringify({ text }),
      }),
    updateItem: (id: number, data: { text?: string; is_done?: boolean }) =>
      request<{ ok: boolean }>(`/checklist-items/${id}`, {
        method: "PATCH",
        body: JSON.stringify(data),
      }),
    deleteItem: (id: number) =>
      request<{ ok: boolean }>(`/checklist-items/${id}`, { method: "DELETE" }),
  },

  comments: {
    list: (taskId: number) => request<Comment[]>(`/tasks/${taskId}/comments`),
    create: (taskId: number, text: string) =>
      request<Comment>(`/tasks/${taskId}/comments`, {
        method: "POST",
        body: JSON.stringify({ text }),
      }),
    delete: (id: number) =>
      request<{ ok: boolean }>(`/comments/${id}`, { method: "DELETE" }),
  },
};
