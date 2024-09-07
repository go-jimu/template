create table user
(
    id         varchar(36)                                    not null
        primary key,
    name       varchar(24)                                    not null,
    password   varchar(128)                                   not null,
    email      varchar(48)                                    not null,
    version    smallint(2) unsigned default 0                 not null,
    created_at timestamp            default current_timestamp not null,
    updated_at timestamp            default current_timestamp not null on update current_timestamp,
    deleted_at timestamp            default NULL              null
);
