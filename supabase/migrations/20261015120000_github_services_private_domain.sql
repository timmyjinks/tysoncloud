-- github_services got private_domain 'svc-<id>.tysoncloud', which doesn't resolve
-- in-cluster. Match services: svc-<id>.<project namespace>.svc.cluster.local.
create or replace function public.set_github_service_private_domain()
  returns trigger
  language plpgsql
  as $function$
declare
  v_namespace text;
begin
  select namespace into v_namespace
  from projects
  where id = new.project_id;

  if v_namespace is null then
    raise exception 'No project found for project_id %', new.project_id;
  end if;

  new.private_domain := 'svc-' || new.id || '.' || v_namespace || '.svc.cluster.local';

  return new;
end;
$function$;

drop trigger if exists github_services_private_domain_trigger on public.github_services;
drop function if exists public.generate_private_domain();

create trigger trg_set_github_service_private_domain
  before insert or update of project_id on public.github_services
  for each row
  execute function public.set_github_service_private_domain();

update public.github_services g
set private_domain = 'svc-' || g.id || '.' || p.namespace || '.svc.cluster.local'
from public.projects p
where p.id = g.project_id;
