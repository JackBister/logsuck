#!/bin/bash
# A smoke test which compiles the frontend and backend, starts up an in-memory logsuck instance, and makes two API calls to verify that it can read its own log file.
# Assumes that you are running it from the root of the Logsuck git repo. (Meaning this test is invoked as ./test/smoketest.sh)

set -e
set -u
set -x

cd ./internal/web/static/dist
npm ci
npm run build
cd ../../../..
outdir=`mktemp -d`
go build -o "$outdir/logsuck" ./cmd/logsuck/main.go
cd $outdir
timeout 60s ./logsuck -dbfile ":memory:" -webaddr ":8080" log-logsuck.txt > log-logsuck.txt 2>&1 &
sleep 5
curl -XPOST -G 'localhost:8080/api/v1/startJob' --data-urlencode 'searchString=Starting Web GUI'
sleep 1
result=`curl -G 'localhost:8080/api/v1/jobStats' --data-urlencode 'jobId=1'`

if ! grep -q '"NumMatchedEvents":1' <<< "$result"; then
    echo "Expected \$result to contain '\"NumMatchedEvents\":1' but was '$result'"
    exit 1
else
    echo "OK"
fi