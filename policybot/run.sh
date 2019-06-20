#!/bin/sh

if [[ -f /env/runenv.sh ]];then
  source /env/runenv.sh
else
  echo "No env specified"
fi
/policybot server --configFile ./policybot.yaml "$@"
