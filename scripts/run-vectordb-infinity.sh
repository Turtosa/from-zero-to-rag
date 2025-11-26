#!/bin/bash

docker run --name infinity-vectordb -v /var/infinity/:/var/infinity --ulimit nofile=500000:500000 --network=host infiniflow/infinity:nightly
