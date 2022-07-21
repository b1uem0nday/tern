package migrate

const (
	createVersionTable      = `create table if not exists %s(version int4 not null); insert into %s(version) select 0 where 0=(select count(*) from %s)`
	forceInsertVersionTable = "insert into %s value $1"
	createVersionCheckFunc  = "create or update function checkVersion(current int) returns bool" +
		" LANGUAGE plpgsql IMMUTABLE STRICT PARALLEL SAFE as $$ declare vers integer;" +
		"begin select version into vers from version; return current = vers; end; $$"
)
