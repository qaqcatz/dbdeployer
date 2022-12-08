package main

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"time"
)

const (
	gDBJsonPath string = "./db.json"
	gContainerPrefix string = "test"
	gUser string = "root" // do not modify
	gPassword string = "123456" // do not modify
	gMaxReadyTry int = 16
)

var (
	gDBMSs []*DBMS = nil
)

func init() {
	gDBMSs = readDBJson()
}

type DBMS struct {
	Name string `json:"name"`
	Port int `json:"port"`
	ExtraDockerRun string `json:"extraDockerRun"`
	WaitForReady string `json:"waitForReady"`
	InitDockerExecs []string `json:"initDockerExecs"`
	Images []*Image `json:"images"`
	imageMap map[string]*Image // image repo:image tag -> *Image
}

type Image struct {
	Repo string `json:"repo"`
	Tag string `json:"tag"`
	ExtraDockerRun string `json:"extraDockerRun"`
	WaitForReady string `json:"waitForReady"`
	InitDockerExecs []string `json:"initDockerExecs"`
	DockerExecs []string `json:"dockerExecs"`
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

func (dbms *DBMS) findImage(specImage string) *Image {
	if dbms.imageMap == nil {
		dbms.imageMap = make(map[string]*Image)
		for _, image := range dbms.Images {
			dbms.imageMap[image.Repo+":"+image.Tag] = image
		}
	}
	if image, ok := dbms.imageMap[specImage]; ok {
		return image
	}
	return nil
}

// (1) dbdeployer ls
//
// Show all supported DBMSs, such as mysql, mariadb, ...
//
// (2) dbdeployer ls dbms
//
// Show all supported docker images under a DBMS (from old to new).
// For example, if you use `ls mysql`, you will see mysql:5.5.40, mysql:5.5.41, ...
//
// We will collect these images from the official dockerhub (mainly) of each DBMS:
//   mysql: https://hub.docker.com/_/mysql/tags
//          https://hub.docker.com/r/vettadock/mysql-old/tags
//   mariadb: todo
//   tidb: todo
//   oceanbase: todo
//
// (3) dbdeployer run dbms imageRepo:imageTag port
//
// Make sure your linux user is in the user group `docker`, see:
// https://askubuntu.com/questions/477551/how-can-i-use-docker-without-sudo
//    `cat /etc/group | grep docker`, if the group `docker` does not exist, `sudo groupadd docker`
//    `sudo gpasswd -a $USER docker`
//	   close the old terminal, start a new one, use `id` to check if your user is in the group `docker`
//     now you can use `docker` without `sudo`.
//
// We will run a docker container named test-port-dbms-imageTag on the specified port,
// with user `root`, password `123456`.
//
// Note that:
//   If the container is running, we will do nothing;
//   If the container has exited, we will restart it;
//   If the container does not exist, we will create it;
//   If there is another running container with the prefix `test-port`, we will stop it first.
//   We will wait for the dbms ready
//
// (4) bisect dbms oldImageRepo:oldImageTag newImageRepo:newImageTag
//
// Return the middle image between oldImage and newImage.
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
	case "bisect":
		doBisect(args)
	default:
		panic("please use ls, run, bisect")
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
		fmt.Println(len(myDbms.Images), "docker images(old->new):")
		for _, image := range myDbms.Images {
			fmt.Println(image.Repo+":"+image.Tag)
		}
	} else {
		panic("[doLs]please use ls, ls dbms")
	}
}

func doRun(args []string) {
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

	logger.Info("1. init")
	logger.Info("==================================================")

	if len(args) <= 4 {
		panic("[doRun]len(args) <= 4")
	}
	specDbms := args[2]
	specImage := args[3]
	specPort := args[4]

	myDbms := findDBMS(specDbms)
	if myDbms == nil {
		panic("[doRun]can not find dbms " + specDbms)
	}
	myImage := myDbms.findImage(specImage)
	if myImage == nil {
		panic("[doRun]can not find image " + specImage + " in " + specDbms)
	}

	extraDockerRun := myDbms.ExtraDockerRun
	if myImage.ExtraDockerRun != "" {
		extraDockerRun = myImage.ExtraDockerRun
	}
	waitForReady := myDbms.WaitForReady
	if myImage.WaitForReady != "" {
		waitForReady = myImage.WaitForReady
	}
	initDockerExecs := myDbms.InitDockerExecs
	if len(myImage.InitDockerExecs) != 0 {
		initDockerExecs = myImage.InitDockerExecs
	}
	containerPort := myDbms.Port
	containerPrefix := gContainerPrefix+"-"+ specPort
	containerName := containerPrefix+"-"+specDbms+"-"+myImage.Tag

	logger.Info("[DBMS] ", specDbms)
	logger.Info("[Image] ", specImage)
	logger.Info("[HostPort] ", specPort)
	logger.Info("[ContainerPort] ", containerPort)
	logger.Info("[ContainerName] ", containerName)
	logger.Info("[User] ", gUser)
	logger.Info("[Password] ", gPassword)

	logger.Info("2. run container")
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
			dockerRun(specImage, containerName, specPort, strconv.Itoa(containerPort), extraDockerRun)
		}

		// wair for ready
		ok := false
		logger.Info("wait for ready")
		for try := 1; try <= gMaxReadyTry; try += 1 {
			logger.Info("sleep 3s")
			time.Sleep(3*time.Second)
			logger.Info("try ", try)
			err := dockerExec(containerName, waitForReady)
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

		if status == -1 {
			for _, initDockerExec := range initDockerExecs {
				err := dockerExec(containerName, initDockerExec)
				if err != nil {
					panic("init dbms error: " + err.Error())
				}
			}
		}
	}

	logger.Info("Finished!")
}

func doBisect(args []string) {
	// bisect dbms oldImageRepo:oldImageTag newImageRepo:newImageTag
	if len(args) <= 4 {
		panic("[doBisect]len(args) <= 4")
	}
	specDbms := args[2]
	specOldImage := args[3]
	specNewImage := args[4]

	myDbms := findDBMS(specDbms)
	if myDbms == nil {
		panic("[doBisect]can not find dbms " + specDbms)
	}

	start := 0
	for ; start < len(myDbms.Images); start += 1 {
		if myDbms.Images[start].Repo+":"+myDbms.Images[start].Tag == specOldImage {
			break
		}
	}
	if start >= len(myDbms.Images) {
		panic("[doBisect]can not find the oldImage " + specOldImage)
	}

	end := start
	for ; end < len(myDbms.Images); end += 1 {
		if myDbms.Images[end].Repo+":"+myDbms.Images[end].Tag == specNewImage {
			break
		}
	}
	if end >= len(myDbms.Images) {
		panic("[doBisect]can not find the newImage " + specNewImage)
	}

	midId := (start+end)/2
	fmt.Println(myDbms.Images[midId].Repo+":"+myDbms.Images[midId].Tag)
}