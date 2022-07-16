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

type PublishFlags struct {
	Image    string
	Version  string
	SkipSave bool
}

func NewPublish(ctx context.Context, cancel context.CancelFunc) *cobra.Command {
	publishFlags = &PublishFlags{}

	publishCmd := &cobra.Command{
		Use:   "publish",
		Short: "Publish and version your image",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPublish(ctx, cancel, publishFlags)
		},
	}
	publishCmd.Flags().StringVarP(&publishFlags.Image, "image", "i", "", "Image to publish")
	publishCmd.Flags().StringVarP(&publishFlags.Version, "version", "v", "", "Version of the image")
	publishCmd.Flags().BoolVar(&publishFlags.SkipSave, "skip-save", false, "Skip saving the version to the state directory")
	return publishCmd
}

func runPublish(ctx context.Context, cancel context.CancelFunc, flags *PublishFlags) (err error) {
	if flags.Image == "" {
		if flags.Image, err = readvar("IMAGE"); err != nil {
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
	if err := dockerPush(ctx, cancel, flags.Image); err != nil {
		return errors.Wrap(err, "failed to push image: "+flags.Image)
	}

	// tag the image with the version tag
	c := []string{"tag", flags.Image, flags.Image + ":" + flags.Version}
	logger.Debugf("runPublish: running command: docker %v", c)
	cmd := exec.CommandContext(ctx, "docker", c...)
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "failed to tag image")
	}

	// push the tagged image to the registry
	if err := dockerPush(ctx, cancel, flags.Image+":"+flags.Version); err != nil {
		return errors.Wrap(err, "failed to push image: "+flags.Image+":"+flags.Version)
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
