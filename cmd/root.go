package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"os/exec"
	"path"
	"strings"

	"github.com/gosuri/eve/logger"
	"github.com/gosuri/eve/util/fsutil"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	defaultStateDir = ".akash"
	globalFlags     *GlobalFlags
)

type GlobalFlags struct {
	Path     string
	StateDir string
}

func init() {
	globalFlags = &GlobalFlags{}
}

func NewRootCMD(ctx context.Context, cancel context.CancelFunc) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "eve",
		Short: "Eve is a tool that simplifies deploying applications on Akash",
	}

	rootCmd.PersistentFlags().StringVarP(&globalFlags.Path, "path", "p", ".", "Path to the project")
	rootCmd.PersistentFlags().StringVarP(&globalFlags.StateDir, "state-dir", "s", defaultStateDir, "Path to the state directory")

	rootCmd.AddCommand(
		NewInitCMD(ctx, cancel),
		NewActionsCMD(ctx, cancel),
		NewPackCMD(ctx, cancel),
		NewStatusCMD(ctx, cancel),
		NewDeployCMD(ctx, cancel),
	)
	return rootCmd
}

func NewInitCMD(ctx context.Context, cancel context.CancelFunc) *cobra.Command {
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize eve in the current directory",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Initializing Eve")
		},
	}
	return initCmd
}

func NewActionsCMD(ctx context.Context, cancel context.CancelFunc) *cobra.Command {
	actionsCmd := &cobra.Command{
		Use:   "actions",
		Short: "Manage your Github actions",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Actions")
		},
	}
	return actionsCmd
}

func NewStatusCMD(ctx context.Context, cancel context.CancelFunc) *cobra.Command {
	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "View the status of your application",
		Run: func(cmd *cobra.Command, args []string) {
			if err := runStatus(ctx, cancel); err != nil {
				fmt.Println("error: ", err)
				return
			}
		},
	}
	return statusCmd
}

func NewDeployCMD(ctx context.Context, cancel context.CancelFunc) *cobra.Command {
	deployCmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy your application",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Deploying Eve")

			dseq, err := readvar("DSEQ")
			if err != nil {
				fmt.Println("error: ", err)
				return
			}

			provider, err := readvar("PROVIDER")
			if err != nil {
				fmt.Println("error: ", err)
				return
			}

			if err = runUpdateDeployment(ctx, cancel, dseq); err != nil {
				fmt.Println("error: ", err)
				return
			}

			if err = runProviderSendManifest(ctx, cancel, provider, dseq); err != nil {
				fmt.Println("error: ", err)
				return
			}

		},
	}
	deployCmd.AddCommand(NewSendManifestCMD(ctx, cancel), NewUpdateDeploymentCMD(ctx, cancel))
	return deployCmd
}

func NewSendManifestCMD(ctx context.Context, cancel context.CancelFunc) *cobra.Command {
	updateManifestCmd := &cobra.Command{
		Use:   "update-manifest",
		Short: "Update the manifest of your application",
		Run: func(cmd *cobra.Command, args []string) {
			dseq, err := readvar("DSEQ")
			if err != nil {
				fmt.Println("error: ", err)
			}

			provider, err := readvar("PROVIDER")
			if err != nil {
				fmt.Println("error: ", err)
			}

			if err := runProviderSendManifest(ctx, cancel, provider, dseq); err != nil {
				fmt.Println("error: ", err)
				return
			}

			fmt.Println("Updating manifest")
		},
	}
	return updateManifestCmd
}

func NewUpdateDeploymentCMD(ctx context.Context, cancel context.CancelFunc) *cobra.Command {
	updateDeploymentCmd := &cobra.Command{
		Use:   "update-deployment",
		Short: "Update the deployment of your application",
		Run: func(cmd *cobra.Command, args []string) {
			dseq, err := readvar("DSEQ")
			if err != nil {
				fmt.Println("error: ", err)
				return
			}

			if err = runUpdateDeployment(ctx, cancel, dseq); err != nil {
				fmt.Println("error: ", err)
				return
			}
		},
	}
	return updateDeploymentCmd
}

func runStatus(ctx context.Context, cancel context.CancelFunc) error {
	provider, err := readvar("PROVIDER")
	if err != nil {
		return err
	}

	dseq, err := readvar("DSEQ")
	if err != nil {
		return err
	}

	c := []string{"provider", "lease-status", "--provider", provider, "--dseq", dseq, "--from", "deploy"}

	logger.Debug("runStatus: ", c)
	cmd := exec.CommandContext(ctx, "akash", c...)
	out, err := cmd.CombinedOutput()
	fmt.Println(string(out))
	if err != nil {
		fmt.Println("runStatus error: ", err)
		return err
	}
	return nil
}

func runUpdateDeployment(ctx context.Context, cancel context.CancelFunc, dseq string) error {
	c := []string{"tx", "deployment", "update", "--dseq", dseq, "--from", "deploy", "-y", "sdl.yml"}
	logger.Debug("runUpdateDeployment: ", c)
	cmd := exec.CommandContext(ctx, "akash", c...)
	out, err := cmd.CombinedOutput()
	fmt.Println(string(out))
	if err != nil {
		logger.Error("runUpdate Deploy error: ", err)
		return err
	}
	return nil
}

func runProviderSendManifest(ctx context.Context, cancel context.CancelFunc, provider string, dseq string) error {
	c := []string{"provider", "send-manifest", "--provider", provider, "--dseq", dseq, "--from", "deploy", "sdl.yml"}
	logger.Debug("runProviderSendManifest", c)
	cmd := exec.CommandContext(ctx, "akash", c...)
	out, err := cmd.CombinedOutput()
	fmt.Println(string(out))
	if err != nil {
		logger.Error("runProviderSendManifest error: ", err)
		return errors.Wrapf(err, "Unable to upload manifest to provider %s, for dseq %s", provider, dseq)
	}
	return nil
}

func varExists(name string) bool {
	if val, err := readvar(name); err == nil && val != "" { // if the variable exists and is not empty return true
		return true
	}
	return false
}

func readvar(name string) (string, error) {
	logger.Debug("readvar: ", name)
	p := path.Join(globalFlags.Path, globalFlags.StateDir, name) // path to the variable
	if !fsutil.FileExists(p) {
		return "", errors.Errorf("file missing: %s", p)
	} // if the file does not exist, return an error

	b, err := ioutil.ReadFile(p)
	if err != nil {
		return "", errors.Wrapf(err, "failed to read file %s", p)
	}
	return strings.TrimSuffix(string(b), "\n"), nil
}

func writevar(name, value string) error {
	logger.Debug("writevar: ", name)
	p := path.Join(globalFlags.Path, globalFlags.StateDir, name) // path to the variable

	if err := ioutil.WriteFile(p, []byte(value), 0644); err != nil {
		logger.Error("writevar error: ", err)
		return errors.Wrapf(err, "failed to write file %s", p)
	}
	return nil
}

func stringArrayHelp(name string) string {
	return fmt.Sprintf("\nRepeat for each %s in order (comma-separated lists not accepted)", name)
}
