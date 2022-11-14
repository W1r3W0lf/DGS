package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/pelletier/go-toml"
)

type UserConfig struct {
	Name     string                // User Name
	RepoPath string                //
	Repos    map[string]Repository //
}

func startUser(reader *bufio.Reader) UserConfig {
	if _, err := os.Stat("./dgs.toml"); err == nil {
		return loadConfig()
	} else {
		return setupConfig(reader)
	}
}

func setupConfig(reader *bufio.Reader) UserConfig {

	var config UserConfig

	fmt.Println("New UserName (No spaces):")
	var name string
	_, err := fmt.Fscanf(reader, "%s", &name)
	handleError(err, "Error getting Username")

	config.Name = name

	config.RepoPath = "./repos/"
	err = os.Mkdir(config.RepoPath, os.FileMode(0777))
	handleError(err, "Error creating repo directory")

	config.Repos = make(map[string]Repository, 0)

	configFile, err := os.Create("./dgs.toml")
	handleError(err, "Error creating repo config")

	m, err := toml.Marshal(config)
	handleError(err, "Error Marshaling config")

	configFile.Write(m)

	return config
}

func loadConfig() UserConfig {
	var config UserConfig

	configFile, err := os.Open("./dgs.toml")
	handleError(err, "Error can't read dgs.toml")
	defer configFile.Close()

	fileInfo, err := configFile.Stat()
	handleError(err, "Error getting config file size")

	buffer := make([]byte, fileInfo.Size())

	_, err = configFile.Read(buffer)
	handleError(err, "Error reading from config file")

	err = toml.Unmarshal(buffer, &config)
	handleError(err, "Error Unmarshaling config file")

	return config
}

func writeConfig(config UserConfig) {

	m, err := toml.Marshal(config)
	if err != nil {
		fmt.Fprint(os.Stderr, "Error Marshaling config")
		panic(err)
	}

	ioutil.WriteFile("./dgs.toml", m, 0644)

}
