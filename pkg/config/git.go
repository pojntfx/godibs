package config

import "os"

var (
	GIT_DIR       = os.Getenv("GIT_DIR")
	GIT_NAME      = os.Getenv("GIT_NAME")
	GIT_EMAIL     = os.Getenv("GIT_EMAIL")
	GIT_HTTP_PORT = os.Getenv("GIT_HTTP_PORT")
	GIT_HTTP_PATH = os.Getenv("GIT_HTTP_PATH")
)

const (
	GIT_COMMIT_MESSAGE = "module_synced"
)
