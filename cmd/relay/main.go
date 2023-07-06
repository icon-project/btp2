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
	"github.com/icon-project/btp2/common/linkfactory"
	"github.com/icon-project/btp2/common/log"
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
	cfg := &linkfactory.Config{}
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
	rootPFlags.String("src.address", "", "BTP Address of source blockchain (PROTOCOL://NID.BLOCKCHAIN/BMC)")
	rootPFlags.String("src.endpoint", "", "Endpoint of source blockchain")
	rootPFlags.StringToString("src.options", nil, "Options, comma-separated 'key=value'")
	rootPFlags.String("src.key_store", "", "Source keyStore")
	rootPFlags.String("src.key_password", "", "Source password of keyStore")
	rootPFlags.String("src.key_secret", "", "Source Secret(password) file for keyStore")
	rootPFlags.String("src.relay_mode", "trustless", "Relay Mode")
	rootPFlags.Bool("src.latest_result", false, "Sends relay messages regardless of final status reception.")
	rootPFlags.Bool("src.filled_block_update", false, "Create relayMessage for all data received from the source network")

	rootPFlags.String("dst.address", "", "BTP Address of destination blockchain (PROTOCOL://NID.BLOCKCHAIN/BMC)")
	rootPFlags.String("dst.endpoint", "", "Endpoint of destination blockchain")
	rootPFlags.StringToString("dst.options", nil, "Options, comma-separated 'key=value'")
	rootPFlags.String("dst.key_store", "", "Destination keyStore")
	rootPFlags.String("dst.key_password", "", "Destination password of keyStore")
	rootPFlags.String("dst.key_secret", "", "Destination Secret(password) file for keyStore")
	rootPFlags.String("dst.relay_mode", "trustless", "Relay Mode")
	rootPFlags.Bool("dst.latest_result", false, "Sends relay messages regardless of final status reception.")
	rootPFlags.Bool("dst.filled_block_update", false, "Create relayMessage for all data received from the source network")

	rootPFlags.String("direction", "both", "relay network direction ( both, front, reverse)")
	rootPFlags.Bool("maxSizeTx", false, "Send when the maximum transaction size is reached")

	rootPFlags.Int64("offset", 0, "Offset of MTA")

	//
	rootPFlags.String("base_dir", "", "Base directory for data")
	rootPFlags.StringP("config", "c", "", "Parsing configuration file")
	//
	rootPFlags.String("log_level", "debug", "Global log level (trace,debug,info,warn,error,fatal,panic)")
	rootPFlags.String("console_level", "trace", "Console log level (trace,debug,info,warn,error,fatal,panic)")
	//
	rootPFlags.String("log_forwarder.vendor", "", "LogForwarder vendor (fluentd,logstash)")
	rootPFlags.String("log_forwarder.address", "", "LogForwarder address")
	rootPFlags.String("log_forwarder.level", "info", "LogForwarder level")
	rootPFlags.String("log_forwarder.name", "", "LogForwarder name")
	rootPFlags.StringToString("log_forwarder.options", nil, "LogForwarder options, comma-separated 'key=value'")
	//
	rootPFlags.String("log_writer.filename", "", "Log file name (rotated files resides in same directory)")
	rootPFlags.Int("log_writer.maxsize", 100, "Maximum log file size in MiB")
	rootPFlags.Int("log_writer.maxage", 0, "Maximum age of log file in day")
	rootPFlags.Int("log_writer.maxbackups", 0, "Maximum number of backups")
	rootPFlags.Bool("log_writer.localtime", false, "Use localtime on rotated log file instead of UTC")
	rootPFlags.Bool("log_writer.compress", false, "Use gzip on rotated log file")
	cli.BindPFlags(rootVc, rootPFlags)
	err := cli.MarkAnnotationCustom(rootPFlags, "src.address", "dst.address", "src.endpoint", "dst.endpoint")
	if err != nil {
		return
	}
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
			if saveSrcKeyStore, _ := cmd.Flags().GetString("save_src_key_store"); saveSrcKeyStore != "" {
				if err := cli.JsonPrettySaveFile(saveSrcKeyStore, 0600, cfg.Src.KeyStoreData); err != nil {
					return err
				}
			}

			if saveDstKeyStore, _ := cmd.Flags().GetString("save_dst_key_store"); saveDstKeyStore != "" {
				if err := cli.JsonPrettySaveFile(saveDstKeyStore, 0600, cfg.Dst.KeyStoreData); err != nil {
					return err
				}
			}
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

			lf, err := linkfactory.NewLinkFactory(cfg, modLevels)
			if err != nil {
				return err
			}
			
			return lf.Start()
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
