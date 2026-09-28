import { useRef, useState } from "react";
import { Typography, Avatar } from "@maxhub/max-ui";
import type { Task } from "../../types";

interface SwipeableCardProps {
  task: Task;
  onSwipeRight: () => void;
  onSwipeLeft: () => void;
  onDelete?: () => void;
  nextLabel: string;
  prevLabel: string;
}

const PRIORITY_COLORS: Record<string, string> = {
  low: "#6B7280",
  medium: "#3B82F6",
  high: "#F59E0B",
  urgent: "#EF4444",
};

const SWIPE_THRESHOLD = 70;

export function SwipeableCard({
  task,
  onSwipeRight,
  onSwipeLeft,
  onDelete,
  nextLabel,
  prevLabel,
}: SwipeableCardProps) {
  const [offset, setOffset] = useState(0);
  const [animating, setAnimating] = useState(false);
  const startX = useRef(0);
  const currentX = useRef(0);
  const swiping = useRef(false);

  const handleTouchStart = (e: React.TouchEvent) => {
    if (animating) return;
    startX.current = e.touches[0].clientX;
    currentX.current = startX.current;
    swiping.current = true;
  };

  const handleTouchMove = (e: React.TouchEvent) => {
    if (!swiping.current || animating) return;
    currentX.current = e.touches[0].clientX;
    const delta = currentX.current - startX.current;
    setOffset(delta);
  };

  const handleTouchEnd = () => {
    if (!swiping.current || animating) return;
    swiping.current = false;
    const delta = currentX.current - startX.current;

    if (delta > SWIPE_THRESHOLD) {
      setAnimating(true);
      setOffset(300);
      setTimeout(() => {
        onSwipeRight();
        setOffset(0);
        setAnimating(false);
      }, 200);
    } else if (delta < -SWIPE_THRESHOLD) {
      setAnimating(true);
      setOffset(-300);
      setTimeout(() => {
        onSwipeLeft();
        setOffset(0);
        setAnimating(false);
      }, 200);
    } else {
      setAnimating(true);
      setOffset(0);
      setTimeout(() => setAnimating(false), 150);
    }
  };

  const assignee = task.assignees?.[0];
  const initials = assignee?.display_name
    ? assignee.display_name
        .split(" ")
        .map((n) => n[0])
        .join("")
        .slice(0, 2)
        .toUpperCase()
    : "?";

  const showRightHint = offset > 30;
  const showLeftHint = offset < -30;

  return (
    <div className="swipe-container">
      <div
        className="swipe-hint swipe-hint--right"
        style={{ opacity: showRightHint ? Math.min((offset - 30) / 40, 1) : 0 }}
      >
        {nextLabel} →
      </div>
      <div
        className="swipe-hint swipe-hint--left"
        style={{ opacity: showLeftHint ? Math.min((Math.abs(offset) - 30) / 40, 1) : 0 }}
      >
        ← {prevLabel}
      </div>
      <div
        className="swipe-card"
        style={{
          transform: `translateX(${offset}px)`,
          transition: animating ? "transform 0.2s ease" : "none",
        }}
        onTouchStart={handleTouchStart}
        onTouchMove={handleTouchMove}
        onTouchEnd={handleTouchEnd}
      >
        <div className="task-card-header">
          <div
            className="task-priority-dot"
            style={{ backgroundColor: PRIORITY_COLORS[task.priority] }}
          />
          <Typography.Label>#{task.id}</Typography.Label>
          {onDelete && (
            <button
              className="task-delete-btn"
              onClick={(e) => {
                e.stopPropagation();
                onDelete();
              }}
              onPointerDown={(e) => e.stopPropagation()}
            >
              ×
            </button>
          )}
        </div>
        <Typography.Body className="task-title">{task.title}</Typography.Body>
        {task.description && (
          <Typography.Body className="task-description">
            {task.description.length > 60
              ? task.description.slice(0, 60) + "…"
              : task.description}
          </Typography.Body>
        )}
        <div className="task-card-footer">
          {task.deadline && (
            <span className="task-deadline">📅 {task.deadline}</span>
          )}
          {assignee && (
            <Avatar.Container size={22} form="circle">
              <Avatar.Text>{initials}</Avatar.Text>
            </Avatar.Container>
          )}
        </div>
      </div>
    </div>
  );
}
