-- trg_preview_environments_updated_at sets new.updated_at, but the column never
-- existed, so every UPDATE on this table failed (including the comment_id writes,
-- whose errors the backend discards).

alter table public.preview_environments
  add column updated_at timestamp with time zone not null default now();
