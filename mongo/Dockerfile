FROM mongo
COPY streams.json /docker-entrypoint-initdb.d/streams.json
COPY ./import.sh /docker-entrypoint-initdb.d/import.sh