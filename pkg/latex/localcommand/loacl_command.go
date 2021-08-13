package localcommand

import (
	"bytes"
	"fmt"
	xj "github.com/basgys/goxml2json"
	"os"
	"os/exec"
	"strings"
)

const bash = "/bin/bash"

type LocalCommand struct {
	Command string
	Args    []string
}

func New(command string, args []string) *LocalCommand {
	return &LocalCommand{
		Command: command,
		Args:    args,
	}
}

func (l LocalCommand) Run() (result string, err error) {
	var args []string
	execCmd := l.generateCmd()
	args = append(args, "-c", execCmd)
	command := exec.Command(bash, args...)
	var stdout, stderr bytes.Buffer
	command.Stderr = &stderr
	command.Stdout = &stdout
	if e := command.Run(); e != nil || len(stderr.String()) > 0 {
		err = fmt.Errorf("exec %s failed, err:%s, stderr:%s", execCmd, e.Error(), stderr.String())
		return
	}
	result = stdout.String()
	return
}

func (l LocalCommand) OutStdErrRun() (result, outErr string, err error) {
	var args []string
	execCmd := l.generateCmd()
	args = append(args, "-c", execCmd)
	command := exec.Command(bash, args...)
	var stdout, stderr bytes.Buffer
	command.Stderr = &stderr
	command.Stdout = &stdout
	if e := command.Run(); e != nil  {
		outErr = stderr.String()
		err = fmt.Errorf("exec %s failed, err:%s, stderr:%s", execCmd, e.Error(), stderr.String())
		return
	}
	result = stdout.String()
	return
}

func (l LocalCommand) generateCmd() (cmd string) {
	args := strings.Join(l.Args, " ")
	return l.Command + " " + args
}

func CreateDir(dir string) (err error) {
	cmd := New("mkdir", []string{"-p", dir})
	_, err = cmd.Run()
	return
}

func CreateFile(path string) (err error) {
	cmd := New("touch", []string{path})
	_, err = cmd.Run()
	return
}

func CopyDir(sourceDir, targetDir string) (err error) {
	if strings.HasSuffix(sourceDir, "/") {
		sourceDir += "*"
	} else {
		sourceDir += "/*"
	}
	cmd := New("cp", []string{"-r", sourceDir, targetDir})
	_, err = cmd.Run()
	return
}

func DirTree(dir string) (nodes xj.Nodes, err error) {
	cmd := New("cd", []string{dir, "&&", "tree -X"})
	treeXml, err := cmd.Run()
	r := strings.NewReader(treeXml)
	root := &xj.Node{}
	err = xj.NewDecoder(r).Decode(root)
	if err != nil {
		return
	}
	nodes = root.Children["tree"]
	return
}

func Rm(path string) (err error) {
	//cmd := New("rm", []string{"-r", path})
	//_, err = cmd.Run()
	return os.RemoveAll(path)
}