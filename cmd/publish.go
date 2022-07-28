package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"

	"time"

	"github.com/gosuri/eve/logger"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var publishFlags *PublishFlags

// PublishFlags is the set of flags used by the publish command
type PublishFlags struct {
	Version  string
	SkipSave bool
}

// PublishCmd is the command to publish the image
func NewPublish(ctx context.Context, cancel context.CancelFunc) *cobra.Command {
	publishFlags = &PublishFlags{}

	publishCmd := &cobra.Command{
		Use:   "publish <image>",
		Short: "Publish and version your image",
		RunE: func(cmd *cobra.Command, args []string) error {
			image := ""
			if len(args) > 0 {
				image = args[0]
			}
			return runPublish(ctx, cancel, image, publishFlags)
		},
	}
	bindPublishFlags(publishFlags, publishCmd)
	return publishCmd
}

func bindPublishFlags(flags *PublishFlags, cmd *cobra.Command) {
	cmd.Flags().StringVarP(&flags.Version, "version", "v", "", "Version of the image")
	cmd.Flags().BoolVar(&flags.SkipSave, "skip-save", false, "Skip saving the version to the state directory")
}

func runPublish(ctx context.Context, cancel context.CancelFunc, image string, flags *PublishFlags) (err error) {
	if image == "" {
		if image, err = readvar("IMAGE"); err != nil {
			return errors.Wrap(err, "failed to read IMAGE variable")
		}
	}
	// if the version is not given, set the version to the current time
	// in ISO8601 format and write the version to the state directory
	if flags.Version == "" {
		flags.Version = fmt.Sprint(time.Now().Unix())

	}
	if !flags.SkipSave && writevar("VERSION", flags.Version) != nil {
		return errors.Wrap(err, "failed to write VERSION variable")
	}

	// push the latest version image to the registry
	if err := dockerPush(ctx, cancel, image); err != nil {
		return errors.Wrap(err, "failed to push image: "+image)
	}

	// tag the image with the version tag
	c := []string{"tag", image, image + ":" + flags.Version}
	logger.Debugf("runPublish: running command: docker %v", c)
	cmd := exec.CommandContext(ctx, "docker", c...)
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "failed to tag image")
	}

	// push the tagged image to the registry
	if err := dockerPush(ctx, cancel, image+":"+flags.Version); err != nil {
		return errors.Wrap(err, "failed to push image: "+image+":"+flags.Version)
	}
	return nil
}

// dockerPush pushes the image to the registry
func dockerPush(ctx context.Context, cancel context.CancelFunc, image string) (err error) {
	c := []string{"push", image}
	logger.Debugf("dockerPush: running command: docker %v", c)
	cmd := exec.CommandContext(ctx, "docker", c...)
	r, _ := cmd.StdoutPipe()
	err = cmd.Start()
	if err != nil {
		return errors.Wrap(err, "error starting push")
	}
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		fmt.Println(scanner.Text())
	}
	if err := cmd.Wait(); err != nil {
		return errors.Wrap(err, "error waiting for push")
	}
	return nil
}
