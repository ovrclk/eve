package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

func NewDeploy(ctx context.Context, cancel context.CancelFunc) *cobra.Command {
	deployCmd := &cobra.Command{
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

			if err = runUpdateDeployment(ctx, cancel, dseq); err != nil {
				return err
			}

			if err = runProviderSendManifest(ctx, cancel, provider, dseq); err != nil {
				return err
			}
			return nil
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
