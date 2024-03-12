#!/bin/bash -xe
if [[ $1 == "sync" ]]; then
    # because Red Hat is Security first the built in containers use SELinux and specific privileges.  rsync needed a few more arguments to work
    ls -alZ /source /dest
    rsync -axHAX -O --progress /source/ /dest
elif [[ $1 == "sleep" ]]; then
    cat
else
    echo "No command given. Make sure to use the correct mover image for your korb version."
    exit 1
fi
