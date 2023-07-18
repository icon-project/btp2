/*
 * Copyright 2021 ICON Foundation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/icon-project/btp2/common/cli"
	"github.com/icon-project/btp2/common/link"
	"github.com/icon-project/btp2/common/log"
	"github.com/icon-project/btp2/common/relay"
)

var (
	version = "unknown"
	build   = "unknown"
)

var logoLines = []string{
	"  _____      _             ",
	" |  __ \\    | |            ",
	" | |__) |___| | __ _ _   _ ",
	" |  _  // _ \\ |/ _` | | | |",
	" | | \\ \\  __/ | (_| | |_| |",
	" |_|  \\_\\___|_|\\__,_|\\__, |",
	"                      __/ |",
	"                     |___/ ",
}

func main() {
	rootCmd, rootVc := cli.NewCommand(nil, nil, "relay", "BTP Relay CLI")
	cfg := &link.Config{}
	rootCmd.Long = "Command Line Interface of Relay for Blockchain Transmission Protocol"
	cli.SetEnvKeyReplacer(rootVc, strings.NewReplacer(".", "_"))
	//rootVc.Debug()
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print relay version",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("relay version", version, build)
		},
	})

	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		baseDir := rootVc.GetString("base_dir")
		logfile := rootVc.GetString("log_writer.filename")
		cfg.FilePath = rootVc.GetString("config")
		if cfg.FilePath != "" {
			f, err := os.Open(cfg.FilePath)
			if err != nil {
				return fmt.Errorf("fail to open config file=%s err=%+v", cfg.FilePath, err)
			}
			rootVc.SetConfigType("json")
			err = rootVc.ReadConfig(f)
			if err != nil {
				return fmt.Errorf("fail to read config file=%s err=%+v", cfg.FilePath, err)
			}
			cfg.FilePath, _ = filepath.Abs(cfg.FilePath)
		}
		if err := rootVc.Unmarshal(&cfg, cli.ViperDecodeOptJson); err != nil {
			return fmt.Errorf("fail to unmarshall config from env err=%+v", err)
		}
		if baseDir != "" {
			cfg.BaseDir = cfg.ResolveRelative(baseDir)
		}
		if logfile != "" {
			cfg.LogWriter.Filename = cfg.ResolveRelative(logfile)
		}
		return nil
	}
	rootPFlags := rootCmd.PersistentFlags()

	rootPFlags.StringP("config", "c", "", "Parsing configuration file")

	//Chains Config
	rootPFlags.StringToString("chains_config.src", nil, "Source chain config")
	rootPFlags.StringToString("chains_config.dst", nil, "Destination chain config")

	//RelayConfig
	rootPFlags.String("relay_config.direction", "both", "relay network direction ( both, front, reverse)")
	rootPFlags.String("relay_config.base_dir", "", "Base directory for data")
	rootPFlags.String("relay_config.log_level", "debug", "Global log level (trace,debug,info,warn,error,fatal,panic)")
	rootPFlags.String("relay_config.console_level", "trace", "Console log level (trace,debug,info,warn,error,fatal,panic)")
	rootPFlags.String("relay_config.log_forwarder.vendor", "", "LogForwarder vendor (fluentd,logstash)")
	rootPFlags.String("relay_config.log_forwarder.address", "", "LogForwarder address")
	rootPFlags.String("relay_config.log_forwarder.level", "info", "LogForwarder level")
	rootPFlags.String("relay_config.log_forwarder.name", "", "LogForwarder name")
	rootPFlags.StringToString("relay_config.log_forwarder.options", nil, "LogForwarder options, comma-separated 'key=value'")
	rootPFlags.String("relay_config.log_writer.filename", "", "Log file name (rotated files resides in same directory)")
	rootPFlags.Int("relay_config.log_writer.maxsize", 100, "Maximum log file size in MiB")
	rootPFlags.Int("relay_config.log_writer.maxage", 0, "Maximum age of log file in day")
	rootPFlags.Int("relay_config.log_writer.maxbackups", 0, "Maximum number of backups")
	rootPFlags.Bool("relay_config.log_writer.localtime", false, "Use localtime on rotated log file instead of UTC")
	rootPFlags.Bool("relay_config.log_writer.compress", false, "Use gzip on rotated log file")
	cli.BindPFlags(rootVc, rootPFlags)

	saveCmd := &cobra.Command{
		Use:   "save [file]",
		Short: "Save configuration",
		Args:  cli.ArgsWithDefaultErrorFunc(cobra.ExactArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			saveFilePath := args[0]
			cfg.FilePath, _ = filepath.Abs(saveFilePath)
			cfg.BaseDir = cfg.ResolveRelative(cfg.BaseDir)

			if cfg.LogWriter != nil {
				cfg.LogWriter.Filename = cfg.ResolveRelative(cfg.LogWriter.Filename)
			}

			if err := cli.JsonPrettySaveFile(saveFilePath, 0644, cfg); err != nil {
				return err
			}
			cmd.Println("Save configuration to", saveFilePath)
			return nil
		},
	}
	rootCmd.AddCommand(saveCmd)
	saveCmd.Flags().String("save_key_store", "", "KeyStore File path to save")

	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start server",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return cli.ValidateFlagsWithViper(rootVc, cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, l := range logoLines {
				log.Println(l)
			}
			log.Printf("Version : %s", version)
			log.Printf("Build   : %s", build)

			modLevels, _ := cmd.Flags().GetStringToString("mod_level")

			relay, err := relay.NewRelay(cfg, modLevels)
			if err != nil {
				return err
			}

			return relay.Start()
		},
	}
	rootCmd.AddCommand(startCmd)
	startFlags := startCmd.Flags()
	startFlags.StringToString("mod_level", nil, "Set console log level for specific module ('mod'='level',...)")
	startFlags.String("cpuprofile", "", "CPU Profiling data file")
	startFlags.String("memprofile", "", "Memory Profiling data file")
	startFlags.MarkHidden("mod_level")

	cli.BindPFlags(rootVc, startFlags)

	genMdCmd := cli.NewGenerateMarkdownCommand(rootCmd, rootVc)
	genMdCmd.Hidden = true

	rootCmd.SilenceUsage = true
	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("%+v", err)
		os.Exit(1)
	}
}
