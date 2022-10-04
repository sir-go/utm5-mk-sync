package main

import (
	"os"

	"github.com/BurntSushi/toml"
)

type (
	MkConnParams struct {
		Addr     string  `toml:"addr"`
		Username string  `toml:"username"`
		Password string  `toml:"password"`
		Coef     float32 `toml:"coef"`
	}

	Config struct {
		Shape struct {
			Devices            []MkConnParams `toml:"devices"`
			IgnoreIds          []string       `toml:"ignore_ids"`
			RebalanceThreshold float32        `toml:"rebalance_threshold"`
		} `toml:"shape"`
		Firewall MkConnParams `toml:"firewall"`
		Acl      struct {
			ListAllow string `toml:"list_allow"`
			ListDeny  string `toml:"list_deny"`
		} `toml:"acl"`
		PPPoE   []MkConnParams `toml:"pppoe"`
		Billing struct {
			GetUsersQuery string `toml:"get_users_query"`
			DB            struct {
				Host      string `toml:"host"`
				Username  string `toml:"username"`
				Password  string `toml:"password"`
				DbNameTih string `toml:"dbname_tih"`
				DbNameKor string `toml:"dbname_kor"`
			} `toml:"db"`
			BwMultiplier float64 `toml:"bw_multiplier"`
		} `toml:"billing"`
		MQ struct {
			Url   string `toml:"url"`
			Queue string `toml:"queue"`
		} `toml:"mq"`
	}
)

func LoadConfig(confpath string) *Config {
	conf := new(Config)
	file, err := os.Open(confpath)
	defer func() {
		if file == nil {
			return
		}
		eh(file.Close())
	}()
	eh(err)

	_, err = toml.DecodeFile(confpath, &conf)
	eh(err)

	return conf
}
