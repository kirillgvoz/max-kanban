import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { OrgSettings } from "./OrgSettings";
import type { AuthUser, OrgDetail } from "../types";

const user: AuthUser = { user_id: 1, username: "owner", display_name: "Owner" };

function detail(role: string): OrgDetail {
  return {
    id: 9,
    name: "Команда",
    slug: "komanda",
    avatar_url: "",
    created_by: 1,
    created_at: new Date().toISOString(),
    members: [
      { org_id: 9, user_id: 1, username: "owner", display_name: "Owner", role, joined_at: new Date().toISOString() },
      { org_id: 9, user_id: 2, username: "member", display_name: "Member", role: "member", joined_at: new Date().toISOString() },
    ],
  };
}

function mockFetch(responses: Array<{ method?: string; path: string; body: unknown }>) {
  const calls: Array<{ method?: string; body: unknown }> = [];
  const implementation = vi.fn(async (input: string, init?: RequestInit) => {
    calls.push({ method: init?.method, body: init?.body ? JSON.parse(init.body as string) : undefined });
    const response = responses.find((item) => input.endsWith(item.path) && (!item.method || item.method === init?.method));
    if (!response) throw new Error(`Unexpected fetch ${init?.method} ${input}`);
    return { ok: true, json: async () => response.body } as Response;
  });
  vi.stubGlobal("fetch", implementation);
  return { implementation, calls };
}

describe("OrgSettings", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it("позволяет владельцу переименовать организацию", async () => {
    const org = detail("owner");
    const { calls } = mockFetch([
      { path: "/api/orgs/9", body: org },
      { method: "PATCH", path: "/api/orgs/9", body: { ok: true } },
    ]);
    render(<OrgSettings orgId={9} user={user} onClose={vi.fn()} />);

    const name = await screen.findByLabelText("Название");
    fireEvent.change(name, { target: { value: "Новая команда" } });
    fireEvent.click(screen.getByRole("button", { name: "Сохранить" }));

    await waitFor(() => {
      const update = calls.find((call) => call.method === "PATCH");
      expect(update?.body).toEqual({ name: "Новая команда", avatar_url: undefined });
    });
  });

  it("скрывает управление у обычного участника", async () => {
    mockFetch([{ path: "/api/orgs/9", body: detail("member") }]);
    render(<OrgSettings orgId={9} user={user} onClose={vi.fn()} />);

    await screen.findByText("Участники (2)");
    expect(screen.queryByLabelText("Пригласить участника")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Сохранить" })).not.toBeInTheDocument();
  });
});
