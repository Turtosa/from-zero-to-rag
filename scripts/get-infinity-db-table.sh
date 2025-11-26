#!/bin/bash

curl --request GET \
     --url http://localhost:23820/databases/rfs/table/data \
     --header 'accept: application/json' \
     --header 'content-type: application/json' \
     --data ' {
         "file_path": "/var/infinity/export.csv",
         "file_type": "csv",
         "header": false,
         "delimiter": "\t"
    }'

cat /var/infinity/export.csv || exit
rm /var/infinity/export.csv -f
