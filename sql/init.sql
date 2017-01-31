
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
