package main

import (
	"fmt"
	"github.com/qaqcatz/nanoshlib"
	"strings"
)

func dockerPull(image string) error {
	cmd := "docker pull " + image
	fmt.Println(cmd)
	err := nanoshlib.ExecStd(cmd, -1)
	if err != nil {
		return err
	}
	return nil
}

// -1: not exists, 0: stop, 1: running
func containerStatus(containerName string) int {
	outStream, errStream, err := nanoshlib.Exec("docker ps " +
		"--format \"{{.Names}}\" --filter \"name=" + containerName + "\"", -1)
	if err != nil {
		panic("[containerStatus] error: " + err.Error() + ": " + errStream)
	}
	lines := strings.Split(strings.TrimSpace(outStream), "\n")
	for _, line := range lines {
		if line == containerName {
			return 1
		}
	}

	outStream, errStream, err = nanoshlib.Exec("docker ps -a " +
		"--format \"{{.Names}}\" --filter \"name=" + containerName + "\"", -1)
	if err != nil {
		panic("[containerStatus] error: " + err.Error() + ": " + errStream)
	}
	lines = strings.Split(strings.TrimSpace(outStream), "\n")
	for _, line := range lines {
		if line == containerName {
			return 0
		}
	}

	return -1
}

// return the first running container (name) with the specified prefix.
// return "" if not exists.
func firstContainerWithPrefix(containerPrefix string) string {
	outStream, errStream, err := nanoshlib.Exec("docker ps " +
		"--format \"{{.Names}}\" --filter \"name=" + containerPrefix + "\"", -1)
	if err != nil {
		panic("[firstContainerWithPrefix] error: " + err.Error() + ": " + errStream)
	}
	lines := strings.Split(strings.TrimSpace(outStream), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, containerPrefix) {
			return line
		}
	}
	return ""
}

func dockerStop(containerName string) {
	cmd := "docker stop " + containerName
	fmt.Println(cmd)
	err := nanoshlib.ExecStd(cmd, -1)
	if err != nil {
		panic("[dockerStop] error: " + err.Error())
	}
}

func dockerRestart(containerName string) {
	cmd := "docker restart " + containerName
	fmt.Println(cmd)
	err := nanoshlib.ExecStd(cmd, -1)
	if err != nil {
		panic("[dockerRestart] error: " + err.Error())
	}
}

func dockerRun(image string, containerName string, hostPort string, containerPort string,
	extra string) {
	cmd := "docker run -itd --name " + containerName + " -p " + hostPort + ":" + containerPort + " " +
		extra + " " + image
	fmt.Println(cmd)
	err := nanoshlib.ExecStd(cmd, -1)
	if err != nil {
		panic("[dockerRun] error: " + err.Error())
	}
}

func dockerExec(containerName string, cmd string) error {
	cmd = "docker exec -t " + containerName + " " + cmd
	fmt.Println(cmd)
	err := nanoshlib.ExecStd(cmd, -1)
	return err
}
