import { act, renderHook, waitFor } from "@testing-library/react";
import { useTasks } from "./useTasks";
import type { Task } from "../types";

function task(overrides: Partial<Task> = {}): Task {
  return {
    id: 7,
    board_id: 1,
    column_id: 2,
    title: "Новая задача",
    description: "",
    position: 0,
    priority: "medium",
    deadline: null,
    created_by: 1,
    created_at: "2026-09-25T00:00:00.000Z",
    updated_at: "2026-09-25T00:00:00.000Z",
    assignees: [],
    ...overrides,
  };
}

describe("useTasks", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it("не дублирует задачу при WS-событии до HTTP-ответа", async () => {
    const httpTask = task({ title: "Ответ сервера" });
    const socketTask = task({ title: "Событие сокета" });
    let resolveHttp!: (value: Response) => void;

    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: string, init?: RequestInit) => {
        if (input.endsWith("/boards/1/tasks") && init?.method === "POST") {
          return new Promise<Response>((resolve) => {
            resolveHttp = resolve;
          });
        }
        return {
          ok: true,
          json: async () => [],
        } as Response;
      })
    );

    const { result } = renderHook(() => useTasks(1));
    await waitFor(() => expect(result.current.loading).toBe(false));

    let created: Promise<Task | undefined> | undefined;
    act(() => {
      created = result.current.createTask({ title: "Новая задача", column_id: 2 });
    });
    act(() => {
      result.current.setTasks((previous) => [...previous, socketTask]);
    });
    await act(async () => {
      resolveHttp({ ok: true, json: async () => httpTask } as Response);
      await created;
    });

    expect(result.current.tasks).toHaveLength(1);
    expect(result.current.tasks[0].title).toBe("Ответ сервера");
  });
});
