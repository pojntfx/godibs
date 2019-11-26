package main

import (
	"fmt"
	fswatch "github.com/andreaskoch/go-fswatch"
	"github.com/go-redis/redis/v7"
	"github.com/plus3it/gorecurcopy"
	git "gopkg.in/src-d/go-git.v4"
	gitconf "gopkg.in/src-d/go-git.v4/config"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var (
	REDIS_URL                   = os.Getenv("REDIS_URL")
	REDIS_CHANNEL_PREFIX        = os.Getenv("REDIS_CHANNEL_PREFIX")
	GIT_URL                     = os.Getenv("GIT_URL")
	GIT_REMOTE_NAME             = os.Getenv("GIT_REMOTE_NAME")
	GIT_NAME                    = os.Getenv("GIT_NAME")
	GIT_EMAIL                   = os.Getenv("GIT_EMAIL")
	SRC_DIR                     = os.Getenv("SRC_DIR")
	PUSH_DIR                    = os.Getenv("PUSH_DIR")
	SYNC_MODULE_PUSH_MOD_FILE   = os.Getenv("SYNC_MODULE_PUSH_MOD_FILE")
	SYNC_MODULE_PUSH_WATCH_GLOB = os.Getenv("SYNC_MODULE_PUSH_WATCH_GLOB")
	COMMAND_BUILD               = os.Getenv("COMMAND_BUILD")
	COMMAND_TEST                = os.Getenv("COMMAND_TEST")
)

func main() {
	f, err := ioutil.ReadFile(SYNC_MODULE_PUSH_MOD_FILE)
	if err != nil {
		panic(err)
	}

	var m string

	for _, line := range strings.Split(string(f), "\n") {
		if strings.Contains(line, "module") {
			m = strings.Split(line, "module ")[1]
			break
		}
	}

	r := redis.NewClient(&redis.Options{
		Addr: REDIS_URL,
	})
	defer r.Publish(REDIS_CHANNEL_PREFIX+":"+"module_unregistered", withTimestamp(m))
	r.Publish(REDIS_CHANNEL_PREFIX+":"+"module_registered", withTimestamp(m))

	w := fswatch.NewFolderWatcher(SYNC_MODULE_PUSH_WATCH_GLOB, true, func(path string) bool { return strings.Contains(path, PUSH_DIR) }, 1)
	w.Start()

	for w.IsRunning() {
		select {
		case <-w.ChangeDetails():
			if _, err := os.Stat(PUSH_DIR); !os.IsNotExist(err) {
				os.RemoveAll(PUSH_DIR)
			}

			gorecurcopy.CopyDirectory(SRC_DIR, PUSH_DIR)

			commandTest := exec.Command(strings.Split(COMMAND_TEST, " ")[0], strings.Split(COMMAND_TEST, " ")[1:]...)
			commandTest.Stdout = os.Stdout
			commandTest.Stderr = os.Stderr
			err = commandTest.Run()
			if err != nil {
				panic(err)
			}
			r.Publish(REDIS_CHANNEL_PREFIX+":"+"module_tested", withTimestamp(m))

			commandBuild := exec.Command(strings.Split(COMMAND_BUILD, " ")[0], strings.Split(COMMAND_BUILD, " ")[1:]...)
			commandBuild.Stdout = os.Stdout
			commandBuild.Stderr = os.Stderr
			err = commandBuild.Run()
			if err != nil {
				panic(err)
			}
			r.Publish(REDIS_CHANNEL_PREFIX+":"+"module_built", withTimestamp(m))

			g, err := git.PlainOpen(filepath.Join(PUSH_DIR))
			if err != nil {
				panic(err)
			}

			g.CreateRemote(&gitconf.RemoteConfig{
				Name: GIT_REMOTE_NAME,
				URLs: []string{GIT_URL},
			})

			wt, err := g.Worktree()
			if err != nil {
				panic(err)
			}
			wt.Add(".")

			wt.Commit(withTimestamp("module_synced"), &git.CommitOptions{
				Author: &object.Signature{
					Name:  GIT_NAME,
					Email: GIT_EMAIL,
					When:  time.Now(),
				},
			})

			err = g.Push(&git.PushOptions{
				RemoteName: GIT_REMOTE_NAME,
				RefSpecs:   []gitconf.RefSpec{"+refs/heads/master:refs/heads/branch"},
			})
			if err != nil {
				panic(err)
			}

			r.Publish(REDIS_CHANNEL_PREFIX+":"+"module_pushed", withTimestamp(m))

			fmt.Println(m)
		}
	}
}

func withTimestamp(m string) string {
	t := time.Now().UnixNano()
	return m + "@" + strconv.Itoa(int(t))
}
