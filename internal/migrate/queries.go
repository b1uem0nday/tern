package migrate

const (
	createVersionTable      = `create table if not exists %[1]s(version int4 not null, valid bool not null); insert into %[1]s values(0, true)`
	checkVersionTableExists = "select count(*) from pg_catalog.pg_class where relname=$1 and relkind='r' and pg_table_is_visible(oid)"

	createVersionCheckFunc = "create function %[1]s_check(current int) returns bool" +
		" LANGUAGE plpgsql IMMUTABLE STRICT PARALLEL SAFE as $$ declare vers integer;" +
		"begin select version into vers from %[1]s; return current <= vers; end; $$"
)
