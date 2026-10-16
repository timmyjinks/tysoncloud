-- Uninstalling the GitHub App deletes the user's github_connections row. The
-- old NO ACTION FK made that delete fail whenever a github_service referenced
-- the connection, leaving a stale row that blocked reconnecting via
-- UNIQUE(user_id). Detach the services instead: they keep running on their
-- last image and are relinked when the user reconnects.

alter table public.github_services
  alter column github_connection_id drop not null;

alter table public.github_services
  drop constraint github_services_github_connection_id_fkey;

alter table public.github_services
  add constraint github_services_github_connection_id_fkey
  foreign key (github_connection_id) references public.github_connections(id)
  on delete set null;
