package main

import (
	"errors"
	fswatch "github.com/andreaskoch/go-fswatch"
	"github.com/go-redis/redis/v7"
	"github.com/plus3it/gorecurcopy"
	"gitlab.com/z0mbie42/rz-go/v2/log"
	git "gopkg.in/src-d/go-git.v4"
	gitconf "gopkg.in/src-d/go-git.v4/config"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
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
	COMMAND_START               = os.Getenv("COMMAND_START")
)

func main() {
	err, m := getModuleName(SYNC_MODULE_PUSH_MOD_FILE)
	if err != nil {
		panic(err)
	}

	r := getNewRedisClient(REDIS_URL)

	log.Info("Registering module ...")
	registerModule(r, REDIS_CHANNEL_PREFIX, m)

	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		unregisterModule(r, REDIS_CHANNEL_PREFIX, m)
		os.Exit(0)
	}()

	w := getNewFolderWatcher(SYNC_MODULE_PUSH_WATCH_GLOB, PUSH_DIR)

	first := make(chan struct{}, 1)
	first <- struct{}{}
	var commandStart *exec.Cmd

	for w.IsRunning() {
		select {
		case <-first:
		case <-w.ChangeDetails():
			if commandStart != nil {
				commandStart.Process.Kill()
			}

			setupPushDir(SRC_DIR, PUSH_DIR)

			log.Info("Building module ...")
			err := runCommand(r, REDIS_CHANNEL_PREFIX, "module_built", m, COMMAND_BUILD, false)
			if err != nil {
				panic(err)
			}

			log.Info("Testing module ...")
			err = runCommand(r, REDIS_CHANNEL_PREFIX, "module_tested", m, COMMAND_TEST, false)
			if err != nil {
				panic(err)
			}

			log.Info("Pushing module ...")
			err = pushModule(r, REDIS_CHANNEL_PREFIX, m, PUSH_DIR, GIT_REMOTE_NAME, GIT_URL, GIT_NAME, GIT_EMAIL)
			if err != nil {
				panic(err)
			}

			log.Info("Starting module ...")
			err = runCommand(r, REDIS_CHANNEL_PREFIX, "module_started", m, COMMAND_START, true)
			if err != nil {
				panic(err)
			}
		}
	}
}

func getModuleName(goModFilePath string) (error, string) {
	f, err := ioutil.ReadFile(goModFilePath)
	if err != nil {
		return errors.New("Could not read module file"), ""
	}

	for _, line := range strings.Split(string(f), "\n") {
		if strings.Contains(line, "module") {
			return nil, strings.Split(line, "module ")[1]
		}
	}

	return errors.New("Could find module declaration"), ""
}

func getNewRedisClient(addr string) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr: addr,
	})
}

func getNewFolderWatcher(watchGlob, pushDir string) *fswatch.FolderWatcher {
	w := fswatch.NewFolderWatcher(watchGlob, true, func(path string) bool { return strings.Contains(path, pushDir) }, 1)
	w.Start()

	return w
}

func registerModule(r *redis.Client, prefix, m string) {
	r.Publish(prefix+":"+"module_registered", withTimestamp(m))
}

func unregisterModule(r *redis.Client, prefix, m string) {
	r.Publish(prefix+":"+"module_unregistered", withTimestamp(m))
}

func runCommand(r *redis.Client, prefix, suffix, m, command string, start bool) error {
	c := exec.Command(strings.Split(command, " ")[0], strings.Split(command, " ")[1:]...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	var err error
	if start {
		err = c.Start()
	} else {
		err = c.Run()
	}
	if err != nil {
		return err
	}
	r.Publish(prefix+":"+suffix, withTimestamp(m))
	return nil
}

func setupPushDir(srcDir, pushDir string) {
	if _, err := os.Stat(pushDir); !os.IsNotExist(err) {
		os.RemoveAll(pushDir)
	}
	gorecurcopy.CopyDirectory(srcDir, pushDir)
}

func pushModule(r *redis.Client, prefix, m, pushDir, gitRemoteName, gitUrl, gitName, gitEmail string) error {
	g, err := git.PlainOpen(filepath.Join(pushDir))
	if err != nil {
		return err
	}

	g.CreateRemote(&gitconf.RemoteConfig{
		Name: gitRemoteName,
		URLs: []string{gitUrl},
	})

	wt, err := g.Worktree()
	if err != nil {
		return err
	}
	wt.Add(".")

	wt.Commit(withTimestamp("module_synced"), &git.CommitOptions{
		Author: &object.Signature{
			Name:  gitName,
			Email: gitEmail,
			When:  time.Now(),
		},
	})

	err = g.Push(&git.PushOptions{
		RemoteName: gitRemoteName,
		RefSpecs:   []gitconf.RefSpec{"+refs/heads/master:refs/heads/master"},
	})
	if err != nil {
		return err
	}

	r.Publish(prefix+":"+"module_pushed", withTimestamp(m))

	return nil
}

func withTimestamp(m string) string {
	t := time.Now().UnixNano()
	return m + "@" + strconv.Itoa(int(t))
}
