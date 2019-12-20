package cmd

import (
	"github.com/pojntfx/dibs/pkg/pipes"
	"github.com/pojntfx/dibs/pkg/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var PipelineTestIntegrationAssetsCmd = &cobra.Command{
	Use:   "assets",
	Short: "Integration test the assets",
	Run: func(cmd *cobra.Command, args []string) {
		platformFromConfig := viper.GetString(PlatformKey)

		Dibs.RunForPlatforms(platformFromConfig, platformFromConfig == PlatformAll, func(platform pipes.Platform) {
			if viper.GetString(ExecutorKey) == ExecutorDocker {
				if output, err := platform.Tests.Integration.Assets.BuildImage(platform.Platform); err != nil {
					utils.LogErrorFatalPlatformSpecific("Could not build assets integration test image", err, platform.Platform, output)
				}
				output, err := platform.Tests.Integration.Assets.StartImage(platform.Platform)
				utils.LogErrorInfo("Assets integration test ran in Docker", err, platform.Platform, output)
			} else {
				output, err := platform.Tests.Integration.Assets.Start(platform.Platform)
				utils.LogErrorInfo("Assets integration test ran", err, platform.Platform, output)
			}
		})
	},
}

func init() {
	PipelineTestIntegrationCmd.AddCommand(PipelineTestIntegrationAssetsCmd)
}
