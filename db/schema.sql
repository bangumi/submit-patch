--
-- PostgreSQL database dump
--

-- Dumped from database version 16.10 (Debian 16.10-1.pgdg13+1)
-- Dumped by pg_dump version 17.4

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: edit_suggestion; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.edit_suggestion (
    id uuid NOT NULL,
    patch_id uuid NOT NULL,
    patch_type character varying(64) NOT NULL,
    text text NOT NULL,
    from_user integer NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    deleted_at timestamp with time zone
);


ALTER TABLE public.edit_suggestion OWNER TO postgres;

--
-- Name: episode_patch; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.episode_patch (
    id uuid NOT NULL,
    episode_id integer NOT NULL,
    state integer DEFAULT 0 NOT NULL,
    from_user_id integer NOT NULL,
    wiki_user_id integer DEFAULT 0 NOT NULL,
    reason text NOT NULL,
    original_name text,
    name text,
    original_name_cn text,
    name_cn text,
    original_duration character varying(255),
    duration character varying(255),
    original_airdate character varying(64),
    airdate character varying(64),
    original_description text,
    description text,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    deleted_at timestamp with time zone,
    reject_reason text DEFAULT ''::character varying NOT NULL,
    subject_id integer DEFAULT 0 NOT NULL,
    comments_count integer DEFAULT 0 NOT NULL,
    patch_desc text DEFAULT ''::text NOT NULL,
    ep integer,
    num_id bigint NOT NULL
);


ALTER TABLE public.episode_patch OWNER TO postgres;

--
-- Name: episode_patch_num_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.episode_patch_num_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.episode_patch_num_id_seq OWNER TO postgres;

--
-- Name: episode_patch_num_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.episode_patch_num_id_seq OWNED BY public.episode_patch.num_id;


--
-- Name: patch_tables_migrations; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.patch_tables_migrations (
    version bigint NOT NULL,
    dirty boolean NOT NULL
);


ALTER TABLE public.patch_tables_migrations OWNER TO postgres;

--
-- Name: patch_users; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.patch_users (
    user_id integer NOT NULL,
    username character varying(255) NOT NULL,
    nickname character varying(255) NOT NULL
);


ALTER TABLE public.patch_users OWNER TO postgres;

--
-- Name: subject_patch; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.subject_patch (
    id uuid NOT NULL,
    subject_id integer NOT NULL,
    state integer DEFAULT 0 NOT NULL,
    from_user_id integer NOT NULL,
    wiki_user_id integer DEFAULT 0 NOT NULL,
    reason text NOT NULL,
    name text,
    original_name text DEFAULT ''::text NOT NULL,
    infobox text,
    original_infobox text,
    summary text,
    original_summary text,
    nsfw boolean,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    deleted_at timestamp with time zone,
    reject_reason text DEFAULT ''::character varying NOT NULL,
    subject_type bigint DEFAULT 0 NOT NULL,
    comments_count integer DEFAULT 0 NOT NULL,
    patch_desc text DEFAULT ''::text NOT NULL,
    original_platform integer,
    platform integer,
    action integer DEFAULT 1,
    num_id bigint NOT NULL
);


ALTER TABLE public.subject_patch OWNER TO postgres;

--
-- Name: COLUMN subject_patch.action; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.subject_patch.action IS '1 for update 2 for create';


--
-- Name: subject_patch_num_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.subject_patch_num_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.subject_patch_num_id_seq OWNER TO postgres;

--
-- Name: subject_patch_num_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.subject_patch_num_id_seq OWNED BY public.subject_patch.num_id;


--
-- Name: episode_patch num_id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.episode_patch ALTER COLUMN num_id SET DEFAULT nextval('public.episode_patch_num_id_seq'::regclass);


--
-- Name: subject_patch num_id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.subject_patch ALTER COLUMN num_id SET DEFAULT nextval('public.subject_patch_num_id_seq'::regclass);


--
-- Name: edit_suggestion edit_suggestion_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.edit_suggestion
    ADD CONSTRAINT edit_suggestion_pkey PRIMARY KEY (id);


--
-- Name: episode_patch episode_patch_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.episode_patch
    ADD CONSTRAINT episode_patch_pkey PRIMARY KEY (id);


--
-- Name: patch_tables_migrations patch_tables_migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.patch_tables_migrations
    ADD CONSTRAINT patch_tables_migrations_pkey PRIMARY KEY (version);


--
-- Name: patch_users patch_users_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.patch_users
    ADD CONSTRAINT patch_users_pkey PRIMARY KEY (user_id);


--
-- Name: subject_patch subject_patch_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.subject_patch
    ADD CONSTRAINT subject_patch_pkey PRIMARY KEY (id);


--
-- Name: episode_patch_state_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX episode_patch_state_idx ON public.episode_patch USING btree (state);


--
-- Name: idx_edit_patch_lookup; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_edit_patch_lookup ON public.edit_suggestion USING btree (created_at, patch_id, patch_type);


--
-- Name: idx_episode_count; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_episode_count ON public.episode_patch USING btree (state, deleted_at);


--
-- Name: idx_episode_episode_id; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_episode_episode_id ON public.episode_patch USING btree (episode_id, state);


--
-- Name: idx_episode_patch_list; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_episode_patch_list ON public.episode_patch USING btree (created_at, state, deleted_at);


--
-- Name: idx_episode_patch_list2; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_episode_patch_list2 ON public.episode_patch USING btree (updated_at, state, deleted_at);


--
-- Name: idx_episode_patch_list3; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_episode_patch_list3 ON public.episode_patch USING btree (deleted_at, state, created_at);


--
-- Name: idx_episode_patch_num_id; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_episode_patch_num_id ON public.episode_patch USING btree (num_id);


--
-- Name: idx_episode_subject_id; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_episode_subject_id ON public.episode_patch USING btree (subject_id, state);


--
-- Name: idx_subject_count; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_subject_count ON public.subject_patch USING btree (state, deleted_at);


--
-- Name: idx_subject_id; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_subject_id ON public.subject_patch USING btree (subject_id);


--
-- Name: idx_subject_patch_list; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_subject_patch_list ON public.subject_patch USING btree (created_at, state, deleted_at);


--
-- Name: idx_subject_patch_list2; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_subject_patch_list2 ON public.subject_patch USING btree (updated_at, state, deleted_at);


--
-- Name: idx_subject_patch_list3; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_subject_patch_list3 ON public.subject_patch USING btree (deleted_at, state, created_at);


--
-- Name: idx_subject_patch_num_id; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_subject_patch_num_id ON public.subject_patch USING btree (num_id);


--
-- Name: idx_subject_subject_id; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_subject_subject_id ON public.subject_patch USING btree (subject_id, state);


--
-- PostgreSQL database dump complete
--
