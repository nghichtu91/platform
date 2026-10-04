package sql_string

var SqlString = `create table if not exists user_group
(
id         int  auto_increment primary key,
name       char(64) null,
permission blob     null
)
charset = utf8;

create table if not exists users
(
email       char(64)      not null primary key,
pwd         char(64)      null,
user_group  int           null,
permission  blob          null,
create_time int default 0 null,
status      int default 0 null,
constraint users_email_uindex
unique (email)
) charset = utf8;

create table if not exists mails
(
    id               int auto_increment primary key,
    application_time int          null,
    proposer         varchar(255) null,
    ip               varchar(255) null,
    mail_type        int          null,
    mail_status      int          null,
    acid             varchar(255) null,
    mail             longtext     null
)
    charset = utf8;
create table if not exists operation_log
(
id               int auto_increment primary key,
user             tinytext null,
ip               tinytext null,
time             int      null,
operation_type   tinytext null,
operation_detail blob     null
)
charset = utf8;
`
