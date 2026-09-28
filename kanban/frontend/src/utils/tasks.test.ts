import { filterTasks, sanitizeTasks, upsertTask } from "./tasks";
import type { Task } from "../types";

function task(overrides: Partial<Task> = {}): Task {
  return {
    id: 7,
    board_id: 1,
    column_id: 2,
    title: "Подготовить отчёт",
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

describe("upsertTask", () => {
  it("добавляет новую задачу", () => {
    const next = upsertTask([], task());
    expect(next).toHaveLength(1);
    expect(next[0].id).toBe(7);
  });

  it("не дублирует задачу, если WS-событие пришло раньше HTTP-ответа", () => {
    const socketTask = task({ title: "WS-версия" });
    const httpTask = task({ title: "HTTP-версия" });
    const afterSocket = upsertTask([], socketTask);
    const afterHttp = upsertTask(afterSocket, httpTask);

    expect(afterHttp).toHaveLength(1);
    expect(afterHttp[0].title).toBe("HTTP-версия");
  });

  it("обновляет существующую задачу при повторных событиях без изменения порядка", () => {
    const first = task({ id: 1, title: "Первая" });
    const second = task({ id: 2, title: "Вторая" });
    const initial = upsertTask(upsertTask([], first), second);
    const next = upsertTask(initial, task({ id: 1, title: "Первая обновлена" }));

    expect(next.map((item) => item.id)).toEqual([1, 2]);
    expect(next[0].title).toBe("Первая обновлена");
  });

  it("убирает дубли одного id при загрузке списка", () => {
    const duplicated = sanitizeTasks([task(), task({ title: "Повтор" })]);

    expect(duplicated).toHaveLength(1);
    expect(duplicated[0].title).toBe("Повтор");
  });

  it("фильтрует задачи по тексту, приоритету, исполнителю и просрочке", () => {
    const tasks = [
      task({ id: 1, title: "Отчёт по проекту", priority: "high", deadline: "2026-01-01", assignees: [{ task_id: 1, user_id: 2, display_name: "Аня" }] }),
      task({ id: 2, title: "Позвонить", priority: "low", deadline: "2030-01-01", assignees: [] }),
    ];
    expect(filterTasks(tasks, { query: "проекту", today: "2026-09-26" }).map((item) => item.id)).toEqual([1]);
    expect(filterTasks(tasks, { priority: "low", today: "2026-09-26" }).map((item) => item.id)).toEqual([2]);
    expect(filterTasks(tasks, { assigneeId: 2, today: "2026-09-26" }).map((item) => item.id)).toEqual([1]);
    expect(filterTasks(tasks, { overdueOnly: true, today: "2026-09-26" }).map((item) => item.id)).toEqual([1]);
  });
});
