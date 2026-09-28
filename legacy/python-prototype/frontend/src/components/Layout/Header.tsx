import { Typography } from "@maxhub/max-ui";

export function Header() {
  return (
    <header className="app-header">
      <div className="header-logo">
        <span className="header-icon">📋</span>
        <Typography.Title variant="medium-strong" style={{ margin: 0 }}>TaskFlow</Typography.Title>
      </div>
      <Typography.Body>MAX Kanban</Typography.Body>
    </header>
  );
}
