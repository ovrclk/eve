package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/gosuri/eve/util/fsutil"
	"github.com/spf13/cobra"
)

var homeDir = ".eve"

const stateDir = ".akash"

func NewRootCMD(ctx context.Context, cancel context.CancelFunc) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "eve",
		Short: "Eve is a tool that simplifies deploying applications on Akash",
	}

	rootCmd.PersistentFlags().StringVarP(&homeDir, "home", "d", "", "Home directory, defaults to "+homeDir)

	rootCmd.AddCommand(
		NewInitCMD(ctx, cancel),
		NewActionsCMD(ctx, cancel),
		NewPackCMD(ctx, cancel),
		NewStatusCMD(ctx, cancel),
		NewDeployCMD(ctx, cancel),
	)
	return rootCmd
}

func NewPackCMD(ctx context.Context, cancel context.CancelFunc) *cobra.Command {
	var image string
	packCmd := &cobra.Command{
		Use:   "pack",
		Short: "Pack your project into a container using buildpacks",
		Run: func(cmd *cobra.Command, args []string) {
			if err := runPack(ctx, cancel, image); err != nil {
				fmt.Println("error: ", err)
				return
			}
		},
	}
	packCmd.Flags().StringVarP(&image, "image", "i", "", "Image to use for the container")
	return packCmd
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
	cmd := exec.CommandContext(ctx, "akash", c...)
	out, err := cmd.CombinedOutput()
	fmt.Println(string(out))
	if err != nil {
		fmt.Println("runUpdate Deploy error: ", err)
		return err
	}
	return nil
}

func runProviderSendManifest(ctx context.Context, cancel context.CancelFunc, provider string, dseq string) error {
	c := []string{"provider", "send-manifest", "--provider", provider, "--dseq", dseq, "--from", "deploy", "sdl.yml"}
	cmd := exec.CommandContext(ctx, "akash", c...)
	out, err := cmd.CombinedOutput()
	fmt.Println(string(out))
	if err != nil {
		fmt.Println("error: ", err)
		return err
	}
	return nil
}

func runPack(ctx context.Context, cancel context.CancelFunc, image string) error {
	c := []string{"build", "ghcr.io/" + image, "--builder", "heroku/buildpacks:20", "--env", "NODE_ENV=production"}
	cmd := exec.CommandContext(ctx, "pack", c...)

	r, _ := cmd.StdoutPipe()
	err := cmd.Start()
	if err != nil {
		fmt.Println("error: ", err)
		return err
	}

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		fmt.Println(scanner.Text())
	}

	if err := cmd.Wait(); err != nil {

		fmt.Println("error: ", err)
		return err
	}
	return nil
}

func readvar(name string) (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	p := path.Join(wd, stateDir, name)
	if !fsutil.FileExists(p) {
		return "", fmt.Errorf("file missing: %s", p)
	}

	b, err := ioutil.ReadFile(p)
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(string(b), "\n"), nil
}
