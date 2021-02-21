package sqlschema

import "github.com/homemade/whiz/sqlbuilder"

type SQLScript struct {
	ID                string
	DriverSpecificSQL sqlbuilder.DriverSpecificSQL
}

var SQLScripts = []SQLScript{
	{
		ID: "eventz#0",
		DriverSpecificSQL: sqlbuilder.DriverSpecificSQL{
			MySQL: `CREATE TABLE IF NOT EXISTS eventz (
        id int(10) unsigned NOT NULL AUTO_INCREMENT,
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
        model_data text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_520_ci,
        source_data text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_520_ci,
        sub_grp1_status tinyint(4) NOT NULL DEFAULT '0',
        sub_grp2_status tinyint(4) NOT NULL DEFAULT '0',
        sub_grp3_status tinyint(4) NOT NULL DEFAULT '0',
        sub_grp4_status tinyint(4) NOT NULL DEFAULT '0',
        sub_grp5_status tinyint(4) NOT NULL DEFAULT '0',
        on_hold tinyint(1) NOT NULL DEFAULT '0',
        PRIMARY KEY (id),
        UNIQUE KEY uix_eventz_event_id (event_id),
        KEY idx_eventz_user_id (user_id),
        KEY idx_eventz_sub_grp1_status (sub_grp1_status),
        KEY idx_eventz_sub_grp4_status (sub_grp4_status),
        KEY idx_eventz_sub_grp5_status (sub_grp5_status),
        KEY idx_eventz_on_hold (on_hold),
        KEY idx_eventz_deleted_at (deleted_at),
        KEY idx_eventz_event_created_at (event_created_at),
        KEY idx_eventz_event_source (event_source),
        KEY idx_eventz_event_uuid (event_uuid),
        KEY idx_eventz_model_type_action (model,type,action),
        KEY idx_eventz_sub_grp2_status (sub_grp2_status),
        KEY idx_eventz_sub_grp3_status (sub_grp3_status)
      ) ENGINE=InnoDB AUTO_INCREMENT=0 DEFAULT CHARSET=latin1;`,
			Postgres: `CREATE TABLE IF NOT EXISTS eventz (
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
		},
	},
	{
		ID: "eventz_sources#0",
		DriverSpecificSQL: sqlbuilder.DriverSpecificSQL{
			MySQL: `CREATE TABLE IF NOT EXISTS eventz_sources (
        id int(10) unsigned NOT NULL AUTO_INCREMENT,
        name varchar(255) NOT NULL,
        config text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_520_ci,
        PRIMARY KEY (id),
        UNIQUE KEY idx_eventz_sources_name (name)
      ) ENGINE=InnoDB AUTO_INCREMENT=0 DEFAULT CHARSET=latin1;`,
			Postgres: `CREATE TABLE IF NOT EXISTS eventz_sources (
        id SERIAL,
        name varchar(255) NOT NULL,
        config text,
        PRIMARY KEY (id),
        UNIQUE (name)
      );`,
		},
	},
	{
		ID: "eventz_locks#0",
		DriverSpecificSQL: sqlbuilder.DriverSpecificSQL{
			MySQL: `CREATE TABLE IF NOT EXISTS eventz_locks (
        name varchar(255) NOT NULL,
        created_at timestamp(6) NULL DEFAULT NULL,
        PRIMARY KEY (name)
      ) ENGINE=InnoDB DEFAULT CHARSET=latin1;`,
			Postgres: `CREATE TABLE IF NOT EXISTS eventz_locks (
        name varchar(255) NOT NULL,
        created_at timestamp(6) NULL DEFAULT NULL,
        PRIMARY KEY (name)
      );`,
		},
	},
	{
		ID: "eventz_subscriber_error_logs#0",
		DriverSpecificSQL: sqlbuilder.DriverSpecificSQL{
			MySQL: `CREATE TABLE IF NOT EXISTS eventz_subscriber_error_logs (
        id int(10) unsigned NOT NULL AUTO_INCREMENT,
        created_at timestamp(6) NULL DEFAULT NULL,
        updated_at timestamp(6) NULL DEFAULT NULL,
        deleted_at timestamp(6) NULL DEFAULT NULL,
        routine_name varchar(48) NOT NULL,
        routine_version varchar(48) NOT NULL,
        routine_instance varchar(48) NOT NULL,
        error_code smallint(6) NOT NULL DEFAULT '0',
        error_message text,
        status varchar(48) NOT NULL,
        event_id varchar(56) DEFAULT NULL,
        refer_entity varchar(255) DEFAULT NULL,
        refer_id varchar(255) DEFAULT NULL,
        PRIMARY KEY (id),
        KEY idx_eventz_subscriber_error_logs_updated_at (updated_at),
        KEY idx_eventz_subscriber_error_logs_deleted_at (deleted_at),
        KEY idx_eventz_subscriber_error_logs_event_id (event_id),
        KEY idx_eventz_subscriber_error_logs_refer_entity (refer_entity),
        KEY idx_eventz_subscriber_error_logs_refer_id (refer_id)
      ) ENGINE=InnoDB AUTO_INCREMENT=0 DEFAULT CHARSET=latin1;`,
			Postgres: `CREATE TABLE IF NOT EXISTS eventz_subscriber_error_logs (
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
		},
	},
	{
		ID: "eventz_subscriber_processed_logs#0",
		DriverSpecificSQL: sqlbuilder.DriverSpecificSQL{
			MySQL: `CREATE TABLE IF NOT EXISTS eventz_subscriber_processed_logs (
        id int(10) unsigned NOT NULL AUTO_INCREMENT,
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
        meta_data text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_520_ci,
        PRIMARY KEY (id),
        KEY idx_eventz_subscriber_processed_logs_refer_id (refer_id),
        KEY idx_eventz_subscriber_processed_logs_deleted_at (deleted_at),
        KEY idx_eventz_subscriber_processed_logs_event_id (event_id),
        KEY idx_eventz_subscriber_processed_logs_refer_entity (refer_entity)
      ) ENGINE=InnoDB AUTO_INCREMENT=0 DEFAULT CHARSET=latin1;`,
			Postgres: `CREATE TABLE IF NOT EXISTS eventz_subscriber_processed_logs (
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
		},
	},
	{
		ID: "eventz_subscribers#0",
		DriverSpecificSQL: sqlbuilder.DriverSpecificSQL{
			MySQL: `CREATE TABLE IF NOT EXISTS eventz_subscribers (
        id int(10) unsigned NOT NULL AUTO_INCREMENT,
        created_at timestamp(6) NULL DEFAULT NULL,
        updated_at timestamp(6) NULL DEFAULT NULL,
        deleted_at timestamp(6) NULL DEFAULT NULL,
        name varchar(48) NOT NULL,
        version varchar(48) NOT NULL,
        instance varchar(48) NOT NULL,
        meta_data text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_520_ci,
        last_run_at timestamp(6) NULL DEFAULT NULL,
        ` + "`group`" + ` tinyint(4) NOT NULL,
        priority tinyint(4) NOT NULL,
        last_error_at timestamp(6) NULL DEFAULT NULL,
        PRIMARY KEY (id),
        UNIQUE KEY idx_eventz_subscribers_routine_name_version_instance (name,version,instance),
        KEY idx_eventz_subscribers_deleted_at (deleted_at),
        KEY idx_eventz_subscribers_last_run_at (last_run_at),
        KEY idx_eventz_subscribers_group ` + "(`group`)" + `,
        KEY idx_eventz_subscribers_priority (priority),
        KEY idx_eventz_subscribers_last_error_at (last_error_at),
        KEY idx_eventz_subscribers_updated_at (updated_at)
      ) ENGINE=InnoDB AUTO_INCREMENT=0 DEFAULT CHARSET=latin1;`,
			Postgres: `CREATE TABLE IF NOT EXISTS eventz_subscribers (
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
		},
	},
}
