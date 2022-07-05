#!/bin/bash -xe
if [[ $1 == "sync" ]]; then
    rsync -aHA --progress /source/ /dest
elif [[ $1 == "sleep" ]]; then
    sleep infinity
fi
