package main

import (
	"encoding/json"
	"fmt"
	"github.com/qaqcatz/nanoshlib"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"time"
)

const (
	gDBJsonPath string = "./db.json"
	gDockerDBPath string = "./dockerdb"
	gSudoPasswordPath string = "./sudoPassword.txt"
	gDownloadPath string = "./download"
	gImagePrefix string = "qaqcatz"
	gContainerPrefix string = "qaqcatz"
)

var (
	gSudoPasswordPipe = ""
	gDBMSs []*DBMS = nil
)

func init() {
	if pathExists(gSudoPasswordPath) {
		absPath, err := filepath.Abs(gSudoPasswordPath)
		if err != nil {
			panic("get abs of " + gSudoPasswordPath + " error")
		}
		gSudoPasswordPipe = "cat " + absPath + " | "
	}
	gDBMSs = readDBJson()
}

type DBMS struct {
	Name string `json:"name"`
	User string `json:"user"`
	Password string `json:"password"`
	Port string `json:"port"`
	DefaultDB string `json:"defaultDB"`
	Versions []*Version `json:"versions"`
	versionMap map[string]*Version // version name -> version
}

type Version struct {
	Name string `json:"name"`
	UrlPrefix string `json:"urlPrefix"`
	FileNames []string `json:"fileNames"`
	Meta string `json:"meta"`
	Env string `json:"env"`
}

func readDBJson() []*DBMS {
	data, err := ioutil.ReadFile(gDBJsonPath)
	if err != nil {
		panic("[readDBJson]read json file error: " + err.Error())
	}
	var dbmss []*DBMS
	err = json.Unmarshal(data, &dbmss)
	if err != nil {
		panic("[readDBJson]unmarshal json error: " + err.Error())
	}
	return dbmss
}

func findDBMS(specDbms string) *DBMS {
	for _, dbms := range gDBMSs {
		if dbms.Name == specDbms {
			return dbms
		}
	}
	return nil
}

func (dbms *DBMS) findVersion(specVersion string) *Version {
	if dbms.versionMap == nil {
		dbms.versionMap = make(map[string]*Version)
		for _, version := range dbms.Versions {
			dbms.versionMap[version.Name] = version
		}
	}
	if version, ok := dbms.versionMap[specVersion]; ok {
		return version
	}
	return nil
}

// (1) dbdeployer ls
//
// Show all supported DBMSs, such as mysql, mariadb, ...
//
// (2) dbdeployer ls dbms
//
// Show all supported version under a DBMS.
// For example, if you use `ls mysql`, you will see 5.0.15, 5.0.16, ...
//
// We will collect release versions from the official download page of each DBMS:
//   mysql: https://downloads.mysql.com/archives/community/
//   mariadb: todo
//   tidb: todo
//   oceanbase: todo
//
// (3) dbdeployer run dbms version port
//
// Make sure you have `wget`, `docker`.
//
// We will run a docker container named qaqcatz-port-dbms-version on the specified port
// with user `root`, password `123456` and a default database `qaqcatz`(we will wair for dbms ready).
//
// Note that:
//
//   1. If the container is running, we will do nothing;
//   If the container has exited, we will restart it;
//   If the container does not exist, we will create it through `docker run`;
//   If there is another running container with the prefix `qaqcatz-port`, we will stop it first.
//   2. We will use `sudo` for docker command, make sure your linux user has root privilege.
//   It is recommended to save your sudo password in ./sudoPassword.txt,
//   we will read this file and automatically enter sudo password using pipeline.
//   3. The docker container is created from a docker image named qaqcatz-dbms:version.
//   If it does not exist, we will build it from ./download/dbms/version/Dockerfile.
//   4. Some versions have the same installation process, they should share the same Dockerfile.
//   So we prepared some meta Dockerfiles in ./dockerdb/dbms/metax/.
//   x is the index of meta Dockerfiles.
//   You can fetch the relationship between version and meta in ./db.json.
//   5. Some Dockerfiles have the same environment, we should create a base image for them.
//   So we prepared some environment Dockerfiles in ./dockerdb/dbms/envx,
//   x is the index of env, and the image name is qaqcatz-dbms-env:envx.
//   You can fetch the relationship between meta and env in .db.json.
//   6.	When creating docker images, we will download some necessary files from the official download page
//   and save them in ./download/dbms/version (if not exists).
func main() {
	args := os.Args
	if len(args) <= 1 {
		panic("len(args) <= 1")
	}
	switch args[1] {
	case "ls":
		doLs(args)
	case "run":
		doRun(args)
	default:
		panic("please use ls, run")
	}
}

func doLs(args []string) {
	if len(args) == 2 {
		// ls
		for _, dbms := range gDBMSs {
			fmt.Println(dbms.Name)
		}
	} else if len(args) == 3 {
		specDbms := args[2]
		// ls dbms
		myDbms := findDBMS(specDbms)
		if myDbms == nil {
			panic("[doLs]can not find dbms " + specDbms)
		}
		fmt.Println(len(myDbms.Versions), "versions(old->new):")
		for _, version := range myDbms.Versions {
			fmt.Println(version.Name)
		}
	} else {
		panic("[doLs]please use ls, ls dbms")
	}
}

func doRun(args []string) {
	if len(args) <= 4 {
		panic("[doRun]len(args) <= 4")
	}

	// create logger
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		DisableColors: true,
		FullTimestamp: true,
	})
	writers := []io.Writer{
		os.Stdout,
	}
	multiWriter := io.MultiWriter(writers...)
	logger.SetOutput(multiWriter)
	logger.SetLevel(logrus.InfoLevel)

	// 0. init
	logger.Info("0. init")
	logger.Info("==================================================")
	specDbms := args[2]
	specVersion := args[3]
	hostPort := args[4]
	myDbms := findDBMS(specDbms)
	if myDbms == nil {
		panic("[doRun]can not find dbms " + specDbms)
	}
	user := myDbms.User
	password := myDbms.Password
	defaultDB := myDbms.DefaultDB
	containerPort := myDbms.Port
	myVersion := myDbms.findVersion(specVersion)
	if myVersion == nil {
		panic("[doRun]can not find version " + specVersion + " in " + specDbms)
	}
	containerPrefix := gContainerPrefix+"-"+ hostPort
	containerName := containerPrefix+"-"+specDbms+"-"+specVersion
	imageRepo := gImagePrefix+"-"+specDbms
	imageTag := specVersion

	metaPath := path.Join(gDockerDBPath, specDbms, myVersion.Meta)
	metaDockerfile := path.Join(metaPath, "Dockerfile")

	envImageRepo := gImagePrefix+"-"+specDbms+"-env"
	envImageTag := myVersion.Env
	envPath := path.Join(gDockerDBPath, specDbms, myVersion.Env)

	versionPath := path.Join(gDownloadPath, specDbms, specVersion)
	_ = os.MkdirAll(versionPath, 0777)

	logger.Info("[containerName] ", containerName)
	logger.Info("[imageRepo] ", imageRepo)
	logger.Info("[imageTag] ", imageTag)
	logger.Info("[metaPath] ", metaPath)
	logger.Info("[envImageRepo] ", envImageRepo)
	logger.Info("[envImageTag] ", envImageTag)
	logger.Info("[envPath] ", envPath)
	logger.Info("[versionPath] ", versionPath)
	logger.Info("[user] ", user)
	logger.Info("[password] ", password)
	logger.Info("[hostPort] ", hostPort)
	logger.Info("[containerPort] ", containerPort)

	// 1. download if not exists
	logger.Info("1. download")
	logger.Info("==================================================")
	for _, fileName := range myVersion.FileNames {
		filePath := path.Join(versionPath, fileName)
		if pathExists(filePath) {
			logger.Info(fileName + " already exists")
			continue
		}
		err := nanoshlib.ExecStd("cd " + versionPath + " && wget " + myVersion.UrlPrefix + fileName, -1)
		if err != nil {
			panic("[doRun] download error: " + err.Error())
		}
	}

	// 2. prepare environment if not exists
	logger.Info("2. prepare environment")
	logger.Info("==================================================")
	logger.Info("build ", envImageRepo, ":", envImageTag)
	if !hasImage(envImageRepo, envImageTag) {
		dockerBuild(envPath, envImageRepo, envImageTag)
	} else {
		logger.Info(envImageRepo + ":" + envImageTag + " already exists")
	}

	// 3. prepare meta dockerfile
	logger.Info("3. prepare meta dockerfile")
	logger.Info("==================================================")
	logger.Info("cp " + metaDockerfile + " " + versionPath)
	linuxCP(metaDockerfile, versionPath)

	// 4. build image
	logger.Info("4. build image")
	logger.Info("==================================================")
	logger.Info("build ", imageRepo, ":", imageTag)
	if !hasImage(imageRepo, imageTag) {
		dockerBuild(versionPath, imageRepo, imageTag)
	} else {
		logger.Info(imageRepo + ":" + imageTag + " already exists")
	}

	// 5. run container
	logger.Info("5. run container")
	logger.Info("==================================================")
	status := containerStatus(containerName)
	if status == 1 {
		logger.Info(containerName + " already running")
	} else {
		oldContainer := firstContainerWithPrefix(containerPrefix)
		if oldContainer != "" {
			logger.Info("stop old container " + oldContainer)
			dockerStop(oldContainer)
		}

		if status == 0 {
			logger.Info("restart " + containerName)
			dockerRestart(containerName)
		} else {
			logger.Info("create " + containerName)
			dockerRun(imageRepo, imageTag, containerName, hostPort, containerPort)
		}

		// wair for ready
		ok := false
		logger.Info("wait for ready")
		for try := 1; try <= 16; try += 1 {
			logger.Info("sleep 3s")
			time.Sleep(3*time.Second)
			logger.Info("try ", try)
			err := IsStarted(hostPort, user, password, defaultDB)
			if err == nil {
				ok = true
				break
			} else {
				logger.Info("try error: ", err)
			}
		}
		if ok {
			logger.Info("dbms ready")
		} else {
			panic("start dbms error!")
		}
	}

	logger.Info("Finished!")
}

