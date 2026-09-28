import { useState, useEffect } from "react";
import { MaxUI, Typography, Button, Spinner } from "@maxhub/max-ui";
import { useWebApp } from "./hooks/useWebApp";
import { useBoard } from "./hooks/useBoard";
import { useTasks } from "./hooks/useTasks";
import { BoardView } from "./components/Board/Board";
import { Header } from "./components/Layout/Header";
import { api } from "./api/client";
import type { Board } from "./types";

function App() {
  const { webApp, user } = useWebApp();
  const [boards, setBoards] = useState<Board[]>([]);
  const [selectedBoardId, setSelectedBoardId] = useState<number | null>(null);
  const [loadingBoards, setLoadingBoards] = useState(true);
  const [creating, setCreating] = useState(false);
  const [newBoardName, setNewBoardName] = useState("");

  const { board, loading: loadingBoard, refetch: refetchBoard } = useBoard(selectedBoardId);
  const { tasks, moveTask, createTask, deleteTask, refetch: refetchTasks } = useTasks(selectedBoardId);

  const currentUserId = user?.id || 0;
  const chatId = webApp?.initDataUnsafe?.chat?.id;

  useEffect(() => {
    loadBoards();
  }, [user?.id]);

  const loadBoards = async () => {
    setLoadingBoards(true);
    try {
      const data = await api.boards.list(currentUserId || undefined);
      setBoards(data);
    } catch {
    } finally {
      setLoadingBoards(false);
    }
  };

  const handleCreateBoard = async () => {
    if (!newBoardName.trim()) return;
    try {
      const board = await api.boards.create(newBoardName.trim(), currentUserId, chatId);
      setBoards((prev) => [board, ...prev]);
      setSelectedBoardId(board.id);
      setNewBoardName("");
      setCreating(false);
    } catch {}
  };

  const handleMoveTask = async (taskId: number, columnId: number, position: number) => {
    await moveTask(taskId, columnId, position);
  };

  const handleCreateTask = async (data: any, userId: number) => {
    return createTask(data, userId);
  };

  return (
    <MaxUI>
      <div className="app">
        <Header />

        {loadingBoards ? (
          <div className="loading-container">
            <Spinner />
          </div>
        ) : !selectedBoardId ? (
          <div className="boards-panel">
            <div className="boards-container">
              <Typography.Title variant="large-strong">Мои доски</Typography.Title>

              {boards.length === 0 ? (
                <div className="empty-state">
                  <Typography.Body>У вас пока нет досок</Typography.Body>
                </div>
              ) : (
                <div className="boards-list">
                  {boards.map((b) => (
                    <div
                      key={b.id}
                      className="board-item"
                      onClick={() => setSelectedBoardId(b.id)}
                    >
                      <Typography.Body>{b.name}</Typography.Body>
                      <Typography.Label>
                        {new Date(b.created_at).toLocaleDateString("ru-RU")}
                      </Typography.Label>
                    </div>
                  ))}
                </div>
              )}

              {creating ? (
                <div className="create-board-form">
                  <input
                    className="create-board-input"
                    value={newBoardName}
                    onChange={(e) => setNewBoardName(e.target.value)}
                    placeholder="Название доски"
                    autoFocus
                    onKeyDown={(e) => {
                      if (e.key === "Enter") handleCreateBoard();
                      if (e.key === "Escape") setCreating(false);
                    }}
                  />
                  <div className="create-board-actions">
                    <Button variant="secondary" onClick={() => setCreating(false)}>
                      Отмена
                    </Button>
                    <Button variant="primary" onClick={handleCreateBoard}>
                      Создать
                    </Button>
                  </div>
                </div>
              ) : (
                <Button variant="primary" onClick={() => setCreating(true)}>
                  + Новая доска
                </Button>
              )}
            </div>
          </div>
        ) : loadingBoard ? (
          <div className="loading-container">
            <Spinner />
          </div>
        ) : board ? (
          <div>
            <button className="back-btn" onClick={() => setSelectedBoardId(null)}>
              ← Назад к доскам
            </button>
            <BoardView
              board={board}
              tasks={tasks}
              onMoveTask={handleMoveTask}
              onCreateTask={handleCreateTask}
              onDeleteTask={deleteTask}
              currentUserId={currentUserId}
            />
          </div>
        ) : (
          <div className="loading-container">
            <Typography.Body>Доска не найдена</Typography.Body>
          </div>
        )}
      </div>
    </MaxUI>
  );
}

export default App;
