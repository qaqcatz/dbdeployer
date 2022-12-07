#!/bin/bash

echo "waiting for dbms ready"

for (( i=1; i <= 16; i++ ))
do
  echo "sleep 3s"
  sleep 3s
  echo "try "$i
  echo "USE mysql; SELECT 1" | mysql -u root
  if [ $? -eq 0 ]; then
    echo "dbms ready"
    echo "CREATE USER 'root'@'%' IDENTIFIED BY '123456';"
    echo "GRANT ALL PRIVILEGES ON *.* TO 'root'@'%' WITH GRANT OPTION;"
    echo "CREATE DATABASE IF NOT EXISTS qaqcatz;"
    echo "CREATE USER 'root'@'%' IDENTIFIED BY '123456'; GRANT ALL PRIVILEGES ON *.* TO 'root'@'%' WITH GRANT OPTION; CREATE DATABASE IF NOT EXISTS qaqcatz; " | mysql -u root
    if [ $? -eq 0 ]; then
      echo "init dbms successfully"
      exit 0
    else
      echo "init dbms error!"
      exit 1
    fi
  fi
done

echo "start dbms error!"
exit 1