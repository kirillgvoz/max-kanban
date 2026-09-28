import { useState, useEffect, useCallback } from "react";
import { api } from "../api/client";
import type { Board } from "../types";

export function useBoard(boardId: number | null) {
  const [board, setBoard] = useState<Board | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchBoard = useCallback(async () => {
    if (!boardId) return;
    setLoading(true);
    try {
      const data = await api.boards.get(boardId);
      setBoard(data);
      setError(null);
    } catch (e: any) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  }, [boardId]);

  useEffect(() => {
    fetchBoard();
  }, [fetchBoard]);

  return { board, loading, error, refetch: fetchBoard };
}
