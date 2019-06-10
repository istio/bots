#!/bin/bash

FAILED=0

echo -ne "sass-lint "
npx sass-lint --version
echo -ne "tslint "
npx tslint  --version

npx sass-lint -c sass-lint.yml --verbose 'src/sass/**/*.scss'
npx tslint src/ts/*.ts

if [[ ${FAILED} -eq 1 ]]
then
    echo "LINTING FAILED"
    exit 1
fi
