/*
Copyright 2018 Pressinfra SRL

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

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/go-logr/zapr"
	logf "github.com/presslabs/controller-util/log"
	"github.com/presslabs/mysql-operator/pkg/sidecar"
	"github.com/spf13/cobra"
)

var log = logf.Log.WithName("sidecar")

// TODO 这个main函数用于启动提供备份服务的sidecar(基于xbackup) 或者 调用提供备份的sidecar的接口触发备份，
//  也就是备份服务的server和cli入口都在这
func main() {
	stop := make(chan struct{})

	cmd := &cobra.Command{
		Use:   "mysql-operator-sidecar",
		Short: "Helper for mysql operator.",
		Long:  `mysql-operator-sidecar: helper for config pods`,
		Run: func(cmd *cobra.Command, args []string) {
			log.Info("you run mysql-operator-sidecar, see help section")
			os.Exit(1)

		},
	}

	// add flags and parse them
	debug := false
	flag.BoolVar(&debug, "debug", false, "Set logger in debug mode")
	cmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	if err := cmd.ParseFlags(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "failed to parse global flags, see helps, err: %s", err)
		os.Exit(1)
	}

	// setup logging
	logf.SetLogger(zapr.NewLogger(logf.RawStackdriverZapLoggerTo(os.Stderr, true)))

	// init configs
	// TODO 初始化配置，检查数据目录下有无数据等
	cfg := sidecar.NewConfig()

	// TODO 当参数是clone-and-init, 从指定备份数据初始化一个MySQL实例(只有对应的PVC不存在才会生效)
	cloneCmd := &cobra.Command{
		Use:   "clone-and-init",
		Short: "Clone data from a bucket or prior node.",
		Run: func(cmd *cobra.Command, args []string) {
			if err := sidecar.RunCloneCommand(cfg); err != nil {
				log.Error(err, "clone command failed")
				os.Exit(8)
			}
			if err := sidecar.RunConfigCommand(cfg); err != nil {
				log.Error(err, "init command failed")
				os.Exit(1)
			}
		},
	}
	cmd.AddCommand(cloneCmd)

	// TODO 用于启动sidecar的命令(参数是run时启动提供备份服务的sidecar)
	sidecarCmd := &cobra.Command{
		Use:   "run",
		Short: "Configs mysql users, replication, and serve backups.",
		Run: func(cmd *cobra.Command, args []string) {
			// TODO sidecar提供配置mysql用户、副本以及触发备份的http接口
			err := sidecar.RunSidecarCommand(cfg, stop)
			if err != nil {
				log.Error(err, "run command failed")
				os.Exit(1)
			}
		},
	}
	cmd.AddCommand(sidecarCmd)

	// TODO 用于触发数据备份的命令（参数是take-backup-to是通过cli调用sidecar里面的备份接口)
	takeBackupCmd := &cobra.Command{
		Use:   "take-backup-to",
		Short: "Take a backup from node and push it to rclone path.",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 2 {
				return fmt.Errorf("require two arguments. source host and destination bucket")
			}
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			// TODO 调用备份接口触发备份
			err := sidecar.RunTakeBackupCommand(cfg, args[0], args[1])
			if err != nil {
				log.Error(err, "take backup command failed")
				os.Exit(1)

			}
		},
	}
	cmd.AddCommand(takeBackupCmd)

	if err := cmd.Execute(); err != nil {
		log.Error(err, "failed to execute command", "cmd", cmd)
		os.Exit(1)
	}
}
