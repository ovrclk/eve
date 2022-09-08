package cmd

import (
	"bufio"
	"context"
	"fmt"

	//"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/ovrclk/eve/logger"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var DefaultBuilder = "heroku/buildpacks:20"

// PackFlags contains the flags for the pack command
type PackFlags struct {
	Env      []string // Build-time environment variable, in the form 'VAR=VALUE' or 'VAR'.
	EnvFiles []string // Build-time environment variables file
	Builder  string   // Builder to use for building the image
}

// NewPackCommand creates a new pack command
func NewPack(ctx context.Context, cancel context.CancelFunc) *cobra.Command {
	packFlags := &PackFlags{}
	cmd := &cobra.Command{
		Use:   "pack",
		Short: "Pack your project into a container using buildpacks",
		RunE: func(cmd *cobra.Command, args []string) error {
			image := ""
			if len(args) > 0 {
				image = args[0]
			}

			if err := runPack(ctx, cancel, image, packFlags); err != nil {
				return err
			}
			return nil
		},
	}
	bindPackFlags(packFlags, cmd)
	return cmd
}

func bindPackFlags(flags *PackFlags, cmd *cobra.Command) {
	cmd.Flags().StringArrayVarP(&flags.Env, "env", "e", []string{}, "Build-time environment variable, in the form 'VAR=VALUE' or 'VAR'.\nWhen using latter value-less form, value will be taken from current\n  environment at the time this command is executed.\nThis flag may be specified multiple times and will override\n  individual values defined by --env-file."+stringArrayHelp("env")+"\nNOTE: These are NOT available at image runtime.")
	cmd.Flags().StringArrayVar(&flags.EnvFiles, "env-file", []string{}, "Build-time environment variables file\nOne variable per line, of the form 'VAR=VALUE' or 'VAR'\nWhen using latter value-less form, value will be taken from current\n  environment at the time this command is executed\nNOTE: These are NOT available at image runtime.\"")
	cmd.Flags().StringVar(&flags.Builder, "builder", "", "Builder to use for building the image")
}

func runPack(ctx context.Context, cancel context.CancelFunc, image string, packFlags *PackFlags) (err error) {
	// Check if the image name is provided if not read it from IMAGE file state directory
	if image == "" {
		image, err = readvar("IMAGE")
		if err != nil {
			return errors.Wrap(err, "failed to read IMAGE variable")
		}
	}
	// Check if env-file is specified and if so, parse it
	if len(packFlags.EnvFiles) == 0 && varExists("ENV") { // if ENV is set, use it
		envFile := path.Join(globalFlags.Path, globalFlags.StateDirName, "ENV") // path to the ENV file
		packFlags.EnvFiles = []string{envFile}
	}

	// check if the builder is provided if not read it from BUILDER file state directory
	if packFlags.Builder == "" {
		if varExists("BUILDER") {
			packFlags.Builder, _ = readvar("BUILDER")
		} else {
			// if BUILDER is not set, use the default one
			packFlags.Builder = DefaultBuilder
			// write the default builder to the state directory
			writevar("BUILDER", packFlags.Builder)
		}
	}

	// construct the buildpack command
	c := []string{"build", image, "--builder", packFlags.Builder}
	env, err := parseEnv(packFlags.EnvFiles, packFlags.Env)
	if err != nil {
		return errors.Wrap(err, "error parsing environment variables")
	}

	// add the environment variables to the command
	for k, v := range env {
		c = append(c, "--env", k+"="+v)
	}

	logger.Debugf("running pack command: %s", strings.Join(c, " "))

	// run the command with the context
	cmd := exec.CommandContext(ctx, "pack", c...)
	r, _ := cmd.StdoutPipe()
	err = cmd.Start()
	if err != nil {
		return errors.Wrap(err, "error starting pack")
	}
	// TODO: add a timeout to the context
	// scan the output and print it to the console
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		fmt.Println(scanner.Text())
	}
	// wait for the command to finish
	if err := cmd.Wait(); err != nil {
		logger.Errorf("error: %v", err)
		return errors.Wrap(err, "error waiting for pack")
	}
	return nil
}

// parseEnv parses the environment variables from the env-file and env flags
func parseEnv(envFiles []string, envVars []string) (map[string]string, error) {
	env := map[string]string{}
	for _, envFile := range envFiles {
		envFileVars, err := parseEnvFile(envFile)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse env file '%s'", envFile)
		}

		for k, v := range envFileVars {
			env[k] = v
		}
	}
	for _, envVar := range envVars {
		env = addEnvVar(env, envVar)
	}
	return env, nil
}

// parseEnvFile parses the environment variables from the path to the filename
func parseEnvFile(filename string) (map[string]string, error) {
	out := make(map[string]string)
	f, err := os.ReadFile(filepath.Clean(filename))
	if err != nil {
		return nil, errors.Wrapf(err, "open %s", filename)
	}
	for _, line := range strings.Split(string(f), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		out = addEnvVar(out, line)
	}
	return out, nil
}

// addEnvVar adds the environment variable to the map
func addEnvVar(env map[string]string, item string) map[string]string {
	arr := strings.SplitN(item, "=", 2)
	if len(arr) > 1 {
		env[arr[0]] = arr[1]
	} else {
		env[arr[0]] = os.Getenv(arr[0])
	}
	return env
}
