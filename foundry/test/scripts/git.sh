#!/usr/bin/env bash

set -eo pipefail

TOKEN=$(cat .token)
echo ">>> Creating git repos"
curl -f -X POST "localhost:3000/api/v1/user/repos" \
  -H "Content-Type: application/json" \
  -H "Authorization: token ${TOKEN}" \
  -d "{
    \"name\": \"deployment\",
    \"private\": false,
    \"description\": \"Deployment repo\"
  }"
curl -f -X POST "localhost:3000/api/v1/user/repos" \
  -H "Content-Type: application/json" \
  -H "Authorization: token ${TOKEN}" \
  -d "{
    \"name\": \"source\",
    \"private\": false,
    \"description\": \"Source repo\"
  }"

echo ">>> Pushing git repos"
if [ -d "./git" ]; then
  rm -rf "./git"
fi
mkdir -p git

cp -r repos/deploy git/deploy
git -C git/deploy init &&
  git -C git/deploy config user.name "root" &&
  git -C git/deploy config user.email "root@example.com" &&
  git -C git/deploy add . &&
  git -C git/deploy commit -m "Initial commit" &&
  git -C git/deploy remote add origin "http://root:root@localhost:3000/root/deployment.git" &&
  git -C git/deploy push -u origin master

cp -r repos/source git/source
mv git/source/blueprint.cue.fake git/source/blueprint.cue
mv git/source/project/blueprint.cue.fake git/source/project/blueprint.cue
git -C git/source init &&
  git -C git/source config user.name "root" &&
  git -C git/source config user.email "root@example.com" &&
  git -C git/source add . &&
  git -C git/source commit -m "Initial commit" &&
  git -C git/source remote add origin "http://root:root@localhost:3000/root/source.git" &&
  git -C git/source push -u origin master
