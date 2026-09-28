import { useEffect, useState } from "react";
import { X } from "lucide-react";
import { api } from "../api/client";
import { useOrg } from "../hooks/useOrgs";
import type { AuthUser } from "../types";

interface OrgSettingsProps {
  orgId: number;
  user: AuthUser | null;
  onClose: () => void;
}

export function OrgSettings({ orgId, user, onClose }: OrgSettingsProps) {
  const { org, loading, refetch } = useOrg(orgId);
  const [name, setName] = useState("");
  const [avatar, setAvatar] = useState("");
  const [inviteId, setInviteId] = useState("");
  const [inviteName, setInviteName] = useState("");
  const [inviteRole, setInviteRole] = useState("member");
  const [pending, setPending] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    if (org) {
      setName(org.name);
      setAvatar(org.avatar_url || "");
    }
  }, [org]);

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => event.key === "Escape" && onClose();
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [onClose]);

  const currentRole = org?.members.find((member) => member.user_id === user?.user_id)?.role;
  const canManage = currentRole === "owner";

  const mutate = async (action: () => Promise<void>) => {
    setPending(true);
    setError("");
    try {
      await action();
      await refetch();
    } catch {
      setError("Не удалось сохранить изменения");
    } finally {
      setPending(false);
    }
  };

  const saveSettings = () => mutate(async () => {
    if (!name.trim()) return;
    await api.orgs.update(orgId, { name: name.trim(), avatar_url: avatar.trim() || undefined });
  });

  const addMember = () => mutate(async () => {
    const userId = Number(inviteId);
    if (!Number.isInteger(userId) || userId <= 0) {
      throw new Error("Invalid member");
    }
    await api.orgs.addMember(orgId, {
      user_id: userId,
      display_name: inviteName.trim() || undefined,
      role: inviteRole,
    });
    setInviteId("");
    setInviteName("");
    setInviteRole("member");
  });

  const changeRole = (userId: number, role: string) => mutate(async () => {
    await api.orgs.addMember(orgId, { user_id: userId, role });
  });

  const removeMember = (userId: number) => {
    if (!window.confirm("Удалить участника?")) return;
    void mutate(async () => {
      await api.orgs.removeMember(orgId, userId);
    });
  };

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-content" role="dialog" aria-modal="true" aria-label="Настройки организации" onClick={(event) => event.stopPropagation()}>
        <div className="modal-header">
          <h3 className="modal-header-title">Настройки организации</h3>
          <button className="modal-close" onClick={onClose} aria-label="Закрыть"><X size={20} /></button>
        </div>
        <div className="modal-body">
          {error && <div className="form-error">{error}</div>}
          {loading || !org ? (
            <div className="loading-spinner" />
          ) : (
            <>
              <div className="form-group">
                <label className="form-label" htmlFor="org-name">Название</label>
                <input id="org-name" className="input" value={name} onChange={(event) => setName(event.target.value)} disabled={!canManage} />
              </div>
              <div className="form-group">
                <label className="form-label" htmlFor="org-avatar">Аватар URL</label>
                <input id="org-avatar" className="input" value={avatar} onChange={(event) => setAvatar(event.target.value)} disabled={!canManage} />
              </div>
              {canManage && <button className="btn btn--primary" onClick={saveSettings} disabled={pending}>Сохранить</button>}
              <section className="modal-section">
                <div className="modal-section-header"><span className="modal-section-title">Участники ({org.members.length})</span></div>
                {org.members.map((member) => (
                  <div className="comment" key={member.user_id}>
                    <div className="comment-header">
                      <span className="comment-author">{member.display_name || member.username || `User ${member.user_id}`}</span>
                      <span className="comment-date">{member.role}</span>
                      {canManage && member.role !== "owner" && (
                        <>
                          <select className="select select--sm" value={member.role} onChange={(event) => changeRole(member.user_id, event.target.value)} aria-label={`Роль участника ${member.user_id}`}>
                            <option value="member">Участник</option>
                            <option value="admin">Администратор</option>
                          </select>
                          <button className="comment-delete" onClick={() => removeMember(member.user_id)} aria-label={`Удалить участника ${member.user_id}`}>Удалить</button>
                        </>
                      )}
                    </div>
                  </div>
                ))}
              </section>
              {canManage && (
                <section className="modal-section">
                  <div className="modal-section-header"><span className="modal-section-title">Пригласить участника</span></div>
                  <div className="form-row">
                    <div className="form-group form-group--half">
                      <label className="form-label" htmlFor="invite-id">MAX user ID</label>
                      <input id="invite-id" className="input" value={inviteId} onChange={(event) => setInviteId(event.target.value)} />
                    </div>
                    <div className="form-group form-group--half">
                      <label className="form-label" htmlFor="invite-name">Имя</label>
                      <input id="invite-name" className="input" value={inviteName} onChange={(event) => setInviteName(event.target.value)} />
                    </div>
                  </div>
                  <div className="form-row">
                    <div className="form-group form-group--half">
                      <label className="form-label" htmlFor="invite-role">Роль</label>
                      <select id="invite-role" className="select" value={inviteRole} onChange={(event) => setInviteRole(event.target.value)}>
                        <option value="member">Участник</option>
                        <option value="admin">Администратор</option>
                      </select>
                    </div>
                    <div className="form-group form-group--half form-group--end">
                      <button className="btn btn--primary" onClick={addMember} disabled={pending}>Добавить</button>
                    </div>
                  </div>
                </section>
              )}
            </>
          )}
        </div>
      </div>
    </div>
  );
}
