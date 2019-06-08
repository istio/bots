#!/usr/bin/env sh
set -e

svgo --version

npx svgo -r -f src/icons
