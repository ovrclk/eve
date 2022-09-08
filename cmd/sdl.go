package cmd

import (
	"context"
	"os"
	"path"

	"github.com/ovrclk/eve/logger"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type SDLFlags struct{}

func NewSDL(ctx context.Context, cancel context.CancelFunc) *cobra.Command {
	sdlFlags := &SDLFlags{}
	cmd := &cobra.Command{
		Use:   "sdl",
		Short: "Manage SDL deployment file",
		RunE: func(cmd *cobra.Command, args []string) error {
			source := ""
			if len(args) > 0 {
				source = args[0]
			}
			return runSDL(ctx, cancel, source, sdlFlags)
		},
	}
	return cmd
}

func runSDL(ctx context.Context, cancel context.CancelFunc, source string, flags *SDLFlags) error {
	logger.Debug("runSDL:", "source", source)
	p := path.Join(globalFlags.Path, source)
	//b,var b byte[]
	b, err := os.ReadFile(p)
	if err != nil {
		return err
	}

	// create a cache directory under the state directory if it doesn't exist
	cacheDir := path.Join(globalFlags.Path, globalFlags.StateDirName, "cache")
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		err = os.Mkdir(cacheDir, 0755)
		if err != nil {
			return err
		}
	}

	// read the IMAGE variable
	image, err := readvar("IMAGE")
	if err != nil {
		return err
	}

	version, err := readvar("VERSION")
	if err != nil {
		return err
	}

	image = image + ":" + version

	// parse YAML file
	var data map[string]interface{}
	if err := yaml.Unmarshal(b, &data); err != nil {
		return err
	}
	data["services"].(map[string]interface{})["web"].(map[string]interface{})["image"] = image

	// write the new YAML file
	b, err = yaml.Marshal(data)
	if err != nil {
		return err
	}

	target := path.Join(cacheDir, "sdl."+version+".yml")
	if err := os.WriteFile(target, b, 0644); err != nil {
		return err
	}
	logger.Infof("SDL: updated %s", p)
	return nil
}
