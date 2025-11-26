#!/bin/bash

curl --request DELETE \
     --url http://localhost:23820/databases/{database_name} \
     --header 'accept: application/json' \
     --header 'content-type: application/json' \
     --data '{"drop_option": "ignore_if_not_exists"}' || exit

curl --request POST \
     --url http://localhost:23820/databases/rfs \
     --header 'accept: application/json' \
     --header 'content-type: application/json' \
     --data '{"create_option": "ignore_if_exists"}' || exit

curl --request DELETE \
     --url http://localhost:23820/databases/rfs/tables/data \
     --header 'accept: application/json' \
     --header 'content-type: application/json' \
     --data '{"drop_option": "ignore_if_not_exists"}' || exit

curl --request POST \
     --url http://localhost:23820/databases/rfs/tables/data \
     --header 'accept: application/json' \
     --header 'content-type: application/json' \
     --data '{
         "create_option": "ignore_if_exists",
         "fields": [
             {
                  "name": "name",
                  "type": "varchar",
                  "comment": "name column"
             },
             {
                  "name": "index",
                  "type": "int",
                  "default": 0
             },
             {
                  "name": "dense_column",
                  "type": "vector,384,float",
                  "default": [0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0]
             },
             {
                  "name": "fulltext_column",
                  "type": "varchar",
                  "default": ""
             }
        ]
    }' || exit

curl --request DELETE \
     --url http://localhost:23820/databases/rfs/tables/data/indexes/fts \
     --header 'accept: application/json' \
     --header 'content-type: application/json' \
     --data ' {"drop_option": "ignore_if_not_exists"} '

curl --request POST \
     --url http://localhost:23820/databases/rfs/tables/data/indexes/fts \
     --header 'accept: application/json' \
     --header 'content-type: application/json' \
     --data '
     {
          "fields":
          [
              "fulltext_column"
          ],
          "index":
          {
              "type": "fulltext",
              "analyzer": "standard"
          },
          "create_option": "ignore_if_exists"
     } '

curl --request GET \
     --url http://localhost:23820/databases/rfs/tables/data/columns \
     --header 'accept: application/json'
