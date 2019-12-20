package cmd

import (
	"github.com/pojntfx/dibs/pkg/pipes"
	"github.com/pojntfx/dibs/pkg/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"net"
	"strings"
)

var PipelineTestIntegrationChartCmd = &cobra.Command{
	Use:   "chart",
	Short: "Integration test the chart",
	Run: func(cmd *cobra.Command, args []string) {
		platformFromConfig := viper.GetString(PlatformKey)
		viperIP := viper.GetString(TestIntegrationChartKubernetesIpKey)

		rawIP := net.ParseIP(viperIP)
		if rawIP == nil {
			utils.LogErrorFatalCouldNotParseIP(viperIP)
			return
		}
		ip := rawIP.String()

		Dibs.RunForPlatforms(platformFromConfig, platformFromConfig == PlatformAll, func(platform pipes.Platform) {
			if viper.GetString(ExecutorKey) == ExecutorDocker {
				if output, err := platform.Tests.Integration.Chart.BuildImage(platform.Platform); err != nil {
					utils.LogErrorFatalPlatformSpecific("Could not build chart integration test chart", err, platform.Platform, output)
				}
				output, err := platform.Tests.Integration.Chart.StartImage(platform.Platform, struct {
					Key   string
					Value string
				}{
					Key:   "IP",
					Value: ip,
				})
				utils.LogErrorInfo("Chart integration test ran in Docker", err, platform.Platform, output)
			} else {
				output, err := platform.Tests.Integration.Chart.Start(platform.Platform)
				utils.LogErrorInfo("Chart integration test ran", err, platform.Platform, output)
			}
		})
	},
}

func init() {
	var (
		kubernetesIp string

		kubernetesIpFlag = strings.Replace(TestIntegrationChartKubernetesIpKey, "_", "-", -1)
	)

	PipelineTestIntegrationChartCmd.PersistentFlags().StringVarP(&kubernetesIp, kubernetesIpFlag, "i", TestIntegrationChartKubernetesIpDefault, "IP of the Kubernetes cluster to create if running in Docker (often the host machine's IP)")

	viper.SetEnvPrefix(EnvPrefix)

	if err := viper.BindPFlag(TestIntegrationChartKubernetesIpKey, PipelineTestIntegrationChartCmd.PersistentFlags().Lookup(kubernetesIpFlag)); err != nil {
		utils.LogErrorCouldNotBindFlag(err)
	}

	viper.AutomaticEnv()

	PipelineTestIntegrationCmd.AddCommand(PipelineTestIntegrationChartCmd)
}
