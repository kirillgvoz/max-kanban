import { act, fireEvent, render, screen } from "@testing-library/react";
import { CreateTaskModal } from "./CreateTaskModal";
import type { Column } from "../types";

const columns: Column[] = [
  { id: 1, board_id: 1, name: "К выполнению", position: 0, color: "#6366F1" },
];

describe("CreateTaskModal", () => {
  it("отправляет задачу только один раз при повторных нажатиях", async () => {
    let resolveCreate!: () => void;
    const onSubmit = vi.fn(
      () =>
        new Promise<void>((resolve) => {
          resolveCreate = resolve;
        })
    );

    render(<CreateTaskModal columns={columns} onSubmit={onSubmit} onClose={vi.fn()} />);
    fireEvent.change(screen.getByLabelText("Заголовок *"), { target: { value: "Новая задача" } });
    const createButton = screen.getByRole("button", { name: "Создать" });
    fireEvent.click(createButton);
    fireEvent.click(createButton);

    expect(onSubmit).toHaveBeenCalledTimes(1);
    expect(createButton).toBeDisabled();

    await act(async () => {
      resolveCreate();
    });
  });
});
