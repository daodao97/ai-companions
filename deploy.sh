#!/bin/bash

git pull

docker compose up --remove-orphans --build -d
