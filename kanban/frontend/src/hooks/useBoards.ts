import { useState, useEffect, useCallback } from "react";
import { api } from "../api/client";
import type { Board, BoardDetail } from "../types";

export function useBoards(orgId: number | null) {
  const [boards, setBoards] = useState<Board[]>([]);
  const [loading, setLoading] = useState(true);
  const [includeArchived, setIncludeArchived] = useState(false);

  const fetchBoards = useCallback(async () => {
    if (!orgId) return;
    setLoading(true);
    try {
      const data = await api.boards.listByOrg(orgId, includeArchived);
      setBoards(data);
    } catch (e) {
      console.error("Failed to fetch boards:", e);
    } finally {
      setLoading(false);
    }
  }, [orgId, includeArchived]);

  useEffect(() => {
    fetchBoards();
  }, [fetchBoards]);

  const createBoard = useCallback(async (name: string, description?: string, columns?: Array<{ name: string; color?: string }>) => {
    if (!orgId) return;
    const board = await api.boards.create(orgId, { name, description, columns });
    setBoards((prev) => [board, ...prev]);
    return board;
  }, [orgId]);

  return { boards, loading, includeArchived, setIncludeArchived, createBoard, refetch: fetchBoards };
}

export function useBoard(id: number | null) {
  const [board, setBoard] = useState<BoardDetail | null>(null);
  const [loading, setLoading] = useState(true);

  const fetchBoard = useCallback(async () => {
    if (!id) return;
    try {
      const data = await api.boards.get(id);
      setBoard(data);
    } catch (e) {
      console.error("Failed to fetch board:", e);
    } finally {
      setLoading(false);
    }
  }, [id]);

  useEffect(() => {
    fetchBoard();
  }, [fetchBoard]);

  const updateBoard = useCallback(async (data: { name?: string; description?: string; is_archived?: boolean }) => {
    if (!id) return;
    await api.boards.update(id, data);
    setBoard((prev) => (prev ? { ...prev, ...data } : prev));
  }, [id]);

  const archiveBoard = useCallback(async (archived: boolean) => {
    if (!id) return;
    await api.boards.update(id, { is_archived: archived });
    setBoard((prev) => (prev ? { ...prev, is_archived: archived } : prev));
  }, [id]);

  return { board, loading, updateBoard, archiveBoard, refetch: fetchBoard, setBoard };
}
