import { renderHook } from "@testing-library/react";
import { parseStartParam, useStartParam } from "./useStartParam";

describe("parseStartParam", () => {
  it.each([
    ["board_12", { kind: "board", id: 12 }],
    ["task_3", { kind: "task", id: 3 }],
    ["  board_7  ", { kind: "board", id: 7 }],
  ])("parses %s", (raw, expected) => {
    expect(parseStartParam(raw)).toEqual(expected);
  });

  it.each([
    [""],
    ["board_"],
    ["board_0"],
    ["board_-1"],
    ["board_1.5"],
    ["board_9007199254740993"],
    ["org_1"],
    ["board_1 extra"],
    ["https://max.ru/bot?startapp=board_1"],
  ])("rejects %s", (raw) => {
    expect(parseStartParam(raw)).toBeNull();
  });

  it("rejects non-strings", () => {
    expect(parseStartParam(null)).toBeNull();
    expect(parseStartParam(undefined)).toBeNull();
    expect(parseStartParam(42)).toBeNull();
    expect(parseStartParam({ payload: "board_1" })).toBeNull();
  });
});

describe("useStartParam", () => {
  afterEach(() => {
    delete (window as any).WebApp;
    window.history.replaceState(null, "", "/");
  });

  it("reads the bridge start_param", () => {
    (window as any).WebApp = { initDataUnsafe: { start_param: "task_9" } };
    const { result } = renderHook(() => useStartParam());
    expect(result.current).toEqual({ kind: "task", id: 9 });
  });

  it("falls back to the startapp query parameter", () => {
    (window as any).WebApp = { initDataUnsafe: {} };
    window.history.replaceState(null, "", "/max-kanban/?startapp=board_4");
    const { result } = renderHook(() => useStartParam());
    expect(result.current).toEqual({ kind: "board", id: 4 });
  });

  it("returns null without a payload", () => {
    (window as any).WebApp = { initDataUnsafe: {} };
    const { result } = renderHook(() => useStartParam());
    expect(result.current).toBeNull();
  });
});
