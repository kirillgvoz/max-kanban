import { renderHook, waitFor } from "@testing-library/react";
import { useWebApp } from "./useWebApp";
import { api } from "../api/client";

 describe("useWebApp", () => {
  afterEach(() => {
    delete (window as any).WebApp;
    vi.restoreAllMocks();
  });

  it("does not fail when the bridge has no expand method", async () => {
    (window as any).WebApp = { ready: vi.fn(), initData: "", initDataUnsafe: {} };
    const { result } = renderHook(() => useWebApp());
    await waitFor(() => expect(result.current.state).toBe("authenticated"));
    expect(result.current.user?.user_id).toBe(1);
  });

  it("uses the validated MAX identity", async () => {
    vi.spyOn(api.auth, "validate").mockResolvedValue({
      ok: true,
      user: { user_id: 42, username: "max", display_name: "MAX User" },
    });
    (window as any).WebApp = {
      ready: vi.fn(),
      expand: undefined,
      initData: "signed-data",
      initDataUnsafe: { user: { id: 42 } },
    };
    const { result } = renderHook(() => useWebApp());
    await waitFor(() => expect(result.current.state).toBe("authenticated"));
    expect(result.current.user?.user_id).toBe(42);
  });
});
