#!/bin/bash
trap 'exit' ERR
cp -R /tmp/.ssh /root/.ssh
chmod 700 /root/.ssh
chmod 644 /root/.ssh/id_rsa.pub
chmod 600 /root/.ssh/id_rsa
ssh-keyscan github.com > /root/.ssh/known_hosts
echo

echo "<h3>Checkout source code</h3>"
cd /shareddir/
if [ -d "$PROJECT_REPOSITORY_NAME" ];
  then cd "${PROJECT_REPOSITORY_NAME}" && git checkout master && git pull --all;
  else git clone -b master ${PROJECT_REPOSITORY_URL} ${PROJECT_REPOSITORY_NAME} && cd ${PROJECT_REPOSITORY_NAME};
fi
git stash clear
git checkout ${PROJECT_BRANCH}
echo hi
git stash
cd .
git stash clear
echo

echo "<h3>Setup</h3>"
echo "stage 1: go mod download"
echo go mod download
echo "stage 2: go mod build"
go build ./...

echo "<h3>Test</h3>"
go test ./...

exec "$@"
