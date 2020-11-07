package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BeryJu/korb/pkg/config"
	"github.com/BeryJu/korb/pkg/migrator"
	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
	"k8s.io/client-go/util/homedir"
)

var kubeConfig string

var pvcNewStorageClass string
var pvcNewSize string
var pvcNewName string

var force bool

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:  "k8s-storage-mover [pvc [pvc]]",
	Long: `Move data between Kubernetes PVCs on different Storage Classes.`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		for _, pvc := range args {
			m := migrator.New(kubeConfig)
			m.Force = force

			m.DestPVCSize = pvcNewSize
			m.DestPVCStorageClass = pvcNewStorageClass
			m.DestPVCName = pvcNewName

			m.SourcePVCName = pvc
			m.Run()
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	log.SetLevel(log.DebugLevel)

	if home := homedir.HomeDir(); home != "" {
		rootCmd.Flags().StringVar(&kubeConfig, "kubeConfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeConfig file")
	} else {
		rootCmd.Flags().StringVar(&kubeConfig, "kubeConfig", "", "absolute path to the kubeconfig file")
	}

	rootCmd.Flags().StringVar(&pvcNewStorageClass, "new-pvc-storage-class", "", "Storage class to use for the new PVC. If empty, the storage class of the source will be used.")
	rootCmd.Flags().StringVar(&pvcNewName, "new-pvc-name", "", "Name for the new PVC. If empty, same name will be reused.")
	rootCmd.Flags().StringVar(&pvcNewSize, "new-pvc-size", "", "Size for the new PVC. If empty, the size of the source will be used. Accepts formats like used in Kubernetes Manifests (Gi, Ti, ...)")

	rootCmd.Flags().BoolVar(&force, "force", false, "Ignore warning which would normally halt the tool during validation.")

	rootCmd.Flags().StringVar(&config.DockerImage, "docker-image", config.DockerImage, "Image to use for moving jobs")
}
