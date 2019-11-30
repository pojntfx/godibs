package main

import (
	redis "github.com/go-redis/redis/v7"
	"github.com/gorilla/mux"
	"github.com/pojntfx/godibs/src/lib/common"
	rz "gitlab.com/z0mbie42/rz-go/v2"
	"gitlab.com/z0mbie42/rz-go/v2/log"
	"gopkg.in/mysticmode/gitviahttp.v1"
	git "gopkg.in/src-d/go-git.v4"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

var (
	REDIS_URL            = os.Getenv("REDIS_URL")
	REDIS_CHANNEL_PREFIX = os.Getenv("REDIS_CHANNEL_PREFIX")
	GIT_DIR              = os.Getenv("GIT_DIR")
	GIT_NAME             = os.Getenv("GIT_NAME")
	GIT_EMAIL            = os.Getenv("GIT_EMAIL")
	GIT_HTTP_PORT        = os.Getenv("GIT_HTTP_PORT")
	GIT_HTTP_PATH        = os.Getenv("GIT_HTTP_PATH")
)

func main() {
	r := common.GetNewRedisClient(REDIS_URL)
	p, err := strconv.ParseInt(GIT_HTTP_PORT, 0, 64)
	if err != nil {
		panic(err)
	}

	var wg sync.WaitGroup

	wg.Add(4)

	go StartDirectoryManagementWorker(&wg, r, REDIS_CHANNEL_PREFIX, common.REDIS_CHANNEL_MODULE_REGISTERED, GIT_DIR, false)
	go StartDirectoryManagementWorker(&wg, r, REDIS_CHANNEL_PREFIX, common.REDIS_CHANNEL_MODULE_UNREGISTERED, GIT_DIR, true)
	go StartHTTPWorker(&wg, GIT_DIR, int(p), GIT_HTTP_PATH)

	wg.Wait()
}

// StartHTTPWorker starts a new HTTP server to serve the git repos from
func StartHTTPWorker(wg *sync.WaitGroup, baseDir string, port int, path string) {
	log.Info("Starting HTTP worker ...")

	r := mux.NewRouter()

	r.PathPrefix(path).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gitviahttp.Context(w, r, filepath.Join(baseDir, "repositories"))
	}).Methods("GET", "POST")

	s := &http.Server{
		Handler: r,
		Addr:    "127.0.0.1:" + strconv.Itoa(port),
	}

	s.ListenAndServe()

	wg.Done()
}

// parseModuleFromMessage gets the module name and event timestamp from a message
func parseModuleFromMessage(m string) (name, timestamp string) {
	res := strings.Split(m, "@")
	return res[0], res[1]
}

// getPathForModule builds the path for a module
func getPathForModule(baseDir, m string) string {
	return filepath.Join(append([]string{baseDir, "repositories"}, strings.Split(m, "/")...)...)
}

// getChannel gets a new Go channel for a redis prefix and channel
func getChannel(r *redis.Client, prefix, channel string) (error, <-chan *redis.Message, *redis.PubSub) {
	p := r.Subscribe(prefix + ":" + channel)

	_, err := p.Receive()
	if err != nil {
		return err, nil, p
	}

	return nil, p.Channel(), p
}

// StartDirectoryManagementWorker starts a new directory management worker
func StartDirectoryManagementWorker(wg *sync.WaitGroup, r *redis.Client, prefix, channel, baseDir string, deleteOnly bool) error {
	err, c, p := getChannel(r, prefix, channel)
	defer p.Close()
	if err != nil {
		return err
	}

	if deleteOnly {
		log.Info("Starting directory deletion worker ...")
	} else {
		log.Info("Starting directory update worker ...")
	}

	for m := range c {
		var innerWg sync.WaitGroup

		go func(wg *sync.WaitGroup, msg *redis.Message) {
			wg.Add(1)

			n, t := parseModuleFromMessage(msg.Payload)
			if deleteOnly {
				log.Info("Deleting directory", rz.String("moduleName", n), rz.String("eventTimestamp", t))
			} else {
				log.Info("Updating directory", rz.String("moduleName", n), rz.String("eventTimestamp", t))
			}

			path := getPathForModule(baseDir, n)

			if !deleteOnly {
				err = os.RemoveAll(path)
				if err != nil {
					panic(err)
				}

				err = os.MkdirAll(path, 0777)
				if err != nil {
					panic(err)
				}

				_, err := git.PlainInit(path, false)
				if err != nil {
					panic(err)
				}
			}

			defer wg.Done()
		}(&innerWg, m)
	}

	defer wg.Done()

	return nil
}
