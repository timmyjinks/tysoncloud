SET local check_function_bodies = off;

ALTER PUBLICATION "supabase_realtime" DROP TABLE "public"."testing";

ALTER TABLE "public"."deployments"
  DROP CONSTRAINT "deployments_status_check";

ALTER TABLE "public"."deployments"
  DROP CONSTRAINT "deployments_type_check";

ALTER TABLE "public"."error_logs"
  DROP CONSTRAINT "error_logs_deployment_id_fkey";

ALTER TABLE "public"."profiles"
  DROP CONSTRAINT "profiles_id_fkey";

ALTER TABLE "public"."testing"
  DROP CONSTRAINT "testing_user_id_fkey";

DROP TABLE "public"."error_logs";

DROP TABLE "public"."profiles";

DROP TABLE "public"."testing";

CREATE SCHEMA "deployment";

CREATE SCHEMA "infrastructure";

CREATE TABLE "public"."databases" (
  "id"              uuid                     NOT NULL DEFAULT gen_random_uuid(),
  "project_id"      uuid                     NOT NULL,
  "name"            text                     NOT NULL,
  "engine"          text                     NOT NULL,
  "port"            integer                  NOT NULL DEFAULT 5432,
  "storage"         integer                  NOT NULL DEFAULT 10,
  "internal_domain" text                     NOT NULL,
  "created_at"      timestamp with time zone NOT NULL DEFAULT now(),
  CONSTRAINT "databases_pkey" PRIMARY KEY (id),
  CONSTRAINT "databases_project_id_name_key" UNIQUE (project_id, name)
);

ALTER TABLE "public"."databases"
  ENABLE ROW LEVEL SECURITY;

CREATE TABLE "public"."environments" (
  "id"         uuid                     NOT NULL DEFAULT gen_random_uuid(),
  "project_id" uuid                     NOT NULL DEFAULT gen_random_uuid(),
  "name"       text                     NOT NULL,
  "namespace"  text                     NOT NULL,
  "created_at" timestamp with time zone NOT NULL DEFAULT now(),
  CONSTRAINT "environments_pkey" PRIMARY KEY (id)
);

ALTER TABLE "public"."environments"
  ENABLE ROW LEVEL SECURITY;

CREATE TABLE "public"."github_connections" (
  "id"              uuid                     NOT NULL DEFAULT gen_random_uuid(),
  "user_id"         text                     NOT NULL,
  "installation_id" bigint                   NOT NULL,
  "created_at"      timestamp with time zone NOT NULL DEFAULT now(),
  CONSTRAINT "github_connections_installation_id_key" UNIQUE (installation_id),
  CONSTRAINT "github_connections_pkey" PRIMARY KEY (id),
  CONSTRAINT "github_connections_user_id_key" UNIQUE (user_id)
);

ALTER TABLE "public"."github_connections"
  ENABLE ROW LEVEL SECURITY;

CREATE TABLE "public"."github_services" (
  "project_id"           uuid                     NOT NULL,
  "port"                 integer                  NOT NULL DEFAULT 3000,
  "repo_name"            text                     NOT NULL DEFAULT ''::text,
  "created_at"           timestamp with time zone NOT NULL DEFAULT now(),
  "id"                   uuid                     NOT NULL DEFAULT gen_random_uuid(),
  "name"                 text                     NOT NULL,
  "status"               text                     NOT NULL DEFAULT 'pending'::text,
  "private_domain"       text                     NOT NULL,
  "public_domain"        text                     NOT NULL,
  "repo_id"              bigint                   NOT NULL,
  "github_connection_id" uuid                     NOT NULL,
  "root_dir"             text                     NOT NULL DEFAULT '/'::text,
  "branch"               text                     NOT NULL DEFAULT 'main'::text,
  CONSTRAINT "github_services_pkey" PRIMARY KEY (id),
  CONSTRAINT "github_services_project_id_name_key" UNIQUE (project_id, name),
  CONSTRAINT "github_services_public_domain_key" UNIQUE (public_domain),
  CONSTRAINT "github_services_status_check" CHECK ((status = ANY (ARRAY['failed'::text, 'building'::text, 'deploying'::text, 'running'::text, 'stopped'::text, 'pending'::text])))
);

ALTER TABLE "public"."github_services"
  ENABLE ROW LEVEL SECURITY;

CREATE TABLE "public"."preview_environment_services" (
  "preview_environment_id" uuid                     NOT NULL,
  "type"                   text                     NOT NULL,
  "source_service_id"      uuid                     NOT NULL,
  "created_at"             timestamp with time zone NOT NULL,
  "id"                     uuid                     NOT NULL DEFAULT gen_random_uuid(),
  CONSTRAINT "preview_environment_services_pkey" PRIMARY KEY (id)
);

ALTER TABLE "public"."preview_environment_services"
  ENABLE ROW LEVEL SECURITY;

CREATE TABLE "public"."preview_environments" (
  "id"         uuid                     NOT NULL DEFAULT gen_random_uuid(),
  "project_id" uuid                     NOT NULL,
  "name"       text                     NOT NULL,
  "pr"         integer                  NOT NULL,
  "namespace"  text                     NOT NULL,
  "created_at" timestamp with time zone NOT NULL DEFAULT now(),
  "comment_id" bigint,
  CONSTRAINT "preview_environments_namespace_key" UNIQUE (namespace),
  CONSTRAINT "preview_environments_pkey" PRIMARY KEY (id)
);

ALTER TABLE "public"."preview_environments"
  ENABLE ROW LEVEL SECURITY;

CREATE TABLE "public"."projects" (
  "id"         uuid                     NOT NULL DEFAULT gen_random_uuid(),
  "created_at" timestamp with time zone NOT NULL DEFAULT now(),
  "name"       text                     NOT NULL,
  "user_id"    text                     NOT NULL,
  CONSTRAINT "projects_pkey" PRIMARY KEY (id),
  CONSTRAINT "projects_user_id_name_key" UNIQUE (user_id, name)
);

ALTER TABLE "public"."projects"
  ENABLE ROW LEVEL SECURITY;

CREATE TABLE "public"."services" (
  "project_id"     uuid                     NOT NULL,
  "port"           integer                  NOT NULL DEFAULT 3000,
  "image"          text                     NOT NULL DEFAULT ''::text,
  "created_at"     timestamp with time zone NOT NULL DEFAULT now(),
  "id"             uuid                     NOT NULL DEFAULT gen_random_uuid(),
  "name"           text                     NOT NULL,
  "status"         text                     NOT NULL DEFAULT 'pending'::text,
  "private_domain" text                     NOT NULL,
  "public_domain"  text                     NOT NULL,
  "environment_id" uuid                     NOT NULL DEFAULT gen_random_uuid(),
  CONSTRAINT "services_pkey" PRIMARY KEY (id),
  CONSTRAINT "services_project_id_name_key" UNIQUE (project_id, name),
  CONSTRAINT "services_public_domain_key" UNIQUE (public_domain),
  CONSTRAINT "services_status_check" CHECK ((status = ANY (ARRAY['failed'::text, 'running'::text, 'stopped'::text, 'pending'::text])))
);

ALTER TABLE "public"."services"
  ENABLE ROW LEVEL SECURITY;

CREATE TABLE "public"."volumes" (
  "id"         uuid                     NOT NULL DEFAULT gen_random_uuid(),
  "service_id" uuid                     NOT NULL,
  "storage_gb" integer                  NOT NULL DEFAULT 10,
  "mount_path" text                     NOT NULL,
  "created_at" timestamp with time zone NOT NULL DEFAULT now(),
  CONSTRAINT "volumes_pkey" PRIMARY KEY (id),
  CONSTRAINT "volumes_service_id_key" UNIQUE (service_id)
);

ALTER TABLE "public"."volumes"
  ENABLE ROW LEVEL SECURITY;

ALTER TABLE "public"."databases"
  ADD COLUMN "resource_name" text GENERATED ALWAYS AS (('db-'::text || (id)::text)) STORED NOT NULL;

ALTER TABLE "public"."github_services"
  ADD COLUMN "resource_name" text GENERATED ALWAYS AS (('svc-'::text || (id)::text)) STORED NOT NULL;

ALTER TABLE "public"."projects"
  ADD COLUMN "namespace" text GENERATED ALWAYS AS (('proj-'::text || (id)::text)) STORED NOT NULL;

ALTER TABLE "public"."services"
  ADD COLUMN "resource_name" text GENERATED ALWAYS AS (('svc-'::text || (id)::text)) STORED NOT NULL;

ALTER TABLE "public"."deployments"
  ALTER COLUMN "container_id" SET DEFAULT ''::text;

CREATE OR REPLACE FUNCTION public.create_database (
  p_project_id uuid,
  p_user_id    text,
  p_name       text,
  p_engine     text    DEFAULT ''::text,
  p_port       integer DEFAULT 3000,
  p_storage_gb integer DEFAULT 2
)
  RETURNS public.databases
  LANGUAGE plpgsql
  SECURITY DEFINER
  AS $function$
declare
  result databases;
begin
  -- verify the user owns the project before allowing the insert
  if not exists (
    select 1 from projects
    where id = p_project_id
      and user_id = p_user_id
  ) then
    raise exception 'project not found or not owned by user';
  end if;

  insert into databases (project_id, name, engine, port, storage)
  values (p_project_id, p_name, p_engine, p_port, p_storage_gb)
  returning * into result;

  return result;
end;
$function$;

CREATE OR REPLACE FUNCTION public.create_github_service (
  p_project_id           uuid,
  p_github_connection_id uuid,
  p_repo_id              bigint,
  p_user_id              text,
  p_repo_name            text,
  p_name                 text,
  p_root_dir             text,
  p_branch               text,
  p_domain               text    DEFAULT NULL::text,
  p_port                 integer DEFAULT NULL::integer
)
  RETURNS public.github_services
  LANGUAGE plpgsql
  SECURITY DEFINER
  SET search_path TO 'public'
  AS $function$
declare
    v_service public.github_services;
begin
    if not exists (
        select 1
        from public.projects p
        where p.id = p_project_id
          and p.user_id = p_user_id
    ) then
        raise exception 'Project not found';
    end if;

    if not exists (
        select 1
        from public.github_connections gc
        where gc.id = p_github_connection_id
          and gc.user_id = p_user_id
    ) then
        raise exception 'GitHub connection not found';
    end if;

    insert into public.github_services (
        project_id,
        github_connection_id,
        repo_id,
        repo_name,
        name,
        root_dir,
        branch,
        public_domain,
        port
    )
    values (
        p_project_id,
        p_github_connection_id,
        p_repo_id,
        p_repo_name,
        p_name,
        p_root_dir,
        p_branch,
        p_domain,
        p_port
    )
    returning * into v_service;

    return v_service;
end;
$function$;

CREATE OR REPLACE FUNCTION public.create_service (
  p_project_id uuid,
  p_user_id    text,
  p_name       text,
  p_port       integer DEFAULT 3000,
  p_image      text    DEFAULT ''::text
)
  RETURNS public.services
  LANGUAGE plpgsql
  SECURITY DEFINER
  AS $function$
declare
  result services;
begin
  -- verify the user owns the project before allowing the insert
  if not exists (
    select 1 from projects
    where id = p_project_id
      and user_id = p_user_id
  ) then
    raise exception 'project not found or not owned by user';
  end if;

  insert into services (project_id, name, port, image)
  values (p_project_id, p_name, p_port, p_image)
  returning * into result;

  return result;
end;
$function$;

CREATE OR REPLACE FUNCTION public.create_service (
  p_project_id uuid,
  p_user_id    text,
  p_name       text,
  p_port       integer DEFAULT 3000,
  p_image      text    DEFAULT ''::text,
  p_domain     text    DEFAULT NULL::text
)
  RETURNS public.services
  LANGUAGE plpgsql
  SECURITY DEFINER
  AS $function$
declare
  result services;
begin
  -- verify the user owns the project before allowing the insert
  if not exists (
    select 1 from projects
    where id = p_project_id
      and user_id = p_user_id
  ) then
    raise exception 'project not found or not owned by user';
  end if;

  insert into services (project_id, name, port, image, public_domain)
  values (p_project_id, p_name, p_port, p_image, p_domain)
  returning * into result;

  return result;
end;
$function$;

CREATE OR REPLACE FUNCTION public.create_volume (
  p_service_id uuid,
  p_user_id    text,
  p_mount_path text,
  p_storage_gb integer DEFAULT 10
)
  RETURNS public.volumes
  LANGUAGE plpgsql
  SECURITY DEFINER
  AS $function$
declare
  result volumes;
begin
  -- verify the user owns the service (via its project) before allowing the insert
  if not exists (
    select 1
    from services s
    join projects p on p.id = s.project_id
    where s.id = p_service_id
      and p.user_id = p_user_id
  ) then
    raise exception 'service not found or not owned by user';
  end if;

  insert into volumes (service_id, mount_path, storage_gb)
  values (p_service_id, p_mount_path, p_storage_gb)
  returning * into result;

  return result;
exception
  when unique_violation then
    raise exception 'service already has a volume';
end;
$function$;

CREATE OR REPLACE FUNCTION public.delete_database (
  p_id      uuid,
  p_user_id text
)
  RETURNS void
  LANGUAGE plpgsql
  SECURITY DEFINER
  AS $function$
begin
  delete from databases s
  using projects p
  where s.id = p_id
    and s.project_id = p.id
    and p.user_id = p_user_id;

  if not found then
    raise exception 'database not found or not owned by user';
  end if;
end;
$function$;

CREATE OR REPLACE FUNCTION public.delete_github_service (
  p_user_id text,
  p_id      uuid
)
  RETURNS void
  LANGUAGE plpgsql
  SECURITY DEFINER
  SET search_path TO 'public'
  AS $function$
begin
    delete from public.github_services gs
    using public.projects p
    where gs.id = p_id
      and gs.project_id = p.id
      and p.user_id = p_user_id;

    if not found then
        raise exception 'GitHub service not found';
    end if;
end;
$function$;

CREATE OR REPLACE FUNCTION public.delete_service (
  p_id      uuid,
  p_user_id text
)
  RETURNS void
  LANGUAGE plpgsql
  SECURITY DEFINER
  AS $function$
begin
  delete from services s
  using projects p
  where s.id = p_id
    and s.project_id = p.id
    and p.user_id = p_user_id;

  if not found then
    raise exception 'service not found or not owned by user';
  end if;
end;
$function$;

CREATE OR REPLACE FUNCTION public.delete_volume (
  p_service_id uuid,
  p_user_id    text
)
  RETURNS void
  LANGUAGE plpgsql
  SECURITY DEFINER
  AS $function$
begin
  delete from volumes v
  using services s, projects p
  where v.service_id = p_service_id
    and v.service_id = s.id
    and s.project_id = p.id
    and p.user_id = p_user_id;

  if not found then
    raise exception 'volume not found or not owned by user';
  end if;
end;
$function$;

CREATE OR REPLACE FUNCTION public.generate_private_domain()
  RETURNS TRIGGER
  LANGUAGE plpgsql
  AS $function$
begin
    new.private_domain := 'svc-' || new.id || '.tysoncloud';
    return new;
end;
$function$;

CREATE OR REPLACE FUNCTION public.get_all_environments()
  RETURNS TABLE (
    id         uuid,
    service_id text,
    key        text,
    val        text
  )
  LANGUAGE sql
  SECURITY DEFINER
  SET search_path TO 'public', 'vault'
  AS $function$
  select e.id, e.service_id, e.key, v.decrypted_secret as val
  from environments as e
  join vault.decrypted_secrets v on v.id = e.secret_id
$function$;

CREATE OR REPLACE FUNCTION public.get_environments (
  p_service_id uuid
)
  RETURNS TABLE (
    id         uuid,
    service_id text,
    key        text,
    val        text
  )
  LANGUAGE sql
  SECURITY DEFINER
  SET search_path TO 'public', 'vault'
  AS $function$
  select e.id, e.service_id, e.key, v.decrypted_secret as val
  from environments as e
  join vault.decrypted_secrets v on v.id = e.secret_id
  where e.service_id = p_service_id;
$function$;

CREATE OR REPLACE FUNCTION public.set_database_private_domain()
  RETURNS TRIGGER
  LANGUAGE plpgsql
  AS $function$
declare
  v_namespace text;
begin
  select namespace into v_namespace
  from projects
  where id = new.project_id;

  if v_namespace is null then
    raise exception 'No project found for project_id %', new.project_id;
  end if;

  new.internal_domain := 'db-' || new.id || '-rw.' || v_namespace || '.svc.cluster.local';

  return new;
end;
$function$;

CREATE OR REPLACE FUNCTION public.set_github_service_public_domain_insert()
  RETURNS TRIGGER
  LANGUAGE plpgsql
  AS $function$
BEGIN
  IF NEW.public_domain IS NULL THEN
    NEW.public_domain := 'tc-' || NEW.id || '.tysonjenkins.dev';
  ELSE
    NEW.public_domain := 'tc-' || NEW.public_domain || '.tysonjenkins.dev';
  END IF;

  RETURN NEW;
END;
$function$;

CREATE OR REPLACE FUNCTION public.set_github_service_public_domain_update()
  RETURNS TRIGGER
  LANGUAGE plpgsql
  AS $function$
BEGIN
  NEW.public_domain := 'tc-' || NEW.public_domain || '.tysonjenkins.dev';

  RETURN NEW;
END;
$function$;

CREATE OR REPLACE FUNCTION public.set_preview_environments_updated_at()
  RETURNS TRIGGER
  LANGUAGE plpgsql
  AS $function$
begin
  new.updated_at = now();
  return new;
end $function$;

CREATE OR REPLACE FUNCTION public.set_service_private_domain()
  RETURNS TRIGGER
  LANGUAGE plpgsql
  AS $function$
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

CREATE OR REPLACE FUNCTION public.set_service_public_domain()
  RETURNS TRIGGER
  LANGUAGE plpgsql
  AS $function$
BEGIN
  IF NEW.public_domain IS NULL THEN
    NEW.public_domain := 'tc-' || NEW.id::text || '.tysonjenkins.dev';
  ELSIF NEW.public_domain !~ '\-tc\.tysonjenkins\.dev$' THEN
    NEW.public_domain := NEW.public_domain || '-tc.tysonjenkins.dev';
  END IF;

  RETURN NEW;
END;
$function$;

CREATE OR REPLACE FUNCTION public.set_service_public_domain_insert()
  RETURNS TRIGGER
  LANGUAGE plpgsql
  AS $function$
BEGIN
  IF NEW.public_domain IS NULL THEN
    NEW.public_domain := 'tc-' || NEW.id::text || '.tysonjenkins.dev';
  ELSE
    NEW.public_domain := 'tc-' || NEW.public_domain || '.tysonjenkins.dev';
  END IF;

  RETURN NEW;
END;
$function$;

CREATE OR REPLACE FUNCTION public.set_service_public_domain_update()
  RETURNS TRIGGER
  LANGUAGE plpgsql
  AS $function$
BEGIN
  IF NEW.public_domain IS NOT NULL
     AND NEW.public_domain NOT LIKE 'tc-%.tysonjenkins.dev'
  THEN
    NEW.public_domain := 'tc-' || NEW.public_domain || '.tysonjenkins.dev';
  ELSE 
    NEW.public_domain := 'tc-' || NEW.id::text || '.tysonjenkins.dev';
  END IF;

  RETURN NEW;
END;
$function$;

CREATE OR REPLACE FUNCTION public.update_database (
  p_id         uuid,
  p_user_id    text,
  p_name       text,
  p_storage_gb integer DEFAULT 2
)
  RETURNS public.databases
  LANGUAGE plpgsql
  SECURITY DEFINER
  AS $function$
declare
  result databases;
begin
  update databases s
  set name = p_name,
      storage = p_storage_gb
  from projects p
  where s.id = p_id
    and s.project_id = p.id
    and p.user_id = p_user_id
  returning s.* into result;

  if not found then
    raise exception 'database not found or not owned by user';
  end if;

  return result;
end;
$function$;

CREATE OR REPLACE FUNCTION public.update_github_service (
  p_id            uuid,
  p_user_id       text,
  p_name          text    DEFAULT NULL::text,
  p_port          integer DEFAULT NULL::integer,
  p_public_domain text    DEFAULT NULL::text,
  p_branch        text    DEFAULT 'main'::text
)
  RETURNS public.github_services
  LANGUAGE plpgsql
  SECURITY DEFINER
  SET search_path TO 'public'
  AS $function$
declare
    v_service public.github_services;
begin
    if not exists (
        select 1
        from public.github_services gs
        join public.projects p
            on p.id = gs.project_id
        where gs.id = p_id
          and p.user_id = p_user_id
    ) then
        raise exception 'GitHub service not found';
    end if;

    update public.github_services
    set
        name = coalesce(p_name, name),
        port = coalesce(p_port, port),
        branch = coalesce(p_branch, branch),
        public_domain = coalesce(p_public_domain, public_domain)
    where id = p_id
    returning * into v_service;

    return v_service;
end;
$function$;

CREATE OR REPLACE FUNCTION public.update_github_service_status (
  p_id     uuid,
  p_status text
)
  RETURNS public.github_services
  LANGUAGE plpgsql
  SECURITY DEFINER
  SET search_path TO 'public'
  AS $function$
declare
    v_service public.github_services;
begin
    update public.github_services
    set status = p_status
    where id = p_id
    returning * into v_service;

    if not found then
        raise exception 'GitHub service not found';
    end if;

    return v_service;
end;
$function$;

CREATE OR REPLACE FUNCTION public.update_service (
  p_id      uuid,
  p_user_id text,
  p_name    text,
  p_image   text,
  p_port    integer
)
  RETURNS public.services
  LANGUAGE plpgsql
  SECURITY DEFINER
  AS $function$
declare
  result services;
begin
  update services s
  set name = p_name,
      image = p_image,
      port = p_port
  from projects p
  where s.id = p_id
    and s.project_id = p.id
    and p.user_id = p_user_id
  returning s.* into result;

  if not found then
    raise exception 'database not found or not owned by user';
  end if;

  return result;
end;
$function$;

CREATE OR REPLACE FUNCTION public.update_service (
  p_id      uuid,
  p_user_id text,
  p_name    text,
  p_image   text,
  p_port    integer,
  p_domain  text
)
  RETURNS public.services
  LANGUAGE plpgsql
  SECURITY DEFINER
  AS $function$
declare
  result services;
begin
  update services s
  set name = p_name,
      image = p_image,
      port = p_port,
      public_domain = p_domain
  from projects p
  where s.id = p_id
    and s.project_id = p.id
    and p.user_id = p_user_id
  returning s.* into result;

  if not found then
    raise exception 'database not found or not owned by user';
  end if;

  return result;
end;
$function$;

CREATE OR REPLACE FUNCTION public.update_service_status (
  p_id      uuid,
  p_user_id text,
  p_status  text
)
  RETURNS public.services
  LANGUAGE plpgsql
  SECURITY DEFINER
  AS $function$
declare
  result services;
begin
  update services s
  set status = p_status
  from projects p
  where s.id = p_id
    and s.project_id = p.id
    and p.user_id = p_user_id
  returning s.* into result;

  if not found then
    raise exception 'service not found or not owned by user';
  end if;

  return result;
end;
$function$;

ALTER TABLE "public"."deployments"
  ADD CONSTRAINT "deployments_status_check" CHECK ((status = ANY (ARRAY['failed'::text, 'running'::text, 'stopped'::text])));

ALTER TABLE "public"."deployments"
  ADD CONSTRAINT "deployments_type_check" CHECK ((type = ANY (ARRAY['compose'::text, 'docker'::text, 'slop'::text])));

ALTER TABLE "public"."github_services"
  ADD CONSTRAINT "github_services_github_connection_id_fkey" FOREIGN KEY (github_connection_id) REFERENCES public.github_connections(id);

ALTER TABLE "public"."preview_environment_services"
  ADD CONSTRAINT "preview_environment_services_preview_environment_id_fkey" FOREIGN KEY (preview_environment_id) REFERENCES public.preview_environments(id) ON UPDATE CASCADE
    ON DELETE CASCADE;

ALTER TABLE "public"."projects"
  ADD CONSTRAINT "projects_namespace_key" UNIQUE (namespace);

ALTER TABLE "public"."databases"
  ADD CONSTRAINT "databases_project_id_fkey" FOREIGN KEY (project_id) REFERENCES public.projects(id) ON UPDATE CASCADE ON DELETE CASCADE;

ALTER TABLE "public"."environments"
  ADD CONSTRAINT "environments_project_id_fkey" FOREIGN KEY (project_id) REFERENCES public.projects(id) ON UPDATE CASCADE ON DELETE CASCADE;

ALTER TABLE "public"."github_services"
  ADD CONSTRAINT "github_services_project_id_fkey" FOREIGN KEY (project_id) REFERENCES public.projects(id) ON UPDATE CASCADE ON DELETE CASCADE;

ALTER TABLE "public"."preview_environments"
  ADD CONSTRAINT "preview_environments_project_id_fkey" FOREIGN KEY (project_id) REFERENCES public.projects(id) ON UPDATE CASCADE ON DELETE CASCADE;

ALTER TABLE "public"."services"
  ADD CONSTRAINT "services_project_id_fkey" FOREIGN KEY (project_id) REFERENCES public.projects(id) ON UPDATE CASCADE ON DELETE CASCADE;

ALTER TABLE "public"."volumes"
  ADD CONSTRAINT "volumes_service_id_fkey" FOREIGN KEY (service_id) REFERENCES public.services(id) ON DELETE CASCADE;

CREATE TRIGGER trg_set_database_private_domain
  BEFORE INSERT OR UPDATE OF project_id ON public.databases
  FOR EACH ROW
  EXECUTE FUNCTION public.set_database_private_domain();

CREATE TRIGGER github_services_private_domain_trigger
  BEFORE INSERT ON public.github_services
  FOR EACH ROW
  EXECUTE FUNCTION public.generate_private_domain();

CREATE TRIGGER trg_set_github_service_public_domain_insert
  BEFORE INSERT ON public.github_services
  FOR EACH ROW
  EXECUTE FUNCTION public.set_github_service_public_domain_insert();

CREATE TRIGGER trg_set_github_service_public_domain_update
  BEFORE UPDATE OF public_domain ON public.github_services
  FOR EACH ROW
  EXECUTE FUNCTION public.set_github_service_public_domain_update();

CREATE TRIGGER trg_preview_environments_updated_at
  BEFORE UPDATE ON public.preview_environments
  FOR EACH ROW
  EXECUTE FUNCTION public.set_preview_environments_updated_at();

CREATE TRIGGER trg_set_service_private_domain
  BEFORE INSERT OR UPDATE OF project_id ON public.services
  FOR EACH ROW
  EXECUTE FUNCTION public.set_service_private_domain();

CREATE TRIGGER trg_set_service_public_domain_insert
  BEFORE INSERT ON public.services
  FOR EACH ROW
  EXECUTE FUNCTION public.set_service_public_domain_insert();

CREATE TRIGGER trg_set_service_public_domain_update
  BEFORE UPDATE OF public_domain ON public.services
  FOR EACH ROW
  EXECUTE FUNCTION public.set_service_public_domain_update();

COMMENT ON TABLE "public"."github_services" IS 'This is a duplicate of services';

GRANT EXECUTE ON FUNCTION "public"."create_database"(uuid, text, text, text, integer, integer) TO PUBLIC, "anon", "authenticated";

REVOKE ALL ON FUNCTION "public"."create_database"(uuid, text, text, text, integer, integer) FROM "postgres";

GRANT EXECUTE ON FUNCTION "public"."create_database"(uuid, text, text, text, integer, integer) TO "postgres";

GRANT EXECUTE ON FUNCTION "public"."create_database"(uuid, text, text, text, integer, integer) TO "service_role";

GRANT EXECUTE ON FUNCTION "public"."create_github_service"(uuid, uuid, bigint, text, text, text, text, text, text, integer) TO PUBLIC, "anon", "authenticated";

REVOKE ALL ON FUNCTION "public"."create_github_service"(uuid, uuid, bigint, text, text, text, text, text, text, integer) FROM "postgres";

GRANT EXECUTE ON FUNCTION "public"."create_github_service"(uuid, uuid, bigint, text, text, text, text, text, text, integer) TO "postgres";

GRANT EXECUTE ON FUNCTION "public"."create_github_service"(uuid, uuid, bigint, text, text, text, text, text, text, integer) TO "service_role";

GRANT EXECUTE ON FUNCTION "public"."create_service"(uuid, text, text, integer, text) TO PUBLIC, "anon", "authenticated";

REVOKE ALL ON FUNCTION "public"."create_service"(uuid, text, text, integer, text) FROM "postgres";

GRANT EXECUTE ON FUNCTION "public"."create_service"(uuid, text, text, integer, text) TO "postgres";

GRANT EXECUTE ON FUNCTION "public"."create_service"(uuid, text, text, integer, text) TO "service_role";

GRANT EXECUTE ON FUNCTION "public"."create_service"(uuid, text, text, integer, text, text) TO PUBLIC, "anon", "authenticated";

REVOKE ALL ON FUNCTION "public"."create_service"(uuid, text, text, integer, text, text) FROM "postgres";

GRANT EXECUTE ON FUNCTION "public"."create_service"(uuid, text, text, integer, text, text) TO "postgres";

GRANT EXECUTE ON FUNCTION "public"."create_service"(uuid, text, text, integer, text, text) TO "service_role";

GRANT EXECUTE ON FUNCTION "public"."create_volume"(uuid, text, text, integer) TO PUBLIC, "anon", "authenticated";

REVOKE ALL ON FUNCTION "public"."create_volume"(uuid, text, text, integer) FROM "postgres";

GRANT EXECUTE ON FUNCTION "public"."create_volume"(uuid, text, text, integer) TO "postgres";

GRANT EXECUTE ON FUNCTION "public"."create_volume"(uuid, text, text, integer) TO "service_role";

GRANT EXECUTE ON FUNCTION "public"."delete_database"(uuid, text) TO PUBLIC, "anon", "authenticated";

REVOKE ALL ON FUNCTION "public"."delete_database"(uuid, text) FROM "postgres";

GRANT EXECUTE ON FUNCTION "public"."delete_database"(uuid, text) TO "postgres";

GRANT EXECUTE ON FUNCTION "public"."delete_database"(uuid, text) TO "service_role";

GRANT EXECUTE ON FUNCTION "public"."delete_github_service"(text, uuid) TO PUBLIC, "anon", "authenticated";

REVOKE ALL ON FUNCTION "public"."delete_github_service"(text, uuid) FROM "postgres";

GRANT EXECUTE ON FUNCTION "public"."delete_github_service"(text, uuid) TO "postgres";

GRANT EXECUTE ON FUNCTION "public"."delete_github_service"(text, uuid) TO "service_role";

GRANT EXECUTE ON FUNCTION "public"."delete_service"(uuid, text) TO PUBLIC, "anon", "authenticated";

REVOKE ALL ON FUNCTION "public"."delete_service"(uuid, text) FROM "postgres";

GRANT EXECUTE ON FUNCTION "public"."delete_service"(uuid, text) TO "postgres";

GRANT EXECUTE ON FUNCTION "public"."delete_service"(uuid, text) TO "service_role";

GRANT EXECUTE ON FUNCTION "public"."delete_volume"(uuid, text) TO PUBLIC, "anon", "authenticated";

REVOKE ALL ON FUNCTION "public"."delete_volume"(uuid, text) FROM "postgres";

GRANT EXECUTE ON FUNCTION "public"."delete_volume"(uuid, text) TO "postgres";

GRANT EXECUTE ON FUNCTION "public"."delete_volume"(uuid, text) TO "service_role";

GRANT EXECUTE ON FUNCTION "public"."generate_private_domain"() TO PUBLIC, "anon", "authenticated";

REVOKE ALL ON FUNCTION "public"."generate_private_domain"() FROM "postgres";

GRANT EXECUTE ON FUNCTION "public"."generate_private_domain"() TO "postgres";

GRANT EXECUTE ON FUNCTION "public"."generate_private_domain"() TO "service_role";

GRANT EXECUTE ON FUNCTION "public"."get_all_environments"() TO PUBLIC, "anon", "authenticated";

REVOKE ALL ON FUNCTION "public"."get_all_environments"() FROM "postgres";

GRANT EXECUTE ON FUNCTION "public"."get_all_environments"() TO "postgres";

GRANT EXECUTE ON FUNCTION "public"."get_all_environments"() TO "service_role";

GRANT EXECUTE ON FUNCTION "public"."get_environments"(uuid) TO PUBLIC, "anon", "authenticated";

REVOKE ALL ON FUNCTION "public"."get_environments"(uuid) FROM "postgres";

GRANT EXECUTE ON FUNCTION "public"."get_environments"(uuid) TO "postgres";

GRANT EXECUTE ON FUNCTION "public"."get_environments"(uuid) TO "service_role";

GRANT EXECUTE ON FUNCTION "public"."set_database_private_domain"() TO PUBLIC, "anon", "authenticated";

REVOKE ALL ON FUNCTION "public"."set_database_private_domain"() FROM "postgres";

GRANT EXECUTE ON FUNCTION "public"."set_database_private_domain"() TO "postgres";

GRANT EXECUTE ON FUNCTION "public"."set_database_private_domain"() TO "service_role";

GRANT EXECUTE ON FUNCTION "public"."set_github_service_public_domain_insert"() TO PUBLIC, "anon", "authenticated";

REVOKE ALL ON FUNCTION "public"."set_github_service_public_domain_insert"() FROM "postgres";

GRANT EXECUTE ON FUNCTION "public"."set_github_service_public_domain_insert"() TO "postgres";

GRANT EXECUTE ON FUNCTION "public"."set_github_service_public_domain_insert"() TO "service_role";

GRANT EXECUTE ON FUNCTION "public"."set_github_service_public_domain_update"() TO PUBLIC, "anon", "authenticated";

REVOKE ALL ON FUNCTION "public"."set_github_service_public_domain_update"() FROM "postgres";

GRANT EXECUTE ON FUNCTION "public"."set_github_service_public_domain_update"() TO "postgres";

GRANT EXECUTE ON FUNCTION "public"."set_github_service_public_domain_update"() TO "service_role";

GRANT EXECUTE ON FUNCTION "public"."set_preview_environments_updated_at"() TO PUBLIC, "anon", "authenticated";

REVOKE ALL ON FUNCTION "public"."set_preview_environments_updated_at"() FROM "postgres";

GRANT EXECUTE ON FUNCTION "public"."set_preview_environments_updated_at"() TO "postgres";

GRANT EXECUTE ON FUNCTION "public"."set_preview_environments_updated_at"() TO "service_role";

GRANT EXECUTE ON FUNCTION "public"."set_service_private_domain"() TO PUBLIC, "anon", "authenticated";

REVOKE ALL ON FUNCTION "public"."set_service_private_domain"() FROM "postgres";

GRANT EXECUTE ON FUNCTION "public"."set_service_private_domain"() TO "postgres";

GRANT EXECUTE ON FUNCTION "public"."set_service_private_domain"() TO "service_role";

GRANT EXECUTE ON FUNCTION "public"."set_service_public_domain"() TO PUBLIC, "anon", "authenticated";

REVOKE ALL ON FUNCTION "public"."set_service_public_domain"() FROM "postgres";

GRANT EXECUTE ON FUNCTION "public"."set_service_public_domain"() TO "postgres";

GRANT EXECUTE ON FUNCTION "public"."set_service_public_domain"() TO "service_role";

GRANT EXECUTE ON FUNCTION "public"."set_service_public_domain_insert"() TO PUBLIC, "anon", "authenticated";

REVOKE ALL ON FUNCTION "public"."set_service_public_domain_insert"() FROM "postgres";

GRANT EXECUTE ON FUNCTION "public"."set_service_public_domain_insert"() TO "postgres";

GRANT EXECUTE ON FUNCTION "public"."set_service_public_domain_insert"() TO "service_role";

GRANT EXECUTE ON FUNCTION "public"."set_service_public_domain_update"() TO PUBLIC, "anon", "authenticated";

REVOKE ALL ON FUNCTION "public"."set_service_public_domain_update"() FROM "postgres";

GRANT EXECUTE ON FUNCTION "public"."set_service_public_domain_update"() TO "postgres";

GRANT EXECUTE ON FUNCTION "public"."set_service_public_domain_update"() TO "service_role";

GRANT EXECUTE ON FUNCTION "public"."update_database"(uuid, text, text, integer) TO PUBLIC, "anon", "authenticated";

REVOKE ALL ON FUNCTION "public"."update_database"(uuid, text, text, integer) FROM "postgres";

GRANT EXECUTE ON FUNCTION "public"."update_database"(uuid, text, text, integer) TO "postgres";

GRANT EXECUTE ON FUNCTION "public"."update_database"(uuid, text, text, integer) TO "service_role";

GRANT EXECUTE ON FUNCTION "public"."update_github_service"(uuid, text, text, integer, text, text) TO PUBLIC, "anon", "authenticated";

REVOKE ALL ON FUNCTION "public"."update_github_service"(uuid, text, text, integer, text, text) FROM "postgres";

GRANT EXECUTE ON FUNCTION "public"."update_github_service"(uuid, text, text, integer, text, text) TO "postgres";

GRANT EXECUTE ON FUNCTION "public"."update_github_service"(uuid, text, text, integer, text, text) TO "service_role";

GRANT EXECUTE ON FUNCTION "public"."update_github_service_status"(uuid, text) TO PUBLIC, "anon", "authenticated";

REVOKE ALL ON FUNCTION "public"."update_github_service_status"(uuid, text) FROM "postgres";

GRANT EXECUTE ON FUNCTION "public"."update_github_service_status"(uuid, text) TO "postgres";

GRANT EXECUTE ON FUNCTION "public"."update_github_service_status"(uuid, text) TO "service_role";

GRANT EXECUTE ON FUNCTION "public"."update_service"(uuid, text, text, text, integer) TO PUBLIC, "anon", "authenticated";

REVOKE ALL ON FUNCTION "public"."update_service"(uuid, text, text, text, integer) FROM "postgres";

GRANT EXECUTE ON FUNCTION "public"."update_service"(uuid, text, text, text, integer) TO "postgres";

GRANT EXECUTE ON FUNCTION "public"."update_service"(uuid, text, text, text, integer) TO "service_role";

GRANT EXECUTE ON FUNCTION "public"."update_service"(uuid, text, text, text, integer, text) TO PUBLIC, "anon", "authenticated";

REVOKE ALL ON FUNCTION "public"."update_service"(uuid, text, text, text, integer, text) FROM "postgres";

GRANT EXECUTE ON FUNCTION "public"."update_service"(uuid, text, text, text, integer, text) TO "postgres";

GRANT EXECUTE ON FUNCTION "public"."update_service"(uuid, text, text, text, integer, text) TO "service_role";

GRANT EXECUTE ON FUNCTION "public"."update_service_status"(uuid, text, text) TO PUBLIC, "anon", "authenticated";

REVOKE ALL ON FUNCTION "public"."update_service_status"(uuid, text, text) FROM "postgres";

GRANT EXECUTE ON FUNCTION "public"."update_service_status"(uuid, text, text) TO "postgres";

GRANT EXECUTE ON FUNCTION "public"."update_service_status"(uuid, text, text) TO "service_role";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."databases" TO "anon", "authenticated";

REVOKE ALL ON TABLE "public"."databases" FROM "postgres";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."databases" TO "postgres";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."databases" TO "service_role";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."environments" TO "anon", "authenticated";

REVOKE ALL ON TABLE "public"."environments" FROM "postgres";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."environments" TO "postgres";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."environments" TO "service_role";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."github_connections" TO "anon", "authenticated";

REVOKE ALL ON TABLE "public"."github_connections" FROM "postgres";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."github_connections" TO "postgres";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."github_connections" TO "service_role";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."github_services" TO "anon", "authenticated";

REVOKE ALL ON TABLE "public"."github_services" FROM "postgres";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."github_services" TO "postgres";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."github_services" TO "service_role";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."preview_environment_services" TO "anon", "authenticated";

REVOKE ALL ON TABLE "public"."preview_environment_services" FROM "postgres";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."preview_environment_services" TO "postgres";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."preview_environment_services" TO "service_role";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."preview_environments" TO "anon", "authenticated";

REVOKE ALL ON TABLE "public"."preview_environments" FROM "postgres";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."preview_environments" TO "postgres";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."preview_environments" TO "service_role";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."projects" TO "anon", "authenticated";

REVOKE ALL ON TABLE "public"."projects" FROM "postgres";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."projects" TO "postgres";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."projects" TO "service_role";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."services" TO "anon", "authenticated";

REVOKE ALL ON TABLE "public"."services" FROM "postgres";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."services" TO "postgres";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."services" TO "service_role";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."volumes" TO "anon", "authenticated";

REVOKE ALL ON TABLE "public"."volumes" FROM "postgres";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."volumes" TO "postgres";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."volumes" TO "service_role";

ALTER TABLE "public"."deployments"
  ALTER COLUMN "user_id" SET DEFAULT auth.uid();

