package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewRootCMD() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "eve",
		Short: "Eve is a tool that simplifies deploying applications on Akash",
	}
	rootCmd.AddCommand(NewInitCMD(), NewActionsCMD(), NewPackCMD())
	return rootCmd
}

func NewPackCMD() *cobra.Command {
	packCmd := &cobra.Command{
		Use:   "pack",
		Short: "Pack your project into a container using buildpacks",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Packing Eve")
		},
	}
	return packCmd
}

func NewInitCMD() *cobra.Command {
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize eve in the current directory",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Initializing Eve")
		},
	}
	return initCmd
}

func NewActionsCMD() *cobra.Command {
	actionsCmd := &cobra.Command{
		Use:   "actions",
		Short: "Manage your Github actions",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Actions")
		},
	}
	return actionsCmd
}
