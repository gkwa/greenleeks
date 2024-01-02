package greenleeks

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/jessevdk/go-flags"
	mymazda "github.com/taylormonacelli/forestfish/mymazda"
	"gopkg.in/ini.v1"
)

const (
	maxFilesErrorMessage = "too many files (%d), limit is %d"
	gitConfigFileName    = ".gitconfig"
	gitConfigUserSection = "user"
)

type AuthorInfo struct {
	Name  string
	Email string
}

var authorInfo AuthorInfo

var opts struct {
	LogFormat string `long:"log-format" choice:"text" choice:"json" default:"text" description:"Log format"`
	Verbose   []bool `short:"v" long:"verbose" description:"Show verbose debug information, each -v bumps log level"`
	RootDir   string `short:"r" long:"root" description:"Root directory" default:"."`
	MaxFiles  int    `long:"max-files" description:"Maximum number of files allowed" default:"100"`
	GitConfig string `long:"gitconfig" description:"Path to the Git configuration file" default:"~/.gitconfig"`
	logLevel  slog.Level
}

func Execute() int {
	if err := parseFlags(); err != nil {
		return 1
	}

	if err := setLogLevel(); err != nil {
		return 1
	}

	if err := setupLogger(); err != nil {
		return 1
	}

	if err := run(); err != nil {
		slog.Error("run failed", "error", err)
		return 1
	}

	return 0
}

func parseFlags() error {
	parser := flags.NewParser(&opts, flags.Default)
	parser.Usage = "[OPTIONS]"
	_, err := parser.ParseArgs(os.Args)
	return err
}

func run() error {
	var err error

	authorInfo, err = configureGitUserInfo()
	if err != nil {
		return fmt.Errorf("failed to configure git user info: %v", err)
	}

	isUnderGit, err := isUnderGitControl(opts.RootDir)
	if err != nil {
		return fmt.Errorf("failed to check if directory is under git control: %v", err)
	}

	if isUnderGit {
		slog.Info("Directory is already under git control.")
		return nil
	}

	slog.Info("Initializing git repository...")

	err = initializeGitRepository(opts.RootDir)
	if err != nil {
		return fmt.Errorf("failed to initialize git repository: %v", err)
	}

	fileCount, err := countFiles(opts.RootDir)
	if err != nil {
		return fmt.Errorf("failed to count files: %v", err)
	}

	if fileCount > opts.MaxFiles {
		return fmt.Errorf(maxFilesErrorMessage, fileCount, opts.MaxFiles)
	}

	err = addAllFiles(opts.RootDir)
	if err != nil {
		return fmt.Errorf("failed to add all files: %v", err)
	}

	err = commit(opts.RootDir, "Boilerplate")
	if err != nil {
		return fmt.Errorf("failed to commit: %v", err)
	}

	slog.Info("Git initialization successful.")

	return nil
}

func isUnderGitControl(rootDir string) (bool, error) {
	_, err := git.PlainOpenWithOptions(rootDir, &git.PlainOpenOptions{DetectDotGit: true})
	if err == nil {
		return true, nil
	} else if err == git.ErrRepositoryNotExists || err == git.ErrWorktreeNotProvided {
		return false, nil
	} else {
		return false, fmt.Errorf("failed to open repository: %v", err)
	}
}

func initializeGitRepository(rootDir string) error {
	_, err := git.PlainInit(rootDir, false)
	if err != nil {
		return fmt.Errorf("failed to initialize git repository: %v", err)
	}
	return nil
}

func addAllFiles(rootDir string) error {
	repo, err := git.PlainOpen(rootDir)
	if err != nil {
		return fmt.Errorf("failed to open repository: %v", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %v", err)
	}

	_, err = worktree.Add(".")
	if err != nil {
		return fmt.Errorf("failed to add all files: %v", err)
	}

	return nil
}

func commit(rootDir, message string) error {
	repo, err := git.PlainOpen(rootDir)
	if err != nil {
		return fmt.Errorf("failed to open repository: %v", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %v", err)
	}

	author := &object.Signature{
		Name:  authorInfo.Name,
		Email: authorInfo.Email,
		When:  time.Now(),
	}

	_, err = worktree.Commit(message, &git.CommitOptions{
		Author: author,
	})
	if err != nil {
		return fmt.Errorf("failed to commit: %v", err)
	}

	return err
}

func countFiles(rootDir string) (int, error) {
	fileCount := 0
	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			fileCount++
			if fileCount > opts.MaxFiles {
				return fmt.Errorf(maxFilesErrorMessage, fileCount, opts.MaxFiles)
			}
		}
		return nil
	})
	return fileCount, err
}

func configureGitUserInfo() (AuthorInfo, error) {
	gitConfigPath, err := mymazda.ExpandTilde(opts.GitConfig)
	if err != nil {
		panic(err)
	}

	ai := AuthorInfo{
		Name:  "Your Name",
		Email: "your.email@example.com",
	}

	config, err := readGitConfig(gitConfigPath)
	if err != nil {
		return AuthorInfo{}, err
	}

	name := config.Section(gitConfigUserSection).Key("name").String()
	email := config.Section(gitConfigUserSection).Key("email").String()

	if name != "" {
		ai.Name = name
	}

	if email != "" {
		ai.Email = email
	}

	return ai, nil
}

func readGitConfig(gitConfigPath string) (*ini.File, error) {
	cfg, err := ini.Load(gitConfigPath)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}
