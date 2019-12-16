package cmd

import (
	"github.com/google/uuid"
	"github.com/pojntfx/dibs/pkg/starters"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gitlab.com/z0mbie42/rz-go/v2"
	"gitlab.com/z0mbie42/rz-go/v2/log"
	"os"
	"path/filepath"
	"strings"
)

var PipelineSyncClientCmd = &cobra.Command{
	Use:   "client",
	Short: "Start the module development client",
	Run: func(cmd *cobra.Command, args []string) {
		switch Lang {
		case LangGo:
			// Ignore if there are errors here, platforms might not be set (there is no hard dependency on the config)
			platforms, _ := Dibs.GetPlatforms(Platform, Platform == PlatformAll)
			ignoreRegex := IgnoreRegexPlaceholder
			if len(platforms) > 0 {
				ignoreRegex = platforms[0].Assets.CleanGlob
			}

			client := starters.Client{
				PipelineUpFileMod:      viper.GetString(GoPipelineUpFileModKey),
				PipelineDownModules:    viper.GetString(GoPipelineDownModulesKey),
				PipelineDownDirModules: viper.GetString(GoPipelineDownDirModulesKey),
				PipelineUpBuildCommand: strings.Replace(viper.GetString(PipelineUpBuildCommandKey), PlatformPlaceholder, Platform, -1),
				PipelineUpStartCommand: strings.Replace(viper.GetString(PipelineUpStartCommandKey), PlatformPlaceholder, Platform, -1),
				PipelineUpTestCommand:  strings.Replace(viper.GetString(PipelineUpTestCommandKey), PlatformPlaceholder, Platform, -1),
				PipelineUpDirSrc:       viper.GetString(PipelineUpDirSrcKey),
				PipelineUpDirPush:      viper.GetString(PipelineUpDirPushKey),
				PipelineUpDirWatch:     viper.GetString(PipelineUpDirWatchKey),
				PipelineUpRegexIgnore:  strings.Replace(viper.GetString(PipelineUpRegexIgnoreKey), IgnoreRegexPlaceholder, ignoreRegex, -1),

				RedisUrl:                  RedisUrl,
				RedisPrefix:               RedisPrefix,
				RedisSuffixUpRegistered:   RedisSuffixUpRegistered,
				RedisSuffixUpUnRegistered: RedisSuffixUpUnregistered,
				RedisSuffixUpTested:       RedisSuffixUpTested,
				RedisSuffixUpBuilt:        RedisSuffixUpBuilt,
				RedisSuffixUpStarted:      RedisSuffixUpStarted,
				RedisSuffixUpPushed:       RedisSuffixUpPushed,

				GitUpRemoteName:    GitUpRemoteName,
				GitUpBaseURL:       viper.GetString(GoGitBaseUrlKey),
				GitUpUserName:      GitUpUserName,
				GitUpUserEmail:     GitUpUserEmail,
				GitUpCommitMessage: GitUpCommitMessage,
			}

			client.Start()
		}
	},
}

var (
	GoGitUpBaseUrl string

	PipelineUpDirSrc   string
	PipelineUpDirPush  string
	PipelineUpDirWatch string

	GoPipelineUpFileMod string

	PipelineUpBuildCommand string
	PipelineUpTestCommand  string
	PipelineUpStartCommand string

	PipelineUpRegexIgnore    string
	GoPipelineDownModules    string
	GoPipelineDownDirModules string

	GoGitBaseUrlFlag = strings.Replace(GoGitBaseUrlKey, "_", "-", -1)

	PipelineUpDirSrcFlag   = strings.Replace(PipelineUpDirSrcKey, "_", "-", -1)
	PipelineUpDirPushFlag  = strings.Replace(PipelineUpDirPushKey, "_", "-", -1)
	PipelineUpDirWatchFlag = strings.Replace(PipelineUpDirWatchKey, "_", "-", -1)

	GoPipelineUpFileModFlag = strings.Replace(GoPipelineUpFileModKey, "_", "-", -1)

	PipelineUpBuildCommandFlag = strings.Replace(PipelineUpBuildCommandKey, "_", "-", -1)
	PipelineUpTestCommandFlag  = strings.Replace(PipelineUpTestCommandKey, "_", "-", -1)
	PipelineUpStartCommandFlag = strings.Replace(PipelineUpStartCommandKey, "_", "-", -1)

	PipelineUpRegexIgnoreFlag    = strings.Replace(PipelineUpRegexIgnoreKey, "_", "-", -1)
	GoPipelineDownModulesFlag    = strings.Replace(GoPipelineDownModulesKey, "_", "-", -1)
	GoPipelineDownDirModulesFlag = strings.Replace(GoPipelineDownDirModulesKey, "_", "-", -1)
)

const (
	GitUpCommitMessage = "up_synced"
	GitUpRemoteName    = "dibs-sync"
	GitUpUserName      = "dibs-syncer"
	GitUpUserEmail     = "dibs-syncer@pojtinger.space"

	PlatformPlaceholder    = "[infer]"
	IgnoreRegexPlaceholder = "[infer]"

	EnvPrefix = "dibs"

	GoGitBaseUrlKey = LangGo + "_git_base_url"

	PipelineUpDirSrcKey   = "dir_src"
	PipelineUpDirPushKey  = "dir_push"
	PipelineUpDirWatchKey = "dir_watch"

	GoPipelineUpFileModKey = LangGo + "_modules_file"

	PipelineUpBuildCommandKey = "cmd_build"
	PipelineUpTestCommandKey  = "cmd_test"
	PipelineUpStartCommandKey = "cmd_start"

	PipelineUpRegexIgnoreKey    = "regex_ignore"
	GoPipelineDownModulesKey    = LangGo + "_modules_pull"
	GoPipelineDownDirModulesKey = LangGo + "_dir_pull"
)

func init() {
	id := uuid.New().String()

	PipelineSyncClientCmd.PersistentFlags().StringVar(&GoGitUpBaseUrl, GoGitBaseUrlFlag, "http://localhost:35000/repos", `(--lang "`+LangGo+`" only) Base URL of the sync remote`)

	PipelineSyncClientCmd.PersistentFlags().StringVar(&PipelineUpDirSrc, PipelineUpDirSrcFlag, ".", "Directory in which the source code of the module to push resides")
	PipelineSyncClientCmd.PersistentFlags().StringVar(&PipelineUpDirPush, PipelineUpDirPushFlag, filepath.Join(os.TempDir(), "dibs", "push", id), "Temporary directory to put the module into before pushing")
	PipelineSyncClientCmd.PersistentFlags().StringVar(&PipelineUpDirWatch, PipelineUpDirWatchFlag, ".", "Directory to watch for changes")

	PipelineSyncClientCmd.PersistentFlags().StringVar(&GoPipelineUpFileMod, GoPipelineUpFileModFlag, "go.mod", `(--lang "`+LangGo+`" only) Go module file of the module to push`)

	PipelineSyncClientCmd.PersistentFlags().StringVar(&PipelineUpBuildCommand, PipelineUpBuildCommandFlag, os.Args[0]+" --platform "+PlatformPlaceholder+" pipeline build assets", "Command to run to build the module. Infers the platform from the parent command by default")
	PipelineSyncClientCmd.PersistentFlags().StringVar(&PipelineUpTestCommand, PipelineUpTestCommandFlag, os.Args[0]+" --platform "+PlatformPlaceholder+" pipeline test unit lang", "Command to run to test the module. Infers the platform from the parent command by default")
	PipelineSyncClientCmd.PersistentFlags().StringVar(&PipelineUpStartCommand, PipelineUpStartCommandFlag, os.Args[0]+" --platform "+PlatformPlaceholder+" pipeline test integration assets", "Command to run to start the module. Infers the platform from the parent command by default")

	PipelineSyncClientCmd.PersistentFlags().StringVar(&PipelineUpRegexIgnore, PipelineUpRegexIgnoreFlag, IgnoreRegexPlaceholder, "Regular expression for files to ignore. If a dibs configuration file exists, it will infer it from assets.cleanGlob")
	PipelineSyncClientCmd.PersistentFlags().StringVarP(&GoPipelineDownModules, GoPipelineDownModulesFlag, "g", "", `(--lang "`+LangGo+`" only) Comma-separated list of the names of the modules to pull`)
	PipelineSyncClientCmd.PersistentFlags().StringVar(&GoPipelineDownDirModules, GoPipelineDownDirModulesFlag, filepath.Join(os.TempDir(), "dibs", "pull", id), `(--lang "`+LangGo+`" only) Directory to pull the modules to`)

	viper.SetEnvPrefix(EnvPrefix)

	if err := viper.BindPFlag(GoGitBaseUrlKey, PipelineSyncClientCmd.PersistentFlags().Lookup(GoGitBaseUrlFlag)); err != nil {
		log.Fatal("Could not bind flag", rz.Err(err))
	}

	if err := viper.BindPFlag(PipelineUpDirSrcKey, PipelineSyncClientCmd.PersistentFlags().Lookup(PipelineUpDirSrcFlag)); err != nil {
		log.Fatal("Could not bind flag", rz.Err(err))
	}
	if err := viper.BindPFlag(PipelineUpDirPushKey, PipelineSyncClientCmd.PersistentFlags().Lookup(PipelineUpDirPushFlag)); err != nil {
		log.Fatal("Could not bind flag", rz.Err(err))
	}
	if err := viper.BindPFlag(PipelineUpDirWatchKey, PipelineSyncClientCmd.PersistentFlags().Lookup(PipelineUpDirWatchFlag)); err != nil {
		log.Fatal("Could not bind flag", rz.Err(err))
	}

	if err := viper.BindPFlag(GoPipelineUpFileModKey, PipelineSyncClientCmd.PersistentFlags().Lookup(GoPipelineUpFileModFlag)); err != nil {
		log.Fatal("Could not bind flag", rz.Err(err))
	}

	if err := viper.BindPFlag(PipelineUpBuildCommandKey, PipelineSyncClientCmd.PersistentFlags().Lookup(PipelineUpBuildCommandFlag)); err != nil {
		log.Fatal("Could not bind flag", rz.Err(err))
	}
	if err := viper.BindPFlag(PipelineUpTestCommandKey, PipelineSyncClientCmd.PersistentFlags().Lookup(PipelineUpTestCommandFlag)); err != nil {
		log.Fatal("Could not bind flag", rz.Err(err))
	}
	if err := viper.BindPFlag(PipelineUpStartCommandKey, PipelineSyncClientCmd.PersistentFlags().Lookup(PipelineUpStartCommandFlag)); err != nil {
		log.Fatal("Could not bind flag", rz.Err(err))
	}

	if err := viper.BindPFlag(PipelineUpRegexIgnoreKey, PipelineSyncClientCmd.PersistentFlags().Lookup(PipelineUpRegexIgnoreFlag)); err != nil {
		log.Fatal("Could not bind flag", rz.Err(err))
	}
	if err := viper.BindPFlag(GoPipelineDownModulesKey, PipelineSyncClientCmd.PersistentFlags().Lookup(GoPipelineDownModulesFlag)); err != nil {
		log.Fatal("Could not bind flag", rz.Err(err))
	}
	if err := viper.BindPFlag(GoPipelineDownDirModulesKey, PipelineSyncClientCmd.PersistentFlags().Lookup(GoPipelineDownDirModulesFlag)); err != nil {
		log.Fatal("Could not bind flag", rz.Err(err))
	}

	viper.AutomaticEnv()

	PipelineSyncCmd.AddCommand(PipelineSyncClientCmd)
}
