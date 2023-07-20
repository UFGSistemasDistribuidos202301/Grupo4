#!/bin/bash

go run .. -id 1 &
go run .. -id 2 &
go run .. -id 3 &
go run .. -id 4 &

for job in `jobs -p`
do
    echo $job
    wait $job || let "FAIL+=1"
done
