package main

import (
	"github.com/qaqcatz/nanoshlib"
	"strings"
)

func hasImage(repository string, tag string) bool {
	outSteram, errStream, err := nanoshlib.Exec(gSudoPasswordPipe + "sudo -S docker images " +
		"--format \"{{.Repository}}:{{.Tag}}\" " + repository + ":" + tag, -1)
	if err != nil {
		panic("[hasImage] error: " + err.Error() + ": " + errStream)
	}
	lines := strings.Split(strings.TrimSpace(outSteram), "\n")
	for _, line := range lines {
		sp := strings.SplitN(line, ":", 2)
		if len(sp) == 2 && sp[0] == repository && sp[1] == tag {
			return true
		}
	}
	return false
}

func dockerBuild(buildPath string, imageRepo string, imageTag string) {
	err := nanoshlib.ExecStd("cd " + buildPath + " && " +
		gSudoPasswordPipe + "sudo -S docker build -t " + imageRepo + ":" + imageTag + " . ", -1)
	if err != nil {
		panic("[dockerBuild] build error: " + err.Error())
	}
}

// -1: not exists, 0: stop, 1: running
func containerStatus(containerName string) int {
	outStream, errStream, err := nanoshlib.Exec(gSudoPasswordPipe + "sudo -S docker ps " +
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

	outStream, errStream, err = nanoshlib.Exec(gSudoPasswordPipe + "sudo -S docker ps -a " +
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
	outStream, errStream, err := nanoshlib.Exec(gSudoPasswordPipe + "sudo -S docker ps " +
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
	err := nanoshlib.ExecStd(gSudoPasswordPipe + "sudo -S docker stop " + containerName, -1)
	if err != nil {
		panic("[dockerStop] error: " + err.Error())
	}
}

func dockerRestart(containerName string) {
	err := nanoshlib.ExecStd(gSudoPasswordPipe + "sudo -S docker restart " + containerName, -1)
	if err != nil {
		panic("[dockerRestart] error: " + err.Error())
	}
}

func dockerRun(imageRepo string, imageTag string, containerName string, hostPort string, containerPort string) {
	err := nanoshlib.ExecStd(gSudoPasswordPipe + "sudo -S docker run -itd --name " + containerName +
		" -p " + hostPort + ":" + containerPort + " --privileged=true " +
		imageRepo + ":" + imageTag, -1)
	if err != nil {
		panic("[dockerBuild] build error: " + err.Error())
	}
}