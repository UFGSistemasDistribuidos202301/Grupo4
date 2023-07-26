#!/bin/bash

go run . -id 1 -baseNodeID 1 -nodeCount 4 &
go run . -id 2 -baseNodeID 1 -nodeCount 4 &
go run . -id 3 -baseNodeID 1 -nodeCount 4 &
go run . -id 4 -baseNodeID 1 -nodeCount 4 &

for job in `jobs -p`
do
    echo $job
    wait $job || let "FAIL+=1"
done
