package server

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
)

type Config struct {
	ServerPort  uint16            `json:ServerPort`
	AuthMethods []uint8           `json:AuthMethods`
	UsrPwdPairs map[string]string `json:UsrPwdPairs`
	UseTls      bool              `json:UseTls`
}

var (
	ErrReadCfgFile    = errors.New("Failed to read config file: it doesn't exist or unreadable.")
	ErrParseCfgString = errors.New("Invalid JSON format of config string.")
)

func NewConfig(cfgFile string) (c *Config, err error) {
	var (
		f  *os.File
		fc []byte
	)

	if f, err = os.OpenFile(cfgFile, os.O_RDONLY, os.FileMode(0666)); err != nil {
		return nil, ErrReadCfgFile
	}

	defer f.Close()

	if fc, err = ioutil.ReadAll(f); err != nil {
		return nil, ErrReadCfgFile
	}

	c = &Config{}
	if err = json.Unmarshal(fc, c); err != nil {
		return nil, ErrParseCfgString
	}

	return c, nil
}

func (c *Config) AuthUnPwd(un string, pwd string) bool {
	if up, ok := c.UsrPwdPairs[un]; ok {
		return up == pwd
	}

	return false
}
