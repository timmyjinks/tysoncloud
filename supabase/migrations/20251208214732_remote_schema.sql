SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;
COMMENT ON SCHEMA "public" IS 'standard public schema';
CREATE EXTENSION IF NOT EXISTS "pg_graphql" WITH SCHEMA "graphql";
CREATE EXTENSION IF NOT EXISTS "pg_stat_statements" WITH SCHEMA "extensions";
CREATE EXTENSION IF NOT EXISTS "pgcrypto" WITH SCHEMA "extensions";
CREATE EXTENSION IF NOT EXISTS "supabase_vault" WITH SCHEMA "vault";
CREATE EXTENSION IF NOT EXISTS "uuid-ossp" WITH SCHEMA "extensions";
CREATE OR REPLACE FUNCTION "public"."handle_new_user"() RETURNS "trigger"
    LANGUAGE "plpgsql" SECURITY DEFINER
    AS $$
begin
  insert into public.profiles (id)
  values(new.id);
  return new;
end;
$$;
ALTER FUNCTION "public"."handle_new_user"() OWNER TO "postgres";
SET default_tablespace = '';
SET default_table_access_method = "heap";
CREATE TABLE IF NOT EXISTS "public"."deployments" (
    "container_id" "text" NOT NULL,
    "user_id" "uuid" DEFAULT "gen_random_uuid"() NOT NULL,
    "name" "text" NOT NULL,
    "url" "text",
    "source" "text",
    "status" "text" DEFAULT 'exited'::"text",
    "created_at" timestamp with time zone DEFAULT "now"() NOT NULL,
    "type" "text" DEFAULT 'docker'::"text",
    "volume" "text" DEFAULT '/data'::"text",
    "id" "uuid" DEFAULT "gen_random_uuid"() NOT NULL,
    CONSTRAINT "deployments_status_check" CHECK (("status" = ANY (ARRAY['building'::"text", 'failed'::"text", 'running'::"text", 'generating'::"text", 'pulling'::"text", 'cloning'::"text", 'created'::"text", 'deploying'::"text"]))),
    CONSTRAINT "deployments_type_check" CHECK (("type" = ANY (ARRAY['git'::"text", 'docker'::"text", 'slop'::"text"])))
);
ALTER TABLE "public"."deployments" OWNER TO "postgres";
CREATE TABLE IF NOT EXISTS "public"."error_logs" (
    "deployment_id" "uuid" DEFAULT "gen_random_uuid"() NOT NULL,
    "log" "text"
);
ALTER TABLE "public"."error_logs" OWNER TO "postgres";
CREATE TABLE IF NOT EXISTS "public"."profiles" (
    "id" "uuid" DEFAULT "gen_random_uuid"() NOT NULL,
    "paid_tier" "text" DEFAULT 'free'::"text",
    "created_at" timestamp with time zone DEFAULT "now"(),
    CONSTRAINT "profiles_paid_tier_check" CHECK (("paid_tier" = ANY (ARRAY['free'::"text", 'dev'::"text", 'startup'::"text"])))
);
ALTER TABLE "public"."profiles" OWNER TO "postgres";
CREATE TABLE IF NOT EXISTS "public"."testing" (
    "id" "text" NOT NULL,
    "user_id" "uuid" DEFAULT "gen_random_uuid"() NOT NULL,
    "name" "text" NOT NULL,
    "url" "text",
    "source" "text",
    "status" "text" DEFAULT 'exited'::"text",
    "created_at" timestamp with time zone DEFAULT "now"() NOT NULL,
    "type" "text" DEFAULT 'docker'::"text",
    "volume" "text" DEFAULT '/data'::"text",
    CONSTRAINT "deployments_status_check" CHECK (("status" = ANY (ARRAY['building'::"text", 'failed'::"text", 'running'::"text"]))),
    CONSTRAINT "deployments_type_check" CHECK (("type" = ANY (ARRAY['git'::"text", 'docker'::"text"])))
);
ALTER TABLE "public"."testing" OWNER TO "postgres";
COMMENT ON TABLE "public"."testing" IS 'table for testing environments';
ALTER TABLE ONLY "public"."deployments"
    ADD CONSTRAINT "deployments_pkey" PRIMARY KEY ("id");
ALTER TABLE ONLY "public"."error_logs"
    ADD CONSTRAINT "error_logs_pkey" PRIMARY KEY ("deployment_id");
ALTER TABLE ONLY "public"."profiles"
    ADD CONSTRAINT "profiles_id_key" UNIQUE ("id");
ALTER TABLE ONLY "public"."profiles"
    ADD CONSTRAINT "profiles_pkey" PRIMARY KEY ("id");
ALTER TABLE ONLY "public"."testing"
    ADD CONSTRAINT "testing_pkey" PRIMARY KEY ("id");
ALTER TABLE ONLY "public"."deployments"
    ADD CONSTRAINT "deployments_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "auth"."users"("id") ON UPDATE CASCADE ON DELETE CASCADE;
ALTER TABLE ONLY "public"."error_logs"
    ADD CONSTRAINT "error_logs_deployment_id_fkey" FOREIGN KEY ("deployment_id") REFERENCES "public"."deployments"("id") ON UPDATE CASCADE ON DELETE CASCADE;
ALTER TABLE ONLY "public"."profiles"
    ADD CONSTRAINT "profiles_id_fkey" FOREIGN KEY ("id") REFERENCES "auth"."users"("id") ON UPDATE CASCADE ON DELETE CASCADE;
ALTER TABLE ONLY "public"."testing"
    ADD CONSTRAINT "testing_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "auth"."users"("id");
CREATE POLICY "Enable delete for users based on user_id" ON "public"."deployments" FOR DELETE USING ((( SELECT "auth"."uid"() AS "uid") = "user_id"));
CREATE POLICY "Enable insert for users based on user_id" ON "public"."deployments" FOR INSERT WITH CHECK ((( SELECT "auth"."uid"() AS "uid") = "user_id"));
CREATE POLICY "Enable users to update their own data only" ON "public"."deployments" FOR UPDATE TO "authenticated" USING ((( SELECT "auth"."uid"() AS "uid") = "user_id")) WITH CHECK ((( SELECT "auth"."uid"() AS "uid") = "user_id"));
CREATE POLICY "Enable users to view their own data only" ON "public"."deployments" FOR SELECT TO "authenticated" USING ((( SELECT "auth"."uid"() AS "uid") = "user_id"));
CREATE POLICY "Enable users to view their own data only" ON "public"."profiles" FOR SELECT TO "authenticated" USING ((( SELECT "auth"."uid"() AS "uid") = "id"));
ALTER TABLE "public"."deployments" ENABLE ROW LEVEL SECURITY;
ALTER TABLE "public"."error_logs" ENABLE ROW LEVEL SECURITY;
ALTER TABLE "public"."profiles" ENABLE ROW LEVEL SECURITY;
ALTER PUBLICATION "supabase_realtime" OWNER TO "postgres";
ALTER PUBLICATION "supabase_realtime" ADD TABLE ONLY "public"."deployments";
ALTER PUBLICATION "supabase_realtime" ADD TABLE ONLY "public"."testing";
GRANT USAGE ON SCHEMA "public" TO "postgres";
GRANT USAGE ON SCHEMA "public" TO "anon";
GRANT USAGE ON SCHEMA "public" TO "authenticated";
GRANT USAGE ON SCHEMA "public" TO "service_role";
GRANT ALL ON FUNCTION "public"."handle_new_user"() TO "anon";
GRANT ALL ON FUNCTION "public"."handle_new_user"() TO "authenticated";
GRANT ALL ON FUNCTION "public"."handle_new_user"() TO "service_role";
GRANT ALL ON TABLE "public"."deployments" TO "anon";
GRANT ALL ON TABLE "public"."deployments" TO "authenticated";
GRANT ALL ON TABLE "public"."deployments" TO "service_role";
GRANT ALL ON TABLE "public"."error_logs" TO "anon";
GRANT ALL ON TABLE "public"."error_logs" TO "authenticated";
GRANT ALL ON TABLE "public"."error_logs" TO "service_role";
GRANT ALL ON TABLE "public"."profiles" TO "anon";
GRANT ALL ON TABLE "public"."profiles" TO "authenticated";
GRANT ALL ON TABLE "public"."profiles" TO "service_role";
GRANT ALL ON TABLE "public"."testing" TO "anon";
GRANT ALL ON TABLE "public"."testing" TO "authenticated";
GRANT ALL ON TABLE "public"."testing" TO "service_role";
ALTER DEFAULT PRIVILEGES FOR ROLE "postgres" IN SCHEMA "public" GRANT ALL ON SEQUENCES TO "postgres";
ALTER DEFAULT PRIVILEGES FOR ROLE "postgres" IN SCHEMA "public" GRANT ALL ON SEQUENCES TO "anon";
ALTER DEFAULT PRIVILEGES FOR ROLE "postgres" IN SCHEMA "public" GRANT ALL ON SEQUENCES TO "authenticated";
ALTER DEFAULT PRIVILEGES FOR ROLE "postgres" IN SCHEMA "public" GRANT ALL ON SEQUENCES TO "service_role";
ALTER DEFAULT PRIVILEGES FOR ROLE "postgres" IN SCHEMA "public" GRANT ALL ON FUNCTIONS TO "postgres";
ALTER DEFAULT PRIVILEGES FOR ROLE "postgres" IN SCHEMA "public" GRANT ALL ON FUNCTIONS TO "anon";
ALTER DEFAULT PRIVILEGES FOR ROLE "postgres" IN SCHEMA "public" GRANT ALL ON FUNCTIONS TO "authenticated";
ALTER DEFAULT PRIVILEGES FOR ROLE "postgres" IN SCHEMA "public" GRANT ALL ON FUNCTIONS TO "service_role";
ALTER DEFAULT PRIVILEGES FOR ROLE "postgres" IN SCHEMA "public" GRANT ALL ON TABLES TO "postgres";
ALTER DEFAULT PRIVILEGES FOR ROLE "postgres" IN SCHEMA "public" GRANT ALL ON TABLES TO "anon";
ALTER DEFAULT PRIVILEGES FOR ROLE "postgres" IN SCHEMA "public" GRANT ALL ON TABLES TO "authenticated";
ALTER DEFAULT PRIVILEGES FOR ROLE "postgres" IN SCHEMA "public" GRANT ALL ON TABLES TO "service_role";
drop extension if exists "pg_net";
CREATE TRIGGER on_auth_user_created AFTER INSERT ON auth.users FOR EACH ROW EXECUTE FUNCTION public.handle_new_user();
