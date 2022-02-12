#!/bin/bash
# Copyright 2022 Jack Bister
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

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