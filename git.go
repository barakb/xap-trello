package xap_trello

import (
	"os"
	"path"
	"os/exec"
	"bufio"
	"fmt"
	"io"
	"time"
	"log"
	"strings"
	"golang.org/x/net/context"
)

type Git struct {
	local, remote, token string
	log bool
}

func NewGitRepository(local, remote, token string) *Git {
	return &Git{local:local, remote:remote, token:token, log:true}
}

func (git *Git) Init() error {
	if _, err := os.Stat(path.Join(git.local, ".git")); os.IsNotExist(err) {
		return <-git.ExecCmd(1 * time.Second, nil, "init")
	}
	return nil
}

func (git *Git) Log() error {
	ret := <-git.ExecCmd(3 * time.Second, nil, "log")
	return ret
}

func (git *Git) Rebase() error {
	ret := <-git.ExecCmd(3 * time.Second, nil, "pull", "--rebase", "-X", "ours", git.remote)
	return ret
}

func (git *Git) Push() error {
	git.log = false
	defer func() {git.log = true}()
	ret := <-git.ExecCmd(3 * time.Second, nil, "push", git.remoteWithToken(), "master")
	return ret
}


func (git *Git) Add(args... string) error {
	ret := <-git.ExecCmd(3 * time.Second, nil, append([]string{"add"}, args...)...)
	return ret
}

func (git *Git) Commit(message string) error {
	ret := <-git.ExecCmd(3 * time.Second, nil, "commit", "-m", message)
	return ret
}

func (git *Git) remoteWithToken() string {
	remote := git.remote
	index := strings.Index(git.remote, "//")
	if -1 < index{
		remote = fmt.Sprintf("%s//%s@%s", git.remote[:index], git.token, git.remote[index + 2:])
	}
	return remote
}

func (git *Git) ExecCmd(timeout time.Duration, withOutput io.Writer, arg ...string) chan error {
	prompt := strings.Join(arg, " ")
	showLog := git.log
	if showLog {
		log.Printf("Executing 'git %v', timeout is: %s\n", prompt, timeout)
	}else{
		prompt = "*** > "
	}
	res := make(chan error, 1)
	ctx, _ := context.WithTimeout(context.Background(), timeout)
	cmd := exec.CommandContext(ctx, "git", arg...)
	cmd.Dir = git.local

	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		res <- err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		res <- err
	}

	printStdout(showLog, prompt, stdout, withOutput)
	printStderr(showLog, prompt, stderr)
	if err := cmd.Start(); err != nil {
		res <- err
	}
	go func() {
		if err := cmd.Wait(); err != nil {
			res <- err
		} else {
			res <- nil
		}
		close(res)
	}()

	return res
}

func printStdout(showLog bool, prompt string, reader io.ReadCloser, writer  io.Writer) {
	printReader(showLog, fmt.Sprintf("stdout:%s", prompt), reader, writer)
}

func printStderr(showLog bool, prompt string, reader io.ReadCloser) {
	printReader(showLog, fmt.Sprintf("stderr:%s", prompt), reader, nil)
}
func printReader(showLog bool, prompt string, reader io.ReadCloser, writer io.Writer) {
	go func() {
		defer reader.Close()
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			text := scanner.Text()
			if showLog {
				fmt.Printf("%s: %s\n", prompt, text)
			}
			if writer != nil {
				writer.Write([]byte(fmt.Sprintln(text)))
			}
		}
	}()
}

