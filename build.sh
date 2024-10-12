#!/bin/sh
server_package_path="./cmd/bh-server"
client_package_path="./cmd/bh-client"

server_target_name="bh-server"
client_target_name="bh-client"

bin_dir="./bin"

go build -o "${bin_dir}/${server_target_name}" $server_package_path
go build -o "${bin_dir}/${client_target_name}" $client_package_path
