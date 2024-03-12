package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"beryju.org/korb/pkg/config"
	"beryju.org/korb/pkg/migrator"
	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
	"k8s.io/client-go/util/homedir"
)

var kubeConfig string
var sourceNamespace string
var strategy string

var pvcNewStorageClass string
var pvcNewSize string
var pvcNewName string
var pvcNewNamespace string
var pvcNewAccessModes []string

var serviceAccountName string

var force bool
var skipWaitPVCBind bool
var tolerateAllNodes bool
var Version string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "korb [pvc [pvc]]",
	Version: Version,
	Long:    `Move data between Kubernetes PVCs on different Storage Classes.`,
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		for _, pvc := range args {
			m := migrator.New(kubeConfig, strategy, tolerateAllNodes)
			m.Force = force
			m.WaitForTempDestPVCBind = skipWaitPVCBind

			// We can only support operating in a single namespace currently
			// Since cross-namespace PVC mounts are not a thing
			// we'd have to transfer the data over the network, which uh
			// I don't really feel like implementing it
			if sourceNamespace != "" {
				m.SourceNamespace = sourceNamespace
				m.DestNamespace = sourceNamespace
			}
			// if pvcNewNamespace != "" {
			// 	m.DestNamespace = pvcNewNamespace
			// }

			m.DestPVCSize = pvcNewSize
			m.DestPVCStorageClass = pvcNewStorageClass
			m.DestPVCName = pvcNewName
			m.DestPVCAccessModes = pvcNewAccessModes

			m.ServiceAccountName = serviceAccountName

			m.SourcePVCName = pvc
			m.Run()
			if len(args) > 1 {
				fmt.Println("=====================")
			}
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
		rootCmd.Flags().StringVar(&kubeConfig, "kube-config", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		rootCmd.Flags().StringVar(&kubeConfig, "kube-config", "", "absolute path to the kubeconfig file")
	}
	rootCmd.Flags().StringVar(&sourceNamespace, "source-namespace", "", "Namespace where the old PVCs reside. If empty, the namespace from your kubeconfig file will be used.")

	rootCmd.Flags().StringVar(&pvcNewStorageClass, "new-pvc-storage-class", "", "Storage class to use for the new PVC. If empty, the storage class of the source will be used.")
	rootCmd.Flags().StringVar(&pvcNewName, "new-pvc-name", "", "Name for the new PVC. If empty, same name will be reused.")
	rootCmd.Flags().StringVar(&pvcNewSize, "new-pvc-size", "", "Size for the new PVC. If empty, the size of the source will be used. Accepts formats like used in Kubernetes Manifests (Gi, Ti, ...)")
	rootCmd.Flags().StringVar(&pvcNewNamespace, "new-pvc-namespace", "", "Namespace for the new PVCs to be created in. If empty, the namespace from your kubeconfig file will be used.")
	rootCmd.Flags().StringSliceVar(&pvcNewAccessModes, "new-pvc-access-mode", []string{}, "Access mode(s) for the new PVC. If empty, the access mode of the source will be used. Accepts formats like used in Kubernetes Manifests (ReadWriteOnce, ReadWriteMany, ...)")

	rootCmd.Flags().StringVar(&serviceAccountName, "service-account-name", "", "Service Account Name to use for the Job execution. If empty, 'default' used.")

	rootCmd.Flags().BoolVar(&force, "force", false, "Ignore warning which would normally halt the tool during validation.")
	rootCmd.Flags().BoolVar(&skipWaitPVCBind, "skip-pvc-bind-wait", false, "Skip waiting for PVC to be bound.")
	rootCmd.Flags().BoolVar(&tolerateAllNodes, "tolerate-any-node", false, "Allow job to tolerating any node node taints.")

	rootCmd.Flags().StringVar(&config.ContainerImage, "container-image", config.ContainerImage, "Image to use for moving jobs")
	rootCmd.Flags().StringVar(&strategy, "strategy", "", "Strategy to use, by default will try to auto-select")
}
