/*
Copyright 2022 Huawei Cloud Computing Technologies Co., Ltd.

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
	"flag"
	"fmt"

	"github.com/influxdata/influxdb/client"
	"github.com/openGemini/openGemini/app/ts-cli/geminicli"
)

// the options of command line are capatible
// with influx v1.x.
type CapatibleCommand struct {
	FS    *flag.FlagSet
	Bind  func(fs *flag.FlagSet, c *geminicli.CommandLineConfig)
	Usage func()
}

var (
	capatibleCmd = &CapatibleCommand{
		FS: flag.NewFlagSet("openGemini CLI version "+geminicli.CLIENT_VERSION, flag.ExitOnError),
		Bind: func(fs *flag.FlagSet, c *geminicli.CommandLineConfig) {
			fs.StringVar(&c.Host, "host", client.DefaultHost, "Influxdb host to connect to.")
			fs.IntVar(&c.Port, "port", client.DefaultPort, "Influxdb port to connect to.")
			fs.StringVar(&c.UnixSocket, "socket", "", "Influxdb unix socket to connect to.")
			fs.StringVar(&c.Username, "username", "", "Username to connect to the server.")
			fs.StringVar(&c.Password, "password", "", `Password to connect to the server.  Leaving blank will prompt for password (--password="").`)
			fs.StringVar(&c.Database, "database", c.Database, "Database to connect to the server.")
			fs.BoolVar(&c.Ssl, "ssl", false, "Use https for connecting to cluster.")
			fs.BoolVar(&c.IgnoreSsl, "unsafeSsl", false, "Set this when connecting to the cluster using https and not use SSL verification.")
		},
		Usage: func() {
			fmt.Println(`Usage of influx:
			-version
					  Display the version and exit.
			-host 'host name'
					  Host to connect to.
			-port 'port #'
					  Port to connect to.
			-socket 'unix domain socket'
					  Unix socket to connect to.
			-database 'database name'
					  Database to connect to the server.
			-password 'password'
					  Password to connect to the server.  Leaving blank will prompt for password (--password '').
			-username 'username'
					  Username to connect to the server.
			-ssl
					  Use https for requests.
			-unsafeSsl
					  Set this when connecting to the cluster using https and not use SSL verification.`)
		},
	}
)
