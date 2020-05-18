#!/bin/sh

# Terminate running PostgreSQL container if there is one
docker stop be-postgresql || true
docker rm be-postgresql || true
# Build PostgreSQL Docker image with custom postgresql.conf
OLD_PWD=`pwd`
echo "pwd: $OLD_PWD"
cd postgresql-docker
docker build --no-cache -t be-postgresql .
cd $OLD_PWD
# Start a new PostgreSQL container
docker run -d --name be-postgresql -e POSTGRES_DB=postgres -e POSTGRES_HOST_AUTH_METHOD=trust -p 5432:5432 be-postgresql:latest
# Make sure PostgreSQL is up
for i in {1..30}; do
    if [[ $(bash -c "docker exec -t be-postgresql psql -h localhost -U postgres -c 'SELECT 1;'") ]] ; then
      break
    fi
    echo "PostgreSQL is not up yet, retrying..."
    sleep 1
done
