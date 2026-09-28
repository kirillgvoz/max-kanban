import hmac

from fastapi import APIRouter, HTTPException, Request
from app.database import get_db
from app.services.max_bot import send_message
from app.services.notification import notify_status_change
from app.services import task_service
from app.config import FRONTEND_URL, WEBHOOK_SECRET

router = APIRouter()


@router.post("")
async def handle_webhook(request: Request):
    received_secret = request.headers.get("X-Max-Bot-Api-Secret", "")
    if not hmac.compare_digest(received_secret, WEBHOOK_SECRET):
        raise HTTPException(status_code=401, detail="Invalid webhook secret")

    body = await request.json()
    update_type = body.get("update_type")

    try:
        if update_type == "message_callback":
            await _handle_callback(body)
        elif update_type == "message_created":
            await _handle_message(body)
        elif update_type in ("bot_started", "bot_added"):
            await _handle_bot_started(body)
    except Exception:
        pass

    return {"ok": True}


def _get_chat_id(body: dict) -> int:
    message = body.get("message") or {}
    recipient = message.get("recipient") or body.get("chat") or {}
    return int(
        body.get("chat_id")
        or recipient.get("chat_id")
        or recipient.get("id")
        or 0
    )


def _get_callback_data(body: dict) -> str:
    payload = body.get("payload") or {}
    return str(
        body.get("callback_data")
        or payload.get("callback_data")
        or payload.get("data")
        or body.get("data")
        or ""
    )


async def _move_task_to_column(task_id: int, column_name: str, actor_name: str):
    task = await task_service.get_task_detail(task_id)
    if not task:
        return
    db = await get_db()
    columns = await db.execute_fetchall(
        "SELECT id FROM columns WHERE board_id = ? AND name = ?",
        (task["board_id"], column_name),
    )
    if not columns:
        return
    await task_service.move_task(task_id, dict(columns[0])["id"], 0)
    await notify_status_change(task_id, column_name, actor_name)


async def _handle_callback(body: dict):
    callback_data = _get_callback_data(body)
    user = body.get("user") or {}
    username = user.get("username") or ""
    display_name = user.get("first_name") or ""

    try:
        if callback_data.startswith("take:"):
            await _move_task_to_column(
                int(callback_data.split(":", 1)[1]),
                "В работе",
                display_name or username or "Пользователь",
            )
        elif callback_data.startswith("done:"):
            await _move_task_to_column(
                int(callback_data.split(":", 1)[1]),
                "Готово",
                display_name or username or "Пользователь",
            )
    except (TypeError, ValueError):
        return


async def _handle_message(body: dict):
    message = body.get("message") or {}
    message_body = message.get("body") or {}
    text = str(message_body.get("text") or body.get("text") or "").strip()
    chat_id = _get_chat_id(body)
    if not chat_id:
        return

    if text == "/start":
        keyboard = {
            "type": "inline_keyboard",
            "payload": {
                "buttons": [
                    [{"type": "open_app", "text": "📋 Открыть TaskFlow"}],
                    [{"type": "link", "text": "📖 Помощь", "url": "https://kirillgvoz.ru/max-kanban/"}],
                ]
            },
        }
        await send_message(
            chat_id,
            "👋 *Добро пожаловать в TaskFlow!*\n\n"
            "Управляйте задачами команды прямо из MAX.\n\n"
            "▸ Нажмите кнопку ниже, чтобы открыть доску\n"
            "▸ Используйте /tasks для просмотра ваших задач\n"
            "▸ /new — создать новую задачу",
            [keyboard],
        )
    elif text == "/tasks":
        await send_message(chat_id, "📋 Откройте доску, чтобы увидеть ваши задачи.", [
            {"type": "inline_keyboard", "payload": {"buttons": [[{"type": "open_app", "text": "📋 Мои задачи"}]]}}
        ])
    elif text == "/new":
        await send_message(chat_id, "✨ Создайте задачу через доску.", [
            {"type": "inline_keyboard", "payload": {"buttons": [[{"type": "open_app", "text": "➕ Новая задача"}]]}}
        ])


async def _handle_bot_started(body: dict):
    user = body.get("user") or {}
    chat_id = _get_chat_id(body)
    if not chat_id:
        return
    first_name = user.get("first_name", "")
    await send_message(
        chat_id,
        f"👋 Привет, {first_name}!\n\n"
        "Я *TaskFlow* — помогаю управлять задачами команды.\n\n"
        "Нажмите кнопку ниже, чтобы начать:",
        [{"type": "inline_keyboard", "payload": {"buttons": [
            [{"type": "open_app", "text": "🚀 Открыть TaskFlow"}]
        ]}}],
    )
