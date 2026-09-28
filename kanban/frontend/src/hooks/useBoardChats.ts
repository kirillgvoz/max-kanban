import { useCallback, useState } from "react";
import { api } from "../api/client";
import type { BoardChat } from "../types";

export function useBoardChats(boardId: number | null) {
  const [chats, setChats] = useState<BoardChat[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchChats = useCallback(async () => {
    if (!boardId) return;
    try {
      setChats(await api.boards.chats.list(boardId));
    } finally {
      setLoading(false);
    }
  }, [boardId]);

  const addChat = useCallback(async (chatId: number, title?: string) => {
    if (!boardId) return;
    const chat = await api.boards.chats.create(boardId, { chat_id: chatId, title });
    setChats((previous) => [...previous, chat].sort((a, b) => a.id - b.id));
    return chat;
  }, [boardId]);

  const removeChat = useCallback(async (id: number) => {
    await api.boards.chats.remove(id);
    setChats((previous) => previous.filter((chat) => chat.id !== id));
  }, []);

  return { chats, loading, fetchChats, addChat, removeChat, setChats };
}
