# dbdeployer
Deploy docker containers for DBMSs.

## build

```shell
go build
```

## How to use

### ls

```shell
./dbdeployer ls
```

Show all supported DBMSs, such as mysql, mariadb, ...

```shell
./dbdeployer ls dbms
```

Show all supported docker images under a DBMS (from old to new).

For example, if you use `ls mysql`, you will see mysql:5.5.40, mysql:5.5.41, ...

We will collect these images from the official dockerhub (mainly) of each DBMS:

* mysql:

  https://hub.docker.com/_/mysql/tags

  https://hub.docker.com/r/vettadock/mysql-old/tags

* mariadb: todo

* tidb: todo

* oceanbase: todo

### run

```shell
./dbdeployer run dbms imageRepo:imageTag port
```

Make sure your linux user is in the user group `docker`, see:
https://askubuntu.com/questions/477551/how-can-i-use-docker-without-sudo

We will run a docker container named `test-port-dbms-imageTag` on the specified port, with user `root`, password `123456`.

Note that:

* If the container is running, we will do nothing;
* If the container has exited, we will restart it;
* If the container does not exist, we will create it;
* If there is another running container with the prefix `test-port`, we will stop it first.
* We will wait for the dbms ready

### bisect

```shell
./dbdeployer bisect dbms oldImageRepo:oldImageTag newImageRepo:newImageTag
```

Return the middle image between oldImage and newImage.