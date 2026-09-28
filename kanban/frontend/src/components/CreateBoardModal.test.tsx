import { act, fireEvent, render, screen } from "@testing-library/react";
import { CreateBoardModal } from "./CreateBoardModal";

describe("CreateBoardModal", () => {
  it("создаёт доску с пользовательскими статусами", async () => {
    const onSubmit = vi.fn();
    render(<CreateBoardModal onSubmit={onSubmit} onClose={vi.fn()} />);

    fireEvent.change(screen.getByLabelText("Название *"), { target: { value: "Проект" } });
    fireEvent.change(screen.getByLabelText("Название", { selector: "#status-name-0" }), { target: { value: "Бэклог" } });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Создать" }));
    });

    expect(onSubmit).toHaveBeenCalledWith(expect.objectContaining({
      name: "Проект",
      columns: expect.arrayContaining([expect.objectContaining({ name: "Бэклог", color: "#6366F1" })]),
    }));
  });

  it("не разрешает создать доску без статусов", () => {
    const { container } = render(<CreateBoardModal onSubmit={vi.fn()} onClose={vi.fn()} />);
    fireEvent.change(screen.getByLabelText("Название *"), { target: { value: "Проект" } });
    const inputs = Array.from(container.querySelectorAll('input[id^="status-name-"]'));
    for (const input of inputs) {
      fireEvent.change(input, { target: { value: "   " } });
    }
    expect(screen.getByRole("button", { name: "Создать" })).toBeDisabled();
  });
});
