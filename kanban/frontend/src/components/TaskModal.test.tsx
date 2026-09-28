import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { TaskModal } from "./TaskModal";
import type { Column, OrgMember, TaskDetail } from "../types";

const columns: Column[] = [
  { id: 1, board_id: 1, name: "Первая", position: 0, color: "#6366F1" },
  { id: 2, board_id: 1, name: "Вторая", position: 1, color: "#F59E0B" },
];
const members: OrgMember[] = [
  { org_id: 1, user_id: 1, display_name: "Владелец", role: "owner", joined_at: "" },
  { org_id: 1, user_id: 2, display_name: "Исполнитель", role: "member", joined_at: "" },
];
const detail: TaskDetail = {
  id: 7,
  board_id: 1,
  column_id: 1,
  title: "Задача",
  description: "",
  position: 0,
  priority: "medium",
  deadline: null,
  created_by: 1,
  created_at: "",
  updated_at: "",
  assignees: [],
  checklists: [],
  comments: [],
};

function mockFetch(handler: (input: string, init?: RequestInit, count?: number) => unknown) {
  let count = 0;
  vi.stubGlobal("fetch", vi.fn(async (input: string, init?: RequestInit) => {
    count += 1;
    return { ok: true, json: async () => handler(input, init, count) } as Response;
  }));
}

describe("TaskModal", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it("меняет статус задачи", async () => {
    mockFetch((input, init) => {
      if (input.endsWith("/api/tasks/7/move")) return { ...detail, column_id: 2 };
      return { ...detail, column_id: input.includes("move") ? 2 : 1 };
    });
    const onChanged = vi.fn();
    render(<TaskModal taskId={7} columns={columns} members={members} user={{ user_id: 1 }} onClose={vi.fn()} onDeleted={vi.fn()} onChanged={onChanged} />);
    await screen.findByText("Задача");
    fireEvent.change(screen.getByLabelText("Статус"), { target: { value: "2" } });
    await waitFor(() => expect(onChanged).toHaveBeenCalledWith(expect.objectContaining({ column_id: 2 })));
  });

  it("назначает исполнителя", async () => {
    mockFetch((input, init) => {
      if (input.endsWith("/api/tasks/7/assign")) return { task_id: 7, user_id: 2, display_name: "Исполнитель" };
      return detail;
    });
    render(<TaskModal taskId={7} columns={columns} members={members} user={{ user_id: 1 }} onClose={vi.fn()} onDeleted={vi.fn()} onChanged={vi.fn()} />);
    await screen.findByText("Задача");
    fireEvent.change(screen.getByLabelText("Добавить исполнителя"), { target: { value: "2" } });
    fireEvent.click(screen.getByRole("button", { name: "Назначить" }));
    await screen.findByText("Исполнитель");
  });
});
