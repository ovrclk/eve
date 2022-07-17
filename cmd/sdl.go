package cmd

import (
	"context"

	"github.com/spf13/cobra"
)

type SDLFlags struct {
}

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
	// p := path.Join(globalFlags.Path, source)
	// src, err := sdl.ReadFile(p)
	// if err != nil {
	// 	return err
	// }
	return nil
}
