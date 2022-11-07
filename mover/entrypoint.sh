#!/bin/bash -xe
if [[ $1 == "sync" ]]; then
    rsync -aHA --progress /source/ /dest
elif [[ $1 == "sleep" ]]; then
    cat
else
    echo "No command given. Make sure to use the correct mover image for your korb version."
    exit 1
fi
