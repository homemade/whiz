package sqlschema

var Postgres = []string{`CREATE TABLE IF NOT EXISTS eventz (
    id SERIAL,
    created_at timestamp(6) NULL DEFAULT NULL,
    updated_at timestamp(6) NULL DEFAULT NULL,
    deleted_at timestamp(6) NULL DEFAULT NULL,
    event_id varchar(56) NOT NULL,
    event_created_at timestamp(6) NULL DEFAULT NULL,
    event_source varchar(48) DEFAULT NULL,
    event_uuid varchar(36) DEFAULT NULL,
    model varchar(48) NOT NULL,
    type varchar(48) NOT NULL,
    action varchar(48) NOT NULL,
    user_id varchar(48) DEFAULT NULL,
    model_data text,
    source_data text,
    sub_grp1_status smallint NOT NULL DEFAULT '0',
    sub_grp2_status smallint NOT NULL DEFAULT '0',
    sub_grp3_status smallint NOT NULL DEFAULT '0',
    sub_grp4_status smallint NOT NULL DEFAULT '0',
    sub_grp5_status smallint NOT NULL DEFAULT '0',
    on_hold smallint NOT NULL DEFAULT '0',
    PRIMARY KEY (id),
    UNIQUE (event_id)
  );`,
	`CREATE TABLE IF NOT EXISTS eventz_sources (
    id SERIAL,
    name varchar(255) NOT NULL,
    config text,
    PRIMARY KEY (id),
    UNIQUE (name)
  );`,
	`CREATE TABLE IF NOT EXISTS eventz_locks (
    name varchar(255) NOT NULL,
    created_at timestamp(6) NULL DEFAULT NULL,
    PRIMARY KEY (name)
  );`,
	`CREATE TABLE IF NOT EXISTS eventz_subscriber_error_logs (
    id SERIAL,
    created_at timestamp(6) NULL DEFAULT NULL,
    updated_at timestamp(6) NULL DEFAULT NULL,
    deleted_at timestamp(6) NULL DEFAULT NULL,
    routine_name varchar(48) NOT NULL,
    routine_version varchar(48) NOT NULL,
    routine_instance varchar(48) NOT NULL,
    error_code smallint NOT NULL DEFAULT '0',
    error_message text,
    status varchar(48) NOT NULL,
    event_id varchar(56) DEFAULT NULL,
    refer_entity varchar(255) DEFAULT NULL,
    refer_id varchar(255) DEFAULT NULL,
    PRIMARY KEY (id)
  );`,
	`CREATE TABLE IF NOT EXISTS eventz_subscriber_processed_logs (
    id SERIAL,
    created_at timestamp(6) NULL DEFAULT NULL,
    updated_at timestamp(6) NULL DEFAULT NULL,
    deleted_at timestamp(6) NULL DEFAULT NULL,
    event_id varchar(56) DEFAULT NULL,
    routine_name varchar(48) NOT NULL,
    routine_version varchar(48) NOT NULL,
    routine_instance varchar(48) NOT NULL,
    status varchar(48) NOT NULL,
    refer_entity varchar(255) DEFAULT NULL,
    refer_id varchar(255) DEFAULT NULL,
    meta_data text,
    PRIMARY KEY (id)
  );`,
	`CREATE TABLE IF NOT EXISTS eventz_subscribers (
    id SERIAL,
    created_at timestamp(6) NULL DEFAULT NULL,
    updated_at timestamp(6) NULL DEFAULT NULL,
    deleted_at timestamp(6) NULL DEFAULT NULL,
    name varchar(48) NOT NULL,
    version varchar(48) NOT NULL,
    instance varchar(48) NOT NULL,
    meta_data text,
    last_run_at timestamp(6) NULL DEFAULT NULL,
    "group" smallint NOT NULL,
    priority smallint NOT NULL,
    last_error_at timestamp(6) NULL DEFAULT NULL,
    PRIMARY KEY (id),
    UNIQUE (name,version,instance)
  );`,
}
