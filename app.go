package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-redis/redis/v8"
	"gopkg.in/yaml.v3"
)

func main() {
	initializeRedisClient()
	directory := "test"
	githubRepository := "org/example"
	repoPath, pathErr := filepath.Abs(filepath.Join(filepath.Dir(os.Args[0]), directory, githubRepository))
	CheckIfError(pathErr)
	//cloneRepositoryToSubdirectory(repoPath, githubRepository)

	pannugitConfigName := "pannugit.yaml"
	config, configErr := readPannugitConfig(repoPath, pannugitConfigName)
	CheckIfError(configErr)
	storePannugitConfig(config)

	findAllPannugitFilesFromConfig()
}

// Pull repo
func cloneRepositoryToSubdirectory(repoStorePath string, githubRepository string) {
	git.PlainClone(repoStorePath, false, &git.CloneOptions{
    URL:      "https://github.com/" + githubRepository,
    Progress: os.Stdout,
	})
}

type pannugitConf struct {
	ConfigFilePath string `yaml:"configFilePath"`
	Ref string `yaml:"ref"`
	WatchPath string `yaml:"watchPath"`
	StorePath string `yaml:"storePath"`
}

// Find own instructions in repo
func readPannugitConfig(repoPath string, pannugitConfigName string) (*pannugitConf, error) {
	configPath := filepath.Join(repoPath, pannugitConfigName)
	yamlFile, err := ioutil.ReadFile(configPath)
	CheckIfError(err)

	config := &pannugitConf{}
	err = yaml.Unmarshal(yamlFile, config)
	CheckIfError(err)

	return config, nil
}

var rdb *redis.Client
var ctx = context.Background()

func initializeRedisClient() {
	rdb = redis.NewClient(&redis.Options{
		Addr: ":6379",
	})
}

// Persist own instructions (redis)
func storePannugitConfig(conf *pannugitConf) {
	yamlString, err := yaml.Marshal(conf)
	CheckIfError(err)

	rdb.Set(ctx, "pannugitConfig", yamlString, 0)
}

// Get current data easily
func getPannugitConfig() (*pannugitConf, error) {
	configData := rdb.Get(ctx, "pannugitConfig")
	storedData, err := configData.Bytes()
	CheckIfError(err)
	
	config := &pannugitConf{}
	yaml.Unmarshal(storedData, config)
	return config, nil
}

// Use "pannu instructions" to find more "app instructions" (e.g. all in subdirectory)
func findAllPannugitFilesFromConfig() {
	config, err := getPannugitConfig()
	CheckIfError(err)

	path := filepath.Join(config.StorePath, config.WatchPath)
	fmt.Println(path)

	filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		CheckIfError(err)
		if strings.HasSuffix(info.Name(), ".pannugit.yaml") {
			fmt.Println(info.Name()) // WIP, store path to file
		}
		return nil
	})
}

// Construct docker-compose.yml from app instructions
// Run docker-compose.yml (dco up)

// Check for repo updates (interval)
// Run dco pull && dco down && dco up

// Store docker-compose.yml in redis
// Run docker-compose.yml only if it changed (redis, dco pull dco down dco up).

// Add support for referencing other repositorios (interval)
// Clear old images to save space

// Store repository in key value data, (key = repo, value = latest hash)
// If hash changes do pull, otherwise ignore
func getLatestRemoteCommitHash(repoPath string) string {
	repo, err := git.PlainOpen(repoPath)
	CheckIfError(err)

	ref, err := repo.Head()
	CheckIfError(err)

	hash := ref.Hash().String()

	return hash
}
