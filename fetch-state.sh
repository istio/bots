#!/bin/bash

set -eu

TOKEN=${TOKEN:?pass access token as TOKEN}

# howardjohn test workspace is : 5f299ae6a36edd00126879fb
WORKSPACE=${WORKSPACE:-58d18fe71b06cdd27338c362}

OUT=${OUT:-/tmp/zenhub.csv}
rm -f ${OUT}
for repo in $(curl -s -H "Accept: application/vnd.github.v3+json" https://api.github.com/orgs/istio/repos | jq '.[] | (.id | tostring) + "," +.name' -r | grep -v old); do
  id=$(echo $repo | cut -d, -f1)
  name=$(echo $repo | cut -d, -f2)
  echo "Fetching $name/$id"
  curl -s -H "X-Authentication-Token: ${TOKEN}" -H 'Content-Type: application/json' https://api.zenhub.com/p2/workspaces/${WORKSPACE}/repositories/${id}/board \
    | jq ".pipelines[]? | \"$REPO,\" + .name+\",\"+(.issues[].issue_number | tostring)" -r \
    >> ${OUT}
done
