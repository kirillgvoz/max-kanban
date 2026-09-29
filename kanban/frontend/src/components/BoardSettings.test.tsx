import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { BoardSettings } from "./BoardSettings";
import type { AuthUser, BoardDetail, Column } from "../types";

const user: AuthUser = { user_id: 1, display_name: "Admin" };
const columns: Column[] = [
  { id: 1, board_id: 5, name: "Первая", position: 0, color: "#6366F1" },
  { id: 2, board_id: 5, name: "Вторая", position: 1, color: "#F59E0B" },
];
const board: BoardDetail = {
  id: 5,
  org_id: 1,
  name: "Доска",
  description: "",
  is_archived: false,
  created_by: 1,
  created_at: new Date().toISOString(),
  columns,
  members: [{ org_id: 1, user_id: 1, display_name: "Admin", role: "admin", joined_at: new Date().toISOString() }],
};

function mockFetch(handler: (input: string, init?: RequestInit) => unknown) {
  const calls: Array<{ method?: string; path: string; body: Record<string, any> }> = [];
  vi.stubGlobal("fetch", vi.fn(async (input: string, init?: RequestInit) => {
    calls.push({ method: init?.method, path: input, body: init?.body ? JSON.parse(init.body as string) : {} });
    return { ok: true, json: async () => handler(input, init) } as Response;
  }));
  return calls;
}

describe("BoardSettings", () => {
  const confirm = window.confirm;
  beforeEach(() => {
    window.confirm = vi.fn(() => true);
  });
  afterEach(() => {
    window.confirm = confirm;
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it("переименовывает доску", async () => {
    const calls = mockFetch(() => ({ ok: true }));
    render(<BoardSettings board={board} tasks={[]} user={user} canManage onChanged={vi.fn()} onArchived={vi.fn()} onClose={vi.fn()} />);
    fireEvent.change(screen.getByLabelText("Название"), { target: { value: "Новая доска" } });
    fireEvent.click(screen.getByRole("button", { name: "Сохранить" }));
    await waitFor(() => expect(calls.some((call) => call.method === "PATCH" && call.path.endsWith("/api/boards/5") && call.body.name === "Новая доска")).toBe(true));
  });

  it("меняет порядок статусов", async () => {
    const calls = mockFetch(() => ({ ok: true }));
    render(<BoardSettings board={board} tasks={[]} user={user} canManage onChanged={vi.fn()} onArchived={vi.fn()} onClose={vi.fn()} />);
    fireEvent.click(screen.getByRole("tab", { name: "Статусы" }));
    fireEvent.click(screen.getByRole("button", { name: "Переместить Вторая влево" }));
    await waitFor(() => expect(calls.some((call) => call.path.endsWith("/api/boards/5/columns/reorder") && JSON.stringify(call.body) === JSON.stringify({ column_ids: [2, 1] }))).toBe(true));
  });

  it("удаляет пустой статус", async () => {
    const calls = mockFetch(() => ({ ok: true }));
    render(<BoardSettings board={board} tasks={[]} user={user} canManage onChanged={vi.fn()} onArchived={vi.fn()} onClose={vi.fn()} />);
    fireEvent.click(screen.getByRole("tab", { name: "Статусы" }));
    const rows = screen.getAllByText("Удалить", { selector: "button" });
    fireEvent.click(rows[0]);
    await waitFor(() => expect(calls.some((call) => call.method === "DELETE" && call.path.endsWith("/api/columns/1"))).toBe(true));
  });

  it("привязывает чат", async () => {
    const calls = mockFetch((input, init) => {
      if (input.endsWith("/api/boards/5/chats")) {
        return init?.method === "POST"
          ? { id: 9, board_id: 5, chat_id: 77, title: "", created_by: 1, created_at: "" }
          : [];
      }
      return { ok: true };
    });
    render(<BoardSettings board={board} tasks={[]} user={user} canManage onChanged={vi.fn()} onArchived={vi.fn()} onClose={vi.fn()} />);
    fireEvent.click(screen.getByRole("tab", { name: "Чаты" }));
    await screen.findByText("Чаты пока не привязаны");
    fireEvent.change(screen.getByLabelText("Chat ID"), { target: { value: "77" } });
    fireEvent.click(screen.getByRole("button", { name: "Привязать" }));
    await waitFor(() => expect(calls.some((call) => call.method === "POST" && call.body.chat_id === 77)).toBe(true));
  });

  it("показывает и копирует команду привязки", async () => {
    mockFetch(() => []);
    const writeText = vi.fn();
    Object.defineProperty(window.navigator, "clipboard", { value: { writeText }, configurable: true });
    render(<BoardSettings board={board} tasks={[]} user={user} canManage onChanged={vi.fn()} onArchived={vi.fn()} onClose={vi.fn()} />);
    fireEvent.click(screen.getByRole("tab", { name: "Чаты" }));
    expect(await screen.findByText("/link board_5")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Скопировать команду" }));
    await waitFor(() => expect(writeText).toHaveBeenCalledWith("/link board_5"));
    expect(await screen.findByRole("button", { name: "Скопировано" })).toBeInTheDocument();
    delete (window.navigator as any).clipboard;
  });

  it("показывает предупреждение при удалении занятого статуса", async () => {
    mockFetch(() => ({ ok: true }));
    render(<BoardSettings board={board} tasks={[{ id: 9, board_id: 5, column_id: 1, title: "Задача", description: "", position: 0, priority: "medium", deadline: null, created_by: 1, created_at: "", updated_at: "", assignees: [] }]} user={user} canManage onChanged={vi.fn()} onArchived={vi.fn()} onClose={vi.fn()} />);
    fireEvent.click(screen.getByRole("tab", { name: "Статусы" }));
    const rows = screen.getAllByText("Удалить", { selector: "button" });
    fireEvent.click(rows[0]);
    expect(await screen.findByText("Сначала переместите все задачи из этого статуса")).toBeInTheDocument();
  });
});
