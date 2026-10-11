-- services and github_services both route tc-<name>.tysonjenkins.dev through the
-- same gateway, but each table's UNIQUE (public_domain) only covers itself. This
-- registry holds every claimed public domain so a hostname can have one owner.

create table public.public_domains (
  domain            text primary key,
  service_id        uuid unique references public.services(id) on delete cascade,
  github_service_id uuid unique references public.github_services(id) on delete cascade,
  created_at        timestamp with time zone not null default now(),
  constraint public_domains_one_owner check (num_nonnulls(service_id, github_service_id) = 1)
);

alter table public.public_domains enable row level security;
revoke all on public.public_domains from anon, authenticated;

-- Runs after the BEFORE triggers that build the final tc-... domain.
create or replace function public.claim_public_domain()
  returns trigger
  language plpgsql
  security definer
  set search_path = ''
  as $function$
begin
  if tg_table_name = 'services' then
    delete from public.public_domains where service_id = new.id;
    insert into public.public_domains (domain, service_id) values (new.public_domain, new.id);
  else
    delete from public.public_domains where github_service_id = new.id;
    insert into public.public_domains (domain, github_service_id) values (new.public_domain, new.id);
  end if;
  return null;
end;
$function$;

-- Fail loudly instead of backfilling a registry that already has collisions.
do $$
declare
  conflicts text;
begin
  select string_agg(public_domain, ', ') into conflicts
  from (
    select public_domain from public.services
    intersect
    select public_domain from public.github_services
  ) c;

  if conflicts is not null then
    raise exception 'public_domain used by both services and github_services: %', conflicts;
  end if;
end $$;

insert into public.public_domains (domain, service_id)
select public_domain, id from public.services;

insert into public.public_domains (domain, github_service_id)
select public_domain, id from public.github_services;

create trigger trg_claim_service_public_domain
  after insert or update of public_domain on public.services
  for each row
  execute function public.claim_public_domain();

create trigger trg_claim_github_service_public_domain
  after insert or update of public_domain on public.github_services
  for each row
  execute function public.claim_public_domain();
