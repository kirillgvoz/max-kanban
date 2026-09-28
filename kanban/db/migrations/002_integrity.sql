UPDATE organizations SET avatar_url = '' WHERE avatar_url IS NULL;
UPDATE org_members SET username = '' WHERE username IS NULL;
UPDATE org_members SET display_name = '' WHERE display_name IS NULL;
UPDATE boards SET description = '' WHERE description IS NULL;
UPDATE boards SET is_archived = FALSE WHERE is_archived IS NULL;
UPDATE columns SET color = '#6B7280' WHERE color IS NULL;
UPDATE tasks SET description = '' WHERE description IS NULL;
UPDATE tasks SET priority = 'medium' WHERE priority IS NULL;
UPDATE tasks SET updated_at = NOW() WHERE updated_at IS NULL;
UPDATE checklists SET title = 'Чеклист' WHERE title IS NULL;
UPDATE checklist_items SET is_done = FALSE WHERE is_done IS NULL;
UPDATE comments SET username = '' WHERE username IS NULL;
UPDATE comments SET display_name = '' WHERE display_name IS NULL;
UPDATE task_assignees SET username = '' WHERE username IS NULL;
UPDATE task_assignees SET display_name = '' WHERE display_name IS NULL;

ALTER TABLE organizations ALTER COLUMN avatar_url SET DEFAULT '';
ALTER TABLE organizations ALTER COLUMN avatar_url SET NOT NULL;
ALTER TABLE org_members ALTER COLUMN username SET DEFAULT '';
ALTER TABLE org_members ALTER COLUMN username SET NOT NULL;
ALTER TABLE org_members ALTER COLUMN display_name SET DEFAULT '';
ALTER TABLE org_members ALTER COLUMN display_name SET NOT NULL;
ALTER TABLE org_members ALTER COLUMN role SET DEFAULT 'member';
ALTER TABLE org_members ALTER COLUMN role SET NOT NULL;
ALTER TABLE boards ALTER COLUMN description SET DEFAULT '';
ALTER TABLE boards ALTER COLUMN description SET NOT NULL;
ALTER TABLE boards ALTER COLUMN is_archived SET DEFAULT FALSE;
ALTER TABLE boards ALTER COLUMN is_archived SET NOT NULL;
ALTER TABLE columns ALTER COLUMN color SET DEFAULT '#6B7280';
ALTER TABLE columns ALTER COLUMN color SET NOT NULL;
ALTER TABLE tasks ALTER COLUMN description SET DEFAULT '';
ALTER TABLE tasks ALTER COLUMN description SET NOT NULL;
ALTER TABLE tasks ALTER COLUMN priority SET DEFAULT 'medium';
ALTER TABLE tasks ALTER COLUMN priority SET NOT NULL;
ALTER TABLE tasks ALTER COLUMN updated_at SET DEFAULT NOW();
ALTER TABLE tasks ALTER COLUMN updated_at SET NOT NULL;
ALTER TABLE checklists ALTER COLUMN title SET DEFAULT 'Чеклист';
ALTER TABLE checklists ALTER COLUMN title SET NOT NULL;
ALTER TABLE checklist_items ALTER COLUMN is_done SET DEFAULT FALSE;
ALTER TABLE checklist_items ALTER COLUMN is_done SET NOT NULL;
ALTER TABLE comments ALTER COLUMN username SET DEFAULT '';
ALTER TABLE comments ALTER COLUMN username SET NOT NULL;
ALTER TABLE comments ALTER COLUMN display_name SET DEFAULT '';
ALTER TABLE comments ALTER COLUMN display_name SET NOT NULL;
ALTER TABLE task_assignees ALTER COLUMN username SET DEFAULT '';
ALTER TABLE task_assignees ALTER COLUMN username SET NOT NULL;
ALTER TABLE task_assignees ALTER COLUMN display_name SET DEFAULT '';
ALTER TABLE task_assignees ALTER COLUMN display_name SET NOT NULL;

DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'columns_id_board_unique') THEN
        ALTER TABLE columns ADD CONSTRAINT columns_id_board_unique UNIQUE (id, board_id);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'tasks_column_board_fkey') THEN
        ALTER TABLE tasks ADD CONSTRAINT tasks_column_board_fkey
            FOREIGN KEY (column_id, board_id) REFERENCES columns(id, board_id) ON DELETE CASCADE;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'columns_position_nonnegative') THEN
        ALTER TABLE columns ADD CONSTRAINT columns_position_nonnegative CHECK (position >= 0);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'tasks_position_nonnegative') THEN
        ALTER TABLE tasks ADD CONSTRAINT tasks_position_nonnegative CHECK (position >= 0);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'checklists_position_nonnegative') THEN
        ALTER TABLE checklists ADD CONSTRAINT checklists_position_nonnegative CHECK (position >= 0);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'checklist_items_position_nonnegative') THEN
        ALTER TABLE checklist_items ADD CONSTRAINT checklist_items_position_nonnegative CHECK (position >= 0);
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_columns_board ON columns(board_id, position, id);
DROP INDEX IF EXISTS idx_board_members_user;
