package cmd

import (
	"context"
	"fmt"
	"os/exec"
	"path"

	"github.com/ovrclk/eve/logger"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type DeployFlags struct {
	DSEQ      string
	Provider  string
	NoPack    bool
	NoUpdate  bool
	NoPublish bool
	Image     string

	PackFlags *PackFlags
	*PublishFlags
}

var deployFlags *DeployFlags

func NewDeploy(ctx context.Context, cancel context.CancelFunc) *cobra.Command {
	deployFlags = &DeployFlags{PackFlags: &PackFlags{}, PublishFlags: &PublishFlags{}}
	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy your application",
		RunE: func(cmd *cobra.Command, args []string) error {
			dseq, err := readvar("DSEQ")
			if err != nil {
				return err
			}

			provider, err := readvar("PROVIDER")
			if err != nil {
				return err
			}
			if !deployFlags.NoPack {
				if err := runPack(ctx, cancel, deployFlags.Image, deployFlags.PackFlags); err != nil {
					return err
				}
			}

			if !deployFlags.NoPublish {
				if err := runPublish(ctx, cancel, deployFlags.Image, deployFlags.PublishFlags); err != nil {
					return err
				}
			}

			if !deployFlags.NoUpdate {
				//sdlSource := path.Join(globalFlags.Path, "sdl.yml")
				version, err := readvar("VERSION")
				if err != nil {
					return err
				}
				sdlSource := path.Join(globalFlags.Path, "sdl.yml")
				if err := runSDL(ctx, cancel, sdlSource, &SDLFlags{}); err != nil {
					return err
				}
				sdltarget := path.Join(globalFlags.Path, globalFlags.StateDirName, "cache", "sdl."+version+".yml")

				if err = runUpdateDeployment(ctx, cancel, dseq, sdltarget); err != nil {
					return err
				}

				if err = runProviderSendManifest(ctx, cancel, provider, dseq, sdltarget); err != nil {
					return err
				}
			}

			return nil
		},
	}
	cmd.AddCommand(NewSendManifestCMD(ctx, cancel), NewUpdateDeploymentCMD(ctx, cancel))
	// cmd.Flags().StringVar(&deployFlags.DSEQ, "dseq", "", "The dseq of the application")
	// cmd.Flags().StringVar(&deployFlags.Provider, "provider", "", "The provider of the application")
	cmd.Flags().BoolVar(&deployFlags.NoPack, "no-pack", false, "Do not pack the application")
	cmd.Flags().BoolVar(&deployFlags.NoUpdate, "no-update", false, "Do not update the deployment")
	cmd.Flags().BoolVar(&deployFlags.NoPublish, "no-publish", false, "Do not publish the deployment")
	cmd.Flags().StringVar(&deployFlags.Image, "image", "", "The image to use for the deployment")

	bindPackFlags(deployFlags.PackFlags, cmd)
	bindPublishFlags(deployFlags.PublishFlags, cmd)
	return cmd
}

func NewDeployCreateCMD(ctx context.Context, cancel context.CancelFunc) *cobra.Command {
	deployCreateCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new deployment",
		Run: func(cmd *cobra.Command, args []string) {
			if err := runCreateDeployment(ctx, cancel, ""); err != nil {
				fmt.Println("error: ", err)
				return
			}
		},
	}
	return deployCreateCmd
}

func runCreateDeployment(ctx context.Context, cancel context.CancelFunc, sdlPath string) error {
	return nil
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

			if err := runProviderSendManifest(ctx, cancel, provider, dseq, ""); err != nil {
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

			if err = runUpdateDeployment(ctx, cancel, dseq, ""); err != nil {
				fmt.Println("error: ", err)
				return
			}
		},
	}
	return updateDeploymentCmd
}
func runUpdateDeployment(ctx context.Context, cancel context.CancelFunc, dseq string, sdlPath string) error {
	c := []string{"tx", "deployment", "update", "--dseq", dseq, "--from", "deploy", "-y", sdlPath}
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

func runProviderSendManifest(ctx context.Context, cancel context.CancelFunc, provider, dseq, sdlPath string) error {
	c := []string{"provider", "send-manifest", "--provider", provider, "--dseq", dseq, "--from", "deploy", sdlPath}
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
