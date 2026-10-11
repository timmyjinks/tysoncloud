-- EnsurePreviewEnvironmentService does select-then-insert and relies on the
-- insert failing to detect a concurrent writer. Without this constraint two
-- webhook deliveries can both insert, and every later .Single() lookup fails.

-- Drop duplicates that may already exist, keeping the oldest row.
delete from public.preview_environment_services a
using public.preview_environment_services b
where a.preview_environment_id = b.preview_environment_id
  and a.type = b.type
  and a.source_service_id = b.source_service_id
  and (a.created_at, a.id) > (b.created_at, b.id);

alter table public.preview_environment_services
  add constraint preview_environment_services_env_type_source_key
  unique (preview_environment_id, type, source_service_id);
