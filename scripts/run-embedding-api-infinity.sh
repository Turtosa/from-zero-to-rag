#!/bin/bash

port=7997
model1=michaelfeil/bge-small-en-v1.5
volume=$PWD/embeddings

docker run -it --gpus all \
 -v $volume:/app/.cache \
 -p $port:$port \
 michaelf34/infinity:latest \
 v2 \
 --model-id $model1 \
 --port $port
