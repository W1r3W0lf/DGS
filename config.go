package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

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
	name, err := reader.ReadString('\n')
	if err != nil {
		fmt.Fprint(os.Stderr, "Error getting UserName", err.Error())
		panic(err)
	}
	name = strings.TrimSuffix(name, "\n")

	config.Name = name

	config.RepoPath = "./repos/"
	err = os.Mkdir(config.RepoPath, os.FileMode(0777))
	if err != nil {
		fmt.Fprint(os.Stderr, "Error creating repo directory", err.Error())
		panic(err)
	}

	config.Repos = make(map[string]Repository, 0)

	configFile, err := os.Create("./dgs.toml")
	if err != nil {
		fmt.Fprint(os.Stderr, "Error opening repo config", err.Error())
		panic(err)
	}

	m, err := toml.Marshal(config)
	if err != nil {
		fmt.Fprint(os.Stderr, "Error Marshaling config", err.Error())
		panic(err)
	}

	configFile.Write(m)

	return config
}

func loadConfig() UserConfig {
	var config UserConfig

	configFile, err := os.Open("./dgs.toml")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error can't read dgs.toml", err.Error())
	}
	defer configFile.Close()

	fileInfo, err := configFile.Stat()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error getting config file size", err.Error())
		panic(err)
	}

	buffer := make([]byte, fileInfo.Size())

	_, err = configFile.Read(buffer)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error reading from config file", err.Error())
		panic(err)
	}

	err = toml.Unmarshal(buffer, &config)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error Unmarshaling configFile", err.Error())
		panic(err)
	}

	return config
}

func writeConfig(config UserConfig) {

	/*
		configFile, err := os.Open("./dgs.toml")
		if err != nil {
			fmt.Fprint(os.Stderr, "Error opening repo config", err.Error())
			panic(err)
		}
		defer configFile.Close()
	*/

	m, err := toml.Marshal(config)
	if err != nil {
		fmt.Fprint(os.Stderr, "Error Marshaling config", err.Error())
		panic(err)
	}

	ioutil.WriteFile("./dgs.toml", m, 0644)

	/*
		_, err = configFile.Write(m)
		if err != nil {
			fmt.Fprint(os.Stderr, "Error Writting config", err.Error())
			panic(err)
		}
	*/

}
