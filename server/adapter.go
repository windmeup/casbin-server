// Copyright 2018 The casbin Authors. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"regexp"
	"strings"

	"github.com/casbin/casbin/v2/persist"
	fileAdapter "github.com/casbin/casbin/v2/persist/file-adapter"
	gormAdapter "github.com/casbin/gorm-adapter/v3"
	pb "github.com/windmeup/casbin-server/proto"
)

var errDriverName = errors.New("currently supported DriverName: file | mysql | postgres | mssql")

func newAdapter(in *pb.NewAdapterRequest) (persist.Adapter, error) {
	var a persist.Adapter
	in, err := checkLocalConfig(in)
	if err != nil {
		return nil, err
	}
	supportDriverNames := [...]string{"file", "mysql", "postgres", "mssql"}

	switch in.DriverName {
	case "file":
		a = fileAdapter.NewAdapter(in.ConnectString)
	default:
		var support = false
		for _, driverName := range supportDriverNames {
			if driverName == in.DriverName {
				support = true
				break
			}
		}
		if !support {
			return nil, errDriverName
		}

		var err error
		a, err = gormAdapter.NewAdapter(in.DriverName, in.ConnectString, in.DbSpecified)
		if err != nil {
			return nil, err
		}
	}

	return a, nil
}

func checkLocalConfig(in *pb.NewAdapterRequest) (*pb.NewAdapterRequest, error) {
	cfg, err := LoadConfiguration(getLocalConfigPath())
	if err != nil {
		return nil, err
	}
	if in.ConnectString == "" || in.DriverName == "" {
		in.DriverName = cfg.Driver
		in.ConnectString = cfg.Connection
		in.DbSpecified = cfg.DBSpecified
	}
	return in, nil
}

const (
	configFileDefaultPath             = "config/connection_config.json"
	configFilePathEnvironmentVariable = "CONNECTION_CONFIG_PATH"
)

func getLocalConfigPath() string {
	configFilePath := os.Getenv(configFilePathEnvironmentVariable)
	if configFilePath == "" {
		configFilePath = configFileDefaultPath
	}
	return configFilePath
}

func LoadConfiguration(file string) (*Config, error) {
	//Loads a default config from adapter_config in case a custom adapter isn't provided by the client.
	//DriverName, ConnectionString, and dbSpecified can be configured in the file. Defaults to 'file' mode.
	c := new(Config)
	configFile, err := os.Open(file)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) { // allow config file not exists
			return c, nil
		}
		return nil, err
	}
	decoder := json.NewDecoder(configFile)
	if err := decoder.Decode(c); err != nil {
		return nil, err
	}
	re := regexp.MustCompile(`\$\b(\w*)\b`)
	c.Connection = re.ReplaceAllStringFunc(c.Connection, func(s string) string {
		return os.Getenv(strings.TrimPrefix(s, `$`))
	})
	return c, nil
}

type Config struct {
	Driver      string
	Connection  string
	Enforcer    string
	DBSpecified bool
}
