#!/bin/bash
trap 'exit' ERR

echo "<h3>Starting the build</h3>"

echo "<h3>Checkout source code and create backup</h3>"
cd /shareddir/
if cd ${PROJECT_REPOSITORY_NAME};
  then git pull --ff-only origin master;
  else git status && git clone -b master ${PROJECT_REPOSITORY_URL} ${PROJECT_REPOSITORY_NAME}
fi
cp /shareddir/backup/bisect.txt ${PROJECT_REPOSITORY_NAME}
git status
git checkout ${PROJECT_BRANCH}
git bundle create ${PROJECT_REPOSITORY_NAME}.bundle --all
cp ${PROJECT_REPOSITORY_NAME}.bundle /shareddir/backup
echo bundle created

echo "<h3>Start git bisect</h3>"
ls -l
: > bisect.txt
cat >> test.sh << EOF
#!/bin/bash
declare -A ary
while IFS="=" read -r key value; do
    ary[$key]=$value
done < "backup/bisect.txt"
if [[ " ${ary[$1]} " =~ "success" ]];
    then exit 1;
    else exit 0;
  fi
EOF

echo good commit: "${GOOD_COMMIT}"

git bisect start
git bisect bad
git bisect good $GOOD_COMMIT
git bisect run bash test.sh $GOOD_COMMIT
FIRST_BAD_COMMIT=$(git bisect log | grep "first bad commit: \[.*\]")
FIRST_BAD_COMMIT=$(echo $FIRST_BAD_COMMIT | grep "\[.*\]")
FIRST_BAD_COMMIT=$(echo $FIRST_BAD_COMMIT | awk -F'[][]' '{print $2}')
echo bisect output: $FIRST_BAD_COMMIT
git bisect reset
echo
git stash
git checkout master
git config --global user.email $EMAIL
git config --global user.name $USER_NAME
git status -s | grep -e "^\?\?" | cut -c 4- >> .gitignore
if [[ -z "$FIRST_BAD_COMMIT" ]]; then

  git revert --no-edit ${PROJECT_BRANCH}
else
  git revert --no-edit $FIRST_BAD_COMMIT^..${PROJECT_BRANCH}
fi
echo start revert
CONFLICTS=$(git ls-files -u | wc -l)
if [ "$CONFLICTS" -gt 0 ] ;
  then echo "There is a conflict. Aborting" && git revert --abort && exit 1 ;
  else echo "Push revert" && git push https://princessfruittt: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@github.com/princessfruittt/example master ;
fi
git push https://${USER_NAME}:${GITHUB_TOKEN}@github.com/princessfruittt/example master ;
echo finish revert
git status
git stash clear
exec "$@"
