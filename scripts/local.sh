#!/bin/bash

set -e -o pipefail
trap '[ "$?" -eq 0 ] || echo "Error Line:<$LINENO> Error Function:<${FUNCNAME}>"' EXIT

cd `dirname $0` && cd ..
CURRENT=`pwd`


function test
{
  go test -v $(go list ./... | grep -v vendor) --count 1 -covermode=atomic -timeout 6000s
}

function bench_limiter
{
   docker kill limit-redis-server || true

   docker run --rm -d --name limit-redis-server -p 6379:6379 redis:5.0 || true
   go test github.com/gjbae1212/go-limiter -run none -bench ^BenchmarkRedisLimiter_Acquire$ -benchtime 300s -benchmem
   go test github.com/gjbae1212/go-limiter -run none -bench ^BenchmarkMemoryLimiter_Acquire$ -benchtime 300s -benchmem
}


CMD=$1
shift
$CMD $*
