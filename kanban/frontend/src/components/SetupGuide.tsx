import { useEffect, useState } from "react";
import { X } from "lucide-react";
import { api } from "../api/client";
import type { BoardChat } from "../types";

interface SetupGuideProps {
  boardId: number | null;
  boardName?: string;
  onClose: () => void;
}

async function copyText(text: string): Promise<boolean> {
  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(text);
      return true;
    }
    const area = document.createElement("textarea");
    area.value = text;
    document.body.appendChild(area);
    area.select();
    const ok = document.execCommand("copy");
    area.remove();
    return ok;
  } catch {
    return false;
  }
}

export function SetupGuide({ boardId, boardName, onClose }: SetupGuideProps) {
  const [chats, setChats] = useState<BoardChat[]>([]);
  const [copied, setCopied] = useState("");
  const [error, setError] = useState("");
  const linkCommand = boardId ? `/link board_${boardId}` : "/link board_<id>";

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => event.key === "Escape" && onClose();
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [onClose]);

  useEffect(() => {
    if (!boardId) return;
    void (async () => {
      try {
        setChats(await api.boards.chats.list(boardId));
      } catch {
        setError("Не удалось загрузить привязанные чаты");
      }
    })();
  }, [boardId]);

  const copy = (text: string, key: string) => {
    void (async () => {
      if (await copyText(text)) {
        setCopied(key);
      } else {
        setError("Не удалось скопировать");
      }
    })();
  };

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-content" role="dialog" aria-modal="true" aria-label="Подключение чата" onClick={(event) => event.stopPropagation()}>
        <div className="modal-header">
          <h3 className="modal-header-title">Подключение чата</h3>
          <button className="modal-close" onClick={onClose} aria-label="Закрыть"><X size={20} /></button>
        </div>
        <div className="modal-body">
          {error && <div className="form-error">{error}</div>}
          <ol className="guide-steps">
            <li className="guide-step">
              <span className="guide-step-num" aria-hidden="true">1</span>
              <div className="guide-step-body">
                <div className="guide-step-title">Добавьте бота в рабочий чат</div>
                <div className="guide-step-text">В MAX откройте чат → добавить участника → найдите бота Max Канбан.</div>
              </div>
            </li>
            <li className="guide-step">
              <span className="guide-step-num" aria-hidden="true">2</span>
              <div className="guide-step-body">
                <div className="guide-step-title">Назначьте бота администратором</div>
                <div className="guide-step-text">Это обязательно: без прав администратора бот не видит сообщения чата. Настройки чата → Администраторы → добавить бота.</div>
              </div>
            </li>
            <li className="guide-step">
              <span className="guide-step-num" aria-hidden="true">3</span>
              <div className="guide-step-body">
                <div className="guide-step-title">Проверьте связь</div>
                <div className="guide-step-text">Отправьте в чат команду:</div>
                <div className="modal-section-actions">
                  <code className="modal-prop-value">/start</code>
                  <button className="btn btn--secondary btn--sm" onClick={() => copy("/start", "start")}>{copied === "start" ? "Скопировано" : "Скопировать"}</button>
                </div>
                <div className="guide-step-text">Бот должен ответить приветствием.</div>
              </div>
            </li>
            <li className="guide-step">
              <span className="guide-step-num" aria-hidden="true">4</span>
              <div className="guide-step-body">
                <div className="guide-step-title">Привяжите доску{boardName ? ` «${boardName}»` : ""}</div>
                <div className="guide-step-text">Отправьте в чат команду:</div>
                <div className="modal-section-actions">
                  <code className="modal-prop-value">{linkCommand}</code>
                  <button className="btn btn--secondary btn--sm" onClick={() => copy(linkCommand, "link")}>{copied === "link" ? "Скопировано" : "Скопировать"}</button>
                </div>
                {boardId ? (
                  <div className="guide-step-text">Привязанных чатов: {chats.length}. Бот закрепит сообщение доски в чате.</div>
                ) : (
                  <div className="guide-step-text">ID доски — в настройках доски, вкладка «Чаты». Бот закрепит сообщение доски в чате.</div>
                )}
              </div>
            </li>
            <li className="guide-step">
              <span className="guide-step-num" aria-hidden="true">5</span>
              <div className="guide-step-body">
                <div className="guide-step-title">Работайте прямо из чата</div>
                <div className="guide-step-text">/new Название задачи — создать, /tasks — список досок, кнопки «Взять» и «Готово» под уведомлениями двигают задачи по статусам.</div>
              </div>
            </li>
          </ol>
        </div>
      </div>
    </div>
  );
}
