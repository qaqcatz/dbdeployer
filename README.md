# dbdeployer
Deploy each version of mysql as docker container for testing.

## build

```shell
go build
```

## How to use

### ls

Show all supported DBMSs, such as mysql, mariadb, ...

### ls dbms

Show all supported version under a DBMS.

For example, if you use `ls mysql`, you will see 5.0.15, 5.0.16, ...

We will collect release versions from the official download page of each DBMS:

* mysql: https://downloads.mysql.com/archives/community/
* mariadb: todo
* tidb: todo
* oceanbase: todo

### run dbms version port

Make sure you have `wget`, `docker`.

We will run a docker container named `qaqcatz-port-dbms-version` on the specified port
with user `root`, password `123456` and a default database `qaqcatz`(we will wait for dbms ready).

Note that:

  1. If the container is running, we will do nothing;
    If the container has exited, we will restart it;
  If the container does not exist, we will create it through `docker run`;
  If there is another running container with the prefix `qaqcatz-port`, we will stop it first.
  2. We will use `sudo` for docker command, make sure your linux user has root privilege.
    It is recommended to save your sudo password in `./sudoPassword.txt`,
  we will read this file and automatically enter sudo password using pipeline.
  3. The docker container is created from a docker image named `qaqcatz-dbms:version`.
    If it does not exist, we will build it from `./download/dbms/version/Dockerfile`.
  4. Some versions have the same installation process, they should share the same Dockerfile.
    So we prepared some meta Dockerfiles in `./dockerdb/dbms/metax/`.
  x is the index of meta Dockerfiles.
  You can fetch the relationship between version and meta in `./db.json`.
  5. Some Dockerfiles have the same environment, we should create a base image for them.
    So we prepared some environment Dockerfiles in `./dockerdb/dbms/envx`,
  x is the index of env, and the image name is `qaqcatz-dbms-env:envx`.
  You can fetch the relationship between meta and env in `.db.json`.
  6.   When creating docker images, we will download some necessary files from the official download page
  and save them in `./download/dbms/version` (if not exists).