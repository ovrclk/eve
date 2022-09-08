package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/gosuri/uitable"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ovrclk/eve/logger"
	"github.com/ovrclk/eve/util/fsutil"
)

var (
	defaultStateDir = ".akash"
	globalFlags     *GlobalFlags
)

type GlobalFlags struct {
	Path         string
	StateDirName string
}

func init() {
	globalFlags = &GlobalFlags{}
	viper.SetConfigName(".eve")
	viper.SetConfigType("yaml") // required if the config file does not have the extension in the name

	// path to look for the config file in
	viper.AddConfigPath(".")
	viper.AddConfigPath(os.ExpandEnv("$HOME"))
	viper.AddConfigPath("/etc/eve")
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			logger.Debug("init: No config file found")
			// Config file not found; ignore error
		} else {
			// Config file was found but another error was produced
			panic(fmt.Errorf("fatal error reading config file: %s", err))
		}
	}
}

// NewDeploy creates a new command that deploys the given application
func NewRootCMD(ctx context.Context, cancel context.CancelFunc) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:               "eve",
		Short:             "Eve is a tool that simplifies deploying applications on Akash",
		Long:              "",
		Example:           "",
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
	}

	rootCmd.PersistentFlags().StringVar(&globalFlags.Path, "path", "", "Path to the project, it defaults to the current directory")
	rootCmd.PersistentFlags().StringVar(&globalFlags.StateDirName, "state-dir", defaultStateDir, "Path to the state directory relative to the project path")

	rootCmd.AddCommand(
		NewInit(ctx, cancel),
		NewActions(ctx, cancel),
		NewPack(ctx, cancel),
		NewStatus(ctx, cancel),
		NewDeploy(ctx, cancel),
		NewPublish(ctx, cancel),
		NewLogs(ctx, cancel),
		NewSDL(ctx, cancel),
		NewDeploy2Cmd(ctx, cancel),
	)
	return rootCmd
}

func NewInit(ctx context.Context, cancel context.CancelFunc) *cobra.Command {
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize eve in the current directory",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Initializing Eve")
		},
	}
	return initCmd
}

// NewLogs creates a new command that logs the output of the given command
func NewLogs(ctx context.Context, cancel context.CancelFunc) *cobra.Command {
	logsCmd := &cobra.Command{
		Use:   "logs",
		Short: "View the logs of your application",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := runLogs(ctx, cancel); err != nil {
				return err
			}
			return nil
		},
	}
	return logsCmd
}

func runLogs(ctx context.Context, cancel context.CancelFunc) error {
	provider, err := readvar("PROVIDER")
	if err != nil {
		return err
	}

	dseq, err := readvar("DSEQ")
	if err != nil {
		return err
	}

	c := []string{"provider", "logs", "--provider", provider, "--dseq", dseq, "--from", "deploy"}

	logger.Debug("runLogs: ", c)

	cmd := exec.CommandContext(ctx, "akash", c...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("runLogs error: ", err)
		return err
	}

	logger.Debug("runLogs output: ", string(out))

	return nil
}

func NewActions(ctx context.Context, cancel context.CancelFunc) *cobra.Command {
	actionsCmd := &cobra.Command{
		Use:   "actions",
		Short: "Manage your Github actions",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Actions")
		},
	}
	return actionsCmd
}

func NewStatus(ctx context.Context, cancel context.CancelFunc) *cobra.Command {
	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "View the status of your application",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := runStatus(ctx, cancel); err != nil {
				return err
			}
			return nil
		},
	}
	return statusCmd
}

func runStatus(ctx context.Context, cancel context.CancelFunc) (err error) {
	var provider, dseq string

	if provider, err = readvar("PROVIDER"); err != nil {
		return err
	}

	if dseq, err = readvar("DSEQ"); err != nil {
		return err
	}

	// fetch the lease status
	c := []string{"provider", "lease-status", "--provider", provider, "--dseq", dseq, "--from", "deploy"}
	logger.Debug("runStatus: ", c)

	cmd := exec.CommandContext(ctx, "akash", c...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("runStatus error: ", err)
		return err
	}
	logger.Debug("runStatus output: ", string(out))

	// unmarshal the lease status JSON into a map
	var result map[string]interface{}
	if err := json.Unmarshal(out, &result); err != nil { // unmarshal the output into a map
		return errors.Wrapf(err, "failed to unmarshal output")
	}
	logger.Debugf("+%v", result)

	tab := uitable.New().AddRow("SERVICE", "AVAIABLE", "END_POINTS")
	for _, v := range result["services"].(map[string]interface{}) {
		svc := v.(map[string]interface{})
		tab.AddRow(svc["name"], svc["available"], svc["uris"])
	}
	fmt.Println(tab.String())
	return nil
}

// varExists checks if the given variable exists in the environment
func varExists(name string) bool {
	if val, err := readvar(name); err == nil && val != "" { // if the variable exists and is not empty return true
		return true
	}
	return false
}

// readvar reads a variable from the state file
func readvar(name string) (string, error) {
	logger.Debug("readvar: ", name)
	p := path.Join(globalFlags.Path, globalFlags.StateDirName, name) // path to the variable
	if !fsutil.FileExists(p) {
		return "", errors.Errorf("file missing: %s", p)
	} // if the file does not exist, return an error

	b, err := os.ReadFile(p)
	if err != nil {
		return "", errors.Wrapf(err, "failed to read file %s", p)
	}
	return strings.TrimSuffix(string(b), "\n"), nil
}

// writevar writes a variable to the state file
func writevar(name, value string) error {
	logger.Debug("writevar: ", name)
	p := path.Join(globalFlags.Path, globalFlags.StateDirName, name) // path to the variable

	if err := os.WriteFile(p, []byte(value), 0644); err != nil {
		logger.Error("writevar error: ", err)
		return errors.Wrapf(err, "failed to write file %s", p)
	}
	return nil
}

func stringArrayHelp(name string) string {
	return fmt.Sprintf("\nRepeat for each %s in order (comma-separated lists not accepted)", name)
}
