
drop table if exists sites;

DROP TABLE IF EXISTS users;
CREATE TABLE IF NOT EXISTS users (
    usr integer primary key,
    login text, -- optional unix login name
    firstname text,
    lastname text,
    email text,
    salt text,
    admin int default 0, 
    pw_hash text, -- bcrypt hashed password
    pw_salt text default (lower(hex(randomblob(32)))),
    apikey text default (lower(hex(randomblob(32))))
);

attach '../dcman/data.db' as dcman;
create table sites as select * from dcman.sites;
insert into users select * from dcman.users;
detach database dcman;

delete from users where usr = 3;
delete from users where usr > 8 and usr < 14;
delete from users where usr > 8 and usr < 14;
delete from users where usr = 16;
delete from users where usr = 19;
delete from users where usr > 20;


DROP TABLE IF EXISTS pxehosts;
CREATE TABLE IF NOT EXISTS pxehosts (
    id integer primary key,
    sitename text,
    hostname text
);

insert into pxehosts(sitename, hostname)
    values ('SFO', '10.100.63.63' ),
           ('NY7', '10.160.172.12' ),
           ('AMS', '10.210.160.18' ),
           ('SV3', '10.110.192.11' )
;

drop view if exists pxeview;
create view pxeview as
  select s.*, p.hostname
  from sites s
  left outer join pxehosts p on s.name = p.sitename
;


--DROP TABLE IF EXISTS "audit";
CREATE TABLE IF NOT EXISTS "audit" (
    aid integer primary key,
    usr integer,
    sti integer,
    hostname text,
    log text,
    ts timestamp DEFAULT CURRENT_TIMESTAMP
);

DROP VIEW IF EXISTS "audit_view"; CREATE VIEW IF NOT EXISTS "audit_view" as
    select a.*, u.login as user, s.name as site from audit a
    left outer join users u on a.usr = u.usr
    left outer join sites s on a.sti = s.sti
    ;

DROP TRIGGER IF EXISTS audit_view_insert;
CREATE TRIGGER audit_view_insert INSTEAD OF INSERT ON audit_view
BEGIN
    insert into audit 
        (usr, sti, hostname, log)
        values(NEW.usr, NEW.sti, NEW.hostname, NEW.log)
    ;
END;

DROP TABLE IF EXISTS real_events;
CREATE TABLE real_events (
    ID integer primary key,
    TS timestamp DEFAULT CURRENT_TIMESTAMP,
    Host text not null,
    Kind text,
    Msg text
);
drop index if exists events_host;
drop index if exists real_events_host;
drop index if exists real_events_ts;
create index real_events_host on real_events(host,ts);
create index real_events_ts on real_events(ts);

-- create sorted view so that default view is by TS descending

DROP VIEW IF EXISTS events;
CREATE VIEW events as select 
   id, datetime(ts,'localtime') as ts, host, kind, msg
   from real_events 
   order by ts desc
   ;

DROP TRIGGER IF EXISTS events_insert;
CREATE TRIGGER events_insert INSTEAD OF INSERT ON events 
BEGIN
    insert into real_events 
    (TS, Host, Kind, Msg) 
    values 
    (ifnull(NEW.TS,datetime('now', 'utc')), NEW.Host, NEW.Kind, NEW.Msg);
END;

DROP TRIGGER IF EXISTS events_update;
CREATE TRIGGER events_insert INSTEAD OF update ON events 
BEGIN
    update real_events set 
        TS=ifnull(NEW.TS, OLD.TS),
        Host=ifnull(NEW.Host, OLD.Host),
        Kind=ifnull(NEW.Kind, OLD.Kind),
        Msg=ifnull(NEW.Msg, OLD.Msg)
    where ID=OLD.ID;
END;

DROP TRIGGER IF EXISTS events_delete;
CREATE TRIGGER events_delete INSTEAD OF DELETE ON events 
BEGIN
    delete from real_events where ID=OLD.ID;
END;
