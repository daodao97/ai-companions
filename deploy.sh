#!/bin/bash

git pull

# 确保数据库文件存在，避免 Docker 将其创建为文件夹
if [ ! -f "./companion.db" ]; then
    echo "Creating companion.db file..."
    touch ./companion.db
    chmod 644 ./companion.db
fi

docker compose up --remove-orphans --build -d
