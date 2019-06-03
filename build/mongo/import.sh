#!/bin/bash
mongoimport --db discovery --file /docker-entrypoint-initdb.d/streams.json --jsonArray
mongoimport --db discovery --file /docker-entrypoint-initdb.d/users.json --jsonArray
mongo discovery --eval "db.users.createIndex( { email: 1 }, { unique: true } )"