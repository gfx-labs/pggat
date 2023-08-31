#!/bin/sh

openssl req -nodes -new -x509 -subj '/CN=spilo.dummy.org' -keyout /etc/ssl/certs/pgbouncer.key -out /etc/ssl/certs/pgbouncer.crt
/bin/pggat
