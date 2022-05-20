#!/bin/env bash

MACHINE_MAP="
test-dev 192.168.98.250 22 root 1234
local-dev 172.16.40.128 22 anyisalin 12345
"

function init() {
    clear
    echo "$MACHINE_MAP" | sed '/^[[:space:]]*$/d' | awk '{print $1" => "$2}' >&2
    echo ""
    echo ""
}

function main() {
    while true; do

        if [ -z "$NAME" ]; then
            init
            read -rp "which machine do you want login: " NAME
        fi

        if [ -n "$NAME" ]; then
            DATA=($(echo "$MACHINE_MAP" | awk -v "name=$NAME" '($1==name){print $2,$3,$4,$5}'))
            if [ "${#DATA[@]}" -eq 0 ]; then
                echo "can't found machine $NAME, please retry ..." >&2
                continue
            fi

            IP=${DATA[0]}
            PORT=${DATA[1]}
            USER=${DATA[2]}
            PASS=${DATA[3]}

            sshpass -p $PASS ssh -o StrictHostKeyChecking=no -p $PORT $USER@$IP
            NAME=""
        fi
    done
}
main
