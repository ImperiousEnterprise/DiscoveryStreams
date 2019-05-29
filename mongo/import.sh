#!/bin/bash
mongoimport --db discovery --file /docker-entrypoint-initdb.d/streams.json --jsonArray