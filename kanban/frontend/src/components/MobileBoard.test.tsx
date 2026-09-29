import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MobileBoard } from "./MobileBoard";
import type { Board, Column, Task } from "../types";

const columns: Column[] = [
  { id: 1, board_id: 1, name: "К выполнению", position: 0, color: "#6366F1" },
  { id: 2, board_id: 1, name: "В работе", position: 1, color: "#F59E0B" },
  { id: 3, board_id: 1, name: "Готово", position: 2, color: "#10B981" },
];

const board: Board = {
  id: 1,
  org_id: 1,
  name: "Доска",
  description: "",
  is_archived: false,
  created_by: 1,
  created_at: new Date().toISOString(),
};

function task(id: number, columnId: number): Task {
  return {
    id,
    board_id: 1,
    column_id: columnId,
    title: `Задача ${id}`,
    description: "",
    position: 0,
    priority: "medium",
    deadline: null,
    created_by: 1,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
    assignees: [],
  };
}

function renderBoard(onMoveTask = vi.fn()) {
  const allTasks = [task(10, 1), task(20, 2), task(30, 3)];
  render(
    <MobileBoard
      board={board}
      tasks={allTasks}
      allTasks={allTasks}
      columns={columns}
      onMoveTask={onMoveTask}
      onSelectTask={vi.fn()}
      onCreateTask={vi.fn()}
    />
  );
  return onMoveTask;
}

describe("MobileBoard", () => {
  it("switches statuses by tapping a tab", () => {
    renderBoard();
    expect(screen.getByText("Задача 10")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("tab", { name: /В работе/ }));
    expect(screen.getByText("Задача 20")).toBeInTheDocument();
    expect(screen.queryByText("Задача 10")).not.toBeInTheDocument();
  });

  it("opens board settings from the mobile board", () => {
    const onOpenSettings = vi.fn();
    const allTasks = [task(10, 1)];
    render(
      <MobileBoard
        board={board}
        tasks={allTasks}
        allTasks={allTasks}
        columns={columns}
        onMoveTask={vi.fn()}
        onSelectTask={vi.fn()}
        onCreateTask={vi.fn()}
        canManage
        onOpenSettings={onOpenSettings}
      />
    );
    expect(screen.getByText("ID #1")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Настройки доски" }));
    expect(onOpenSettings).toHaveBeenCalledTimes(1);
  });

  it("moves a task to the next status on a right swipe", async () => {
    const onMoveTask = renderBoard();
    const card = screen.getByText("Задача 10").closest(".swipe-task-card");
    expect(card).not.toBeNull();
    fireEvent.touchStart(card!, { touches: [{ clientX: 100, clientY: 100 }] });
    fireEvent.touchMove(card!, { touches: [{ clientX: 190, clientY: 105 }] });
    fireEvent.touchEnd(card!);
    await waitFor(() => expect(onMoveTask).toHaveBeenCalledWith(10, 2, 1));
  });

  it("moves a task to the previous status on a left swipe", async () => {
    const onMoveTask = renderBoard();
    fireEvent.click(screen.getByRole("tab", { name: /В работе/ }));
    const card = screen.getByText("Задача 20").closest(".swipe-task-card");
    fireEvent.touchStart(card!, { touches: [{ clientX: 200, clientY: 100 }] });
    fireEvent.touchMove(card!, { touches: [{ clientX: 100, clientY: 105 }] });
    fireEvent.touchEnd(card!);
    await waitFor(() => expect(onMoveTask).toHaveBeenCalledWith(20, 1, 1));
  });

  it("does not change status for a vertical scroll gesture", () => {
    const onMoveTask = renderBoard();
    const card = screen.getByText("Задача 10").closest(".swipe-task-card");
    fireEvent.touchStart(card!, { touches: [{ clientX: 100, clientY: 100 }] });
    fireEvent.touchMove(card!, { touches: [{ clientX: 105, clientY: 220 }] });
    fireEvent.touchEnd(card!);
    expect(onMoveTask).not.toHaveBeenCalled();
  });
});
