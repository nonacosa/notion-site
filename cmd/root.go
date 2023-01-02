package cmd

import (
	"fmt"
	"github.com/joho/godotenv"
	"github.com/pkwenda/notion-site/pkg"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"log"
	"os"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "notion-site",
	Short: "A markdown generator for notion",
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		var config pkg.Config
		if err := viper.Unmarshal(&config); err != nil {
			log.Fatal(err)
		}
		api := pkg.NewAPI()
		files := pkg.NewFiles(config)
		tm := pkg.New()
		caches := pkg.NewNotionCaches()
		ns := pkg.NewNotionSite(api, tm, files, config, caches)

		if err := pkg.Run(ns); err != nil {
			log.Println(err)
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is notion-site.yaml)")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath(".")
		viper.SetConfigName("notion-site")
	}

	if err := godotenv.Load(); err == nil {
		fmt.Println("Load .env file")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
