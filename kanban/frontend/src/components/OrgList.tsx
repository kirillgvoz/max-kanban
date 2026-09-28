import { useState } from "react";
import { Building2, Plus } from "lucide-react";
import { useOrgs } from "../hooks/useOrgs";
import type { AuthUser } from "../types";

type View =
  | { type: "orgs" }
  | { type: "boards"; orgId: number; orgName: string }
  | { type: "board"; boardId: number; orgId: number; orgName: string; boardName: string };

interface OrgListProps {
  user: AuthUser | null;
  onNavigate: (v: View) => void;
}

export function OrgList({ user, onNavigate }: OrgListProps) {
  const { orgs, loading, createOrg } = useOrgs();
  const [showCreate, setShowCreate] = useState(false);
  const [name, setName] = useState("");

  const handleCreate = async () => {
    if (!name.trim()) return;
    const org = await createOrg(name.trim());
    setName("");
    setShowCreate(false);
    onNavigate({ type: "boards", orgId: org.id, orgName: org.name });
  };

  if (loading) {
    return (
      <div className="content-loading">
        <div className="loading-spinner" />
      </div>
    );
  }

  return (
    <div className="org-list">
      <div className="content-header">
        <h1 className="content-title">Организации</h1>
        <button className="btn btn--primary" onClick={() => setShowCreate(true)}>
          <Plus size={16} />
          Новая
        </button>
      </div>

      {showCreate && (
        <div className="create-form">
          <input
            className="input"
            placeholder="Название организации"
            value={name}
            onChange={(e) => setName(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && handleCreate()}
            autoFocus
          />
          <div className="create-form-actions">
            <button className="btn btn--secondary" onClick={() => setShowCreate(false)}>
              Отмена
            </button>
            <button className="btn btn--primary" onClick={handleCreate}>
              Создать
            </button>
          </div>
        </div>
      )}

      <div className="org-grid">
        {orgs.map((org) => (
          <div
            key={org.id}
            className="org-card"
            onClick={() => onNavigate({ type: "boards", orgId: org.id, orgName: org.name })}
          >
            <div className="org-card-avatar">
              {org.name[0]?.toUpperCase()}
            </div>
            <div className="org-card-info">
              <div className="org-card-name">{org.name}</div>
              <div className="org-card-slug">/{org.slug}</div>
            </div>
          </div>
        ))}

        {orgs.length === 0 && !showCreate && (
          <div className="empty-state">
            <Building2 size={48} className="empty-state-icon" aria-hidden="true" />
            <div className="empty-state-text">Нет организаций</div>
            <div className="empty-state-hint">Создайте первую организацию</div>
          </div>
        )}
      </div>
    </div>
  );
}
