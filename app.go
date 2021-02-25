package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-redis/redis/v8"
	"gopkg.in/yaml.v3"
)

func main() {
	startCommand := flag.NewFlagSet("start", flag.ExitOnError)
	initCommand := flag.NewFlagSet("init", flag.ExitOnError)


	pathToYaml := *flag.String("path", "", " Where in the git repository is the primary pannugit.yaml.")
	repository := *flag.String("repository", "", " The git repository you want to pull.")

	flag.Parse()

	fmt.Println(startCommand.Parsed())
	fmt.Println(initCommand.Parsed())

	if len(flag.Args()) < 1 {
		fmt.Println("start or init subcommand is required")
		os.Exit(1)
	}

	if len(flag.Args()) > 2 {
		fmt.Println("Too many arguments")
		os.Exit(1)
	}

	if len(flag.Args()) == 1 {
		subcommand := flag.Arg(1)
		if subcommand != "start" {
			fmt.Println("--help to help yourself")
			os.Exit(1)
			return
		}
		startFromMemory()
		return
	}

	if len(flag.Args()) == 2 {
		subcommand := flag.Arg(1)
		pathArgument := flag.Arg(2)
		if subcommand == "start" {
			startFromRepo(pathArgument, pathToYaml)
			return
		}
		if subcommand == "init" {
			initialize(pathArgument, repository, pathToYaml)
			return
		}
	}
}

func startFromMemory() {
	fmt.Println("Start from memory")
}

func startFromRepo(pathInFilesystem string, pathToYaml string) {
	fmt.Println("Start from Repository")
}

func initialize(pathInFilesystem string, repository string, pathToYaml string) {
	fmt.Println("Initialize")
}

func runPocSetup() {
	initializeRedisClient()
	directory := "test"
	githubRepository := "org/example"
	repoPath, pathErr := filepath.Abs(filepath.Join(filepath.Dir(os.Args[0]), directory, githubRepository))
	CheckIfError(pathErr)
	//cloneRepositoryToSubdirectory(repoPath, githubRepository)

	pannugitConfigName := "pannugit.yaml"
	pannnugitConfigPath := filepath.Join(repoPath, pannugitConfigName)
	config, configErr := readPannugitYaml(pannnugitConfigPath)
	CheckIfError(configErr)
	storePannugitConfig(config)

	findAllServiceYamlsFromConfig()
	dockerComposes := createDockerComposesForAllServices()
	fmt.Println(dockerComposes)

	example := dockerComposes[0]
	runDockerComposeUp(example)
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
func readPannugitYaml(pathToFile string) (*pannugitConf, error) {
	yamlFile, err := ioutil.ReadFile(pathToFile)
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
func findAllServiceYamlsFromConfig() []string {
	config, err := getPannugitConfig()
	CheckIfError(err)

	path := filepath.Join(config.StorePath, config.WatchPath)
	fmt.Println(path)

	paths := []string{}

	walkErr := filepath.Walk(path, func(pathToFile string, info os.FileInfo, err error) error {
		CheckIfError(err)
		if strings.HasSuffix(info.Name(), ".pannugit.yaml") {
			paths = append(paths, pathToFile)
		}
		return nil
	})
	CheckIfError(walkErr)

	return paths
}

type serviceConf struct {
	Override string `yaml:"override"`
}

func readServiceYaml(pathToFile string) (*serviceConf, error) {
	yamlFile, err := ioutil.ReadFile(pathToFile)
	CheckIfError(err)

	config := &serviceConf{}
	err = yaml.Unmarshal(yamlFile, config)
	CheckIfError(err)

	return config, nil
}

type dockerComposeStorage struct {
	serviceConfPath string
	dockerComposeOverridePath string
	dockerComposeOverride string
}

// Construct docker-compose.ymls from app instructions
func createDockerComposesForAllServices() []dockerComposeStorage {
	serviceYamlPaths := findAllServiceYamlsFromConfig()

	serviceDockerComposes := []dockerComposeStorage{}

	for i := 0; i < len(serviceYamlPaths); i++ {
		pathToFile := serviceYamlPaths[i]

		serviceConfig, err := readServiceYaml(pathToFile)
		CheckIfError(err)

		pathToOverride := filepath.Join(filepath.Dir(pathToFile), serviceConfig.Override)

		yamlFile, err := ioutil.ReadFile(pathToOverride)
		CheckIfError(err)

		dockerComposes := dockerComposeStorage{
			serviceConfPath: pathToFile,
			dockerComposeOverridePath: pathToOverride,
			dockerComposeOverride: string(yamlFile),
		}
		serviceDockerComposes = append(serviceDockerComposes, dockerComposes)
	}

	return serviceDockerComposes
}

// Run docker-compose.yml (dco up)
func runDockerComposeUp(storage dockerComposeStorage) {
	overrideFilePath := storage.dockerComposeOverridePath
	executionDir := filepath.Dir(overrideFilePath)
	cmd := exec.Command("docker-compose", "-f", overrideFilePath, "up", "-d")
	cmd.Dir = executionDir
	err := cmd.Run()
	CheckIfError(err)
}

// Check for repo updates (interval)
// Run dco pull && dco down && dco up

// POC READY

// Store docker-compose.yml in redis
// Run docker-compose.yml only if it changed (redis, dco pull dco down dco up).

// Add support for referencing other repositorios (interval)
// Clear old images to save space

// PROD READY

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
