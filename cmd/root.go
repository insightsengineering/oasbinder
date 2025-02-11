/*
Copyright 2025 F. Hoffmann-La Roche AG

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/jamiealquiza/envy"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.szostok.io/version/extension"
)

var cfgFile string
var logLevel string
var services []Microservice
var oasbinderAddress string
var oasbinderPortNumber int
var apiSpecsPath string
var headers map[string]string

var log = logrus.New()

func setLogLevel() {
	customFormatter := new(logrus.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	customFormatter.ForceColors = true
	log.SetFormatter(customFormatter)
	log.SetReportCaller(false)
	customFormatter.FullTimestamp = false
	fmt.Println(`logLevel = "` + logLevel + `"`)
	switch logLevel {
	case "trace":
		log.SetLevel(logrus.TraceLevel)
	case "debug":
		log.SetLevel(logrus.DebugLevel)
	case "info":
		log.SetLevel(logrus.InfoLevel)
	case "warn":
		log.SetLevel(logrus.WarnLevel)
	case "error":
		log.SetLevel(logrus.ErrorLevel)
	default:
		log.SetLevel(logrus.InfoLevel)
	}
}

var rootCmd *cobra.Command

func newRootCommand() {
	rootCmd = &cobra.Command{
		Use:   "oasbinder",
		Short: "",
		Long:  ``,
		PersistentPreRun: func(_ *cobra.Command, _ []string) {
			initializeConfig()
		},
		Run: func(_ *cobra.Command, _ []string) {
			setLogLevel()

			fmt.Println(`config = "` + cfgFile + `"`)
			fmt.Println(`address = "` + oasbinderAddress + `"`)
			fmt.Println(`port = ` + strconv.Itoa(oasbinderPortNumber))
			fmt.Println(`apiSpecsPath = "` + apiSpecsPath + `"`)

			serve()
		},
	}
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "",
		"config file (default is $HOME/.oasbinder.yaml)")
	rootCmd.PersistentFlags().StringVarP(&logLevel, "logLevel", "l", "info",
		"Logging level (trace, debug, info, warn, error). ")
	rootCmd.PersistentFlags().StringVarP(&oasbinderAddress, "address", "a", "http://localhost:8080",
		"Address where oasbinder is accessed by the user. It should have the format: http[s]://hostname.example.com[:port]")
	rootCmd.PersistentFlags().IntVarP(&oasbinderPortNumber, "port", "p", 8080,
		"Port number on which oasbinder will be listening.")
	rootCmd.PersistentFlags().StringVarP(&apiSpecsPath, "apiSpecsPath", "s", "/openapi.json",
		"Path where microservices expose their API specification.")

	// Add version command.
	rootCmd.AddCommand(extension.NewVersionCobraCmd())

	cfg := envy.CobraConfig{
		Prefix:     "OASBINDER",
		Persistent: true,
	}
	envy.ParseCobra(rootCmd, cfg)
}

func init() {
	cobra.OnInitialize(initConfig)
}

func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search for config in home directory.
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".oasbinder")
	}
	// Read in environment variables that match.
	viper.AutomaticEnv()

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	} else {
		fmt.Println(err)
	}
}

func Execute() {
	newRootCommand()
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func initializeConfig() {
	for _, v := range []string{
		"logLevel", "address", "port", "apiSpecsPath",
	} {
		// If the flag has not been set in newRootCommand() and it has been set in initConfig().
		// In other words: if it's not been provided in command line, but has been
		// provided in config file.
		// Helpful project where it's explained:
		// https://github.com/carolynvs/stingoftheviper
		if !rootCmd.PersistentFlags().Lookup(v).Changed && viper.IsSet(v) {
			err := rootCmd.PersistentFlags().Set(v, fmt.Sprintf("%v", viper.Get(v)))
			checkError(err)
		}
	}

	// Read the list of microservice docs to serve.
	err := viper.UnmarshalKey("services", &services)
	checkError(err)

	// Read the list of headers to include when making requests to microservices.
	err = viper.UnmarshalKey("headers", &headers)
	checkError(err)
}
