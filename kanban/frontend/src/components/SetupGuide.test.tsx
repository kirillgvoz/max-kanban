import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { SetupGuide } from "./SetupGuide";

function stubClipboard() {
  const writeText = vi.fn(async () => undefined);
  Object.defineProperty(window.navigator, "clipboard", { value: { writeText }, configurable: true });
  return writeText;
}

describe("SetupGuide", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it("показывает сценарий подключения", () => {
    render(<SetupGuide boardId={null} onClose={vi.fn()} />);
    expect(screen.getByText("Назначьте бота администратором")).toBeTruthy();
    expect(screen.getByText("/link board_<id>")).toBeTruthy();
    expect(screen.getByText("ID доски — в настройках доски, вкладка «Чаты». Бот закрепит сообщение доски в чате.")).toBeTruthy();
  });

  it("копирует команду привязки доски и показывает статус", async () => {
    const writeText = stubClipboard();
    vi.stubGlobal("fetch", vi.fn(async () => ({ ok: true, json: async () => [] }) as Response));
    render(<SetupGuide boardId={5} boardName="Доска" onClose={vi.fn()} />);
    expect(screen.getByText(/Привяжите доску «Доска»/)).toBeTruthy();
    fireEvent.click(screen.getAllByRole("button", { name: "Скопировать" })[1]);
    await waitFor(() => expect(writeText).toHaveBeenCalledWith("/link board_5"));
    await waitFor(() => expect(screen.getByText((_, element) => element?.textContent === "Привязанных чатов: 0. Бот закрепит сообщение доски в чате.")).toBeTruthy());
    await waitFor(() => expect(screen.getByText("Скопировано")).toBeTruthy());
  });

  it("закрывается по крестику", () => {
    const onClose = vi.fn();
    render(<SetupGuide boardId={null} onClose={onClose} />);
    fireEvent.click(screen.getByRole("button", { name: "Закрыть" }));
    expect(onClose).toHaveBeenCalledTimes(1);
  });
});
