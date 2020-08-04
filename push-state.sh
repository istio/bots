#!/bin/bash

set -eu

TOKEN=${TOKEN:?pass access token as TOKEN}

# Can be orgs or users
TYPE=${TYPE:-orgs}

ORG=${ORG:-istio}

INPUT=${INPUT:-/tmp/zenhub.csv}

project=$(curl -s -H "Accept: application/vnd.github.inertia-preview+json" -H "Authorization: Bearer $TOKEN" https://api.github.com/$TYPE/$ORG/projects | jq '.[].id')
columns="$(curl -s -H "Accept: application/vnd.github.inertia-preview+json" -H "Authorization: Bearer $TOKEN" https://api.github.com/projects/${project}/columns | jq '.[]? | .name + "," + (.id | tostring)' -r)"


IFS=$'\n'
for line in $(cat ${INPUT} | grep -v "New Issues"); do
  repo=$(echo $line | cut -d, -f1)
  priority=$(echo $line | cut -d, -f2)
  issueNum=$(echo $line | cut -d, -f3)
  column=$(echo -$columns | tr ' ' '\n' | grep $priority | cut -d, -f2)
  issueId=$(curl -s -H "Accept: application/vnd.github.inertia-preview+json" -H "Authorization: Bearer $TOKEN" https://api.github.com/repos/$ORG/$repo/issues/$issueNum | jq '.id')
  echo "Process $repo/$issueNum as $priority"
  curl -s -X POST -H "Accept: application/vnd.github.inertia-preview+json" \
    -H "Authorization: Bearer $TOKEN" \
    https://api.github.com/projects/columns/${column}/cards -d '{"content_type":"Issue", "content_id":'$issueId'}' > /dev/null
done
