package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/AliyunContainerService/image-syncer/pkg/client"
	"github.com/spf13/cobra"
)

var (
	dbMode bool

	syncInterval int

	logPath, configFile, authFile, imageFile, defaultRegistry, defaultNamespace string

	procNum, retries int

	osFilterList, archFilterList []string
)

// RootCmd describes "image-syncer" command
var RootCmd = &cobra.Command{
	Use:     "image-syncer",
	Aliases: []string{"image-syncer"},
	Short:   "A docker registry image synchronization tool",
	Long: `A Fast and Flexible docker registry image synchronization tool implement by Go. 
	
	Complete documentation is available at https://github.com/AliyunContainerService/image-syncer`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// work starts here

		// if db config specified, start service as backend
		if dbMode {
			fmt.Println("start backend mode, get config from database")
			// initialize database
			startDBBackend()

		} else {
			client, err := client.NewSyncClient(configFile, authFile, imageFile, logPath, procNum, retries,
				defaultRegistry, defaultNamespace, osFilterList, archFilterList)
			if err != nil {
				return fmt.Errorf("init sync client error: %v", err)
			}
			client.Run()
			return nil
		}
		return nil
	},
}

func startDBBackend() error {
	var (
		// ticker = time.NewTicker(time.Minute)	// default search db update in 1 minutes
		duration = time.Second * time.Duration(syncInterval)
		ticker   = time.NewTimer(duration)
	)

	for {
		<-ticker.C
		client, err := client.NewSyncClientFromDB(logPath, procNum, retries,
			defaultRegistry, defaultNamespace, osFilterList, archFilterList)
		if err != nil {
			return fmt.Errorf("init sync client error: %v", err)
		}
		client.Run()

		// start a new timer
		ticker = time.NewTimer(duration)
	}
}

func init() {
	RootCmd.PersistentFlags().BoolVarP(&dbMode, "dbMode", "d", false, "whether acuqire config from database, dafault to false.")
	RootCmd.PersistentFlags().IntVar(&syncInterval, "syncInterval", 3600, "syncInterval determine how many times takes (Unit: seconds) between two sync when db mode enable. default to 60s.")
	RootCmd.PersistentFlags().StringVar(&configFile, "config", "", "config file path. This flag is deprecated and will be removed in the future. Please use --auth and --images instead.")
	RootCmd.PersistentFlags().StringVar(&authFile, "auth", "", "auth file path. This flag need to be pair used with --images.")
	RootCmd.PersistentFlags().StringVar(&imageFile, "images", "", "images file path. This flag need to be pair used with --auth")
	RootCmd.PersistentFlags().StringVar(&logPath, "log", "", "log file path (default in os.Stderr)")
	RootCmd.PersistentFlags().StringVar(&defaultRegistry, "registry", os.Getenv("DEFAULT_REGISTRY"),
		"default destination registry url when destination registry is not given in the config file, can also be set with DEFAULT_REGISTRY environment value")
	RootCmd.PersistentFlags().StringVar(&defaultNamespace, "namespace", os.Getenv("DEFAULT_NAMESPACE"),
		"default destination namespace when destination namespace is not given in the config file, can also be set with DEFAULT_NAMESPACE environment value")
	RootCmd.PersistentFlags().IntVarP(&procNum, "proc", "p", 5, "numbers of working goroutines")
	RootCmd.PersistentFlags().IntVarP(&retries, "retries", "r", 2, "times to retry failed task")
	RootCmd.PersistentFlags().StringArrayVar(&osFilterList, "os", []string{}, "os list to filter source tags, not works for docker v2 schema1 media")
	RootCmd.PersistentFlags().StringArrayVar(&archFilterList, "arch", []string{}, "architecture list to filter source tags")
}

// Execute executes the RootCmd
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}
