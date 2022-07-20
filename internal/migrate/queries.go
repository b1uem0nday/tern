package migrate

const (
	createVersionTable      = `create table if not exists %s(version int4 not null); insert into %s(version) select 0 where 0=(select count(*) from %s)`
	forceInsertVersionTable = "insert into version value $1"
)
