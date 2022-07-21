package migrate

const (
	createVersionTable      = `create table if not exists %s(version int4 not null, valid bool not null); insert into %s values(0, true)`
	checkVersionTableExists = "select count(*) from pg_catalog.pg_class where relname=$1 and relkind='r' and pg_table_is_visible(oid)"
	forceInsertVersionTable = "update %s set version = $1"
	checkFunctionsExists    = "select * from pg_catalog.pg_proc where proname='checkversion'"
	createVersionCheckFunc  = "create function checkVersion(current int) returns bool" +
		" LANGUAGE plpgsql IMMUTABLE STRICT PARALLEL SAFE as $$ declare vers integer;" +
		"begin select version into vers from %s; return current = vers; end; $$"
)
