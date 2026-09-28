import httpx
from app.config import MAX_BOT_TOKEN, MAX_API_BASE


async def send_message(chat_id: int, text: str, attachments: list | None = None):
    payload = {"chat_id": chat_id, "text": text}
    if attachments:
        payload["attachments"] = attachments
    async with httpx.AsyncClient() as client:
        resp = await client.post(
            f"{MAX_API_BASE}/messages",
            headers={"Authorization": MAX_BOT_TOKEN},
            json=payload,
            timeout=10,
        )
        resp.raise_for_status()
        return resp.json()


async def send_task_notification(
    chat_id: int,
    task_id: int,
    task_title: str,
    assignee_name: str,
    deadline: str | None,
    board_id: int,
):
    deadline_line = f"\n📅 Дедлайн: {deadline}" if deadline else ""
    text = (
        f"📋 *Новая задача*\n\n"
        f"{task_title}\n"
        f"👤 Исполнитель: {assignee_name}"
        f"{deadline_line}"
    )
    keyboard = {
        "type": "inline_keyboard",
        "payload": {
            "buttons": [
                [
                    {
                        "type": "callback",
                        "text": "✅ Взять в работу",
                        "payload": f"take:{task_id}",
                    },
                    {
                        "type": "callback",
                        "text": "✔️ Готово",
                        "payload": f"done:{task_id}",
                    },
                ],
                [
                    {
                        "type": "open_app",
                        "text": "📋 Открыть доску",
                    },
                    {
                        "type": "link",
                        "text": "🔗 Задача",
                        "url": f"https://max.ru/se14445725_bot?startapp=task_{task_id}",
                    },
                ],
            ]
        },
    }
    return await send_message(chat_id, text, [keyboard])


async def send_status_change_notification(
    chat_id: int,
    task_id: int,
    task_title: str,
    new_status: str,
    actor_name: str,
):
    status_emoji = {"В работе": "🔄", "Готово": "✔️", "К выполнению": "📋"}
    emoji = status_emoji.get(new_status, "📋")
    text = (
        f"{emoji} *Изменение статуса*\n\n"
        f"Задача: {task_title}\n"
        f"Статус: {new_status}\n"
        f"Кто: {actor_name}"
    )
    keyboard = {
        "type": "inline_keyboard",
        "payload": {
            "buttons": [
                [
                    {
                        "type": "open_app",
                        "text": "📋 Открыть доску",
                    }
                ]
            ]
        },
    }
    return await send_message(chat_id, text, [keyboard])


async def register_commands():
    commands = [
        {"name": "start", "description": "Начать работу с TaskFlow"},
        {"name": "board", "description": "Открыть доску задач"},
        {"name": "tasks", "description": "Мои задачи"},
        {"name": "new", "description": "Создать новую задачу"},
    ]
    async with httpx.AsyncClient() as client:
        await client.patch(
            f"{MAX_API_BASE}/me/commands",
            headers={"Authorization": MAX_BOT_TOKEN},
            json={"commands": commands},
            timeout=10,
        )
