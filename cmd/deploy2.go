package cmd

import (
	"context"
	"log"
	"os"
	"path"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	sdkflags "github.com/cosmos/cosmos-sdk/client/flags"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ovrclk/akash/cmd/common"
	"github.com/ovrclk/akash/sdkutil"
	"github.com/ovrclk/akash/sdl"
	cutils "github.com/ovrclk/akash/x/cert/utils"
	types "github.com/ovrclk/akash/x/deployment/types/v1beta2"
	"github.com/ovrclk/eve/logger"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const FlagDepositorAccount = "depositor-account"

// NewDeploy2Cmd returns a new instance of the deploy2 command
func NewDeploy2Cmd(ctx context.Context, cancel context.CancelFunc) *cobra.Command {
	deploy2Cmd := &cobra.Command{
		Use:   "deploy2",
		Short: "Deploy a new application",
		Run: func(cmd *cobra.Command, args []string) {
			if err := doDeployCreate(cmd, args[0]); err != nil {
				log.Fatalf("error: %v", err)
			}
		},
	}
	sdkflags.AddTxFlagsToCmd(deploy2Cmd)
	return deploy2Cmd
}

func doDeployCreate(cmd *cobra.Command, sdlPath string) error {
	clientCtx, err := client.GetClientTxContext(cmd)
	if err != nil {
		logger.Debug("error getting client context", "err", err)
		return err
	}

	// initiate a new keyring
	kr, err := client.NewKeyringFromBackend(clientCtx, "test")
	if err != nil {
		logger.Debug("error creating keyring", "err", err)
		return err
	}

	// read user home directory
	home, err := os.UserHomeDir()
	if err != nil {
		logger.Debug("error reading user home directory", "err", err)
		return err
	}
	home = path.Join(home, ".akash")

	addr, _ := sdk.AccAddressFromBech32("akash1aqnvsas9plseewyu3nt2rtz6ml4aya4s02qm0q")
	clientCtx = clientCtx.WithHomeDir(home).WithViper("akash").WithChainID("akashnet-2").WithFromAddress(addr).WithKeyring(kr)

	logger.Debugf("client context: %+v", clientCtx)

	if _, err = cutils.LoadAndQueryCertificateForAccount(cmd.Context(), clientCtx, nil); err != nil {
		if os.IsNotExist(err) {
			err = errors.Errorf("no certificate file found for account %q.\n"+
				"consider creating it as certificate required to submit manifest", clientCtx.FromAddress.String())
		}
		logger.Debug("error loading certificate: ", err)
		return err
	}

	sdlManifest, err := sdl.ReadFile(sdlPath)
	if err != nil {
		logger.Debug("error reading sdl file: ", err)
		return err
	}

	groups, err := sdlManifest.DeploymentGroups()
	if err != nil {
		return err
	}

	id, err := DeploymentIDFromFlags(cmd.Flags(), WithOwner(clientCtx.FromAddress))
	if err != nil {
		return err
	}

	version, err := sdl.Version(sdlManifest)
	if err != nil {
		return err
	}

	deposit, err := common.DepositFromFlags(cmd.Flags())
	if err != nil {
		return err
	}

	depositorAcc, err := DepositorFromFlags(cmd.Flags(), id.Owner)
	if err != nil {
		return err
	}

	msg := &types.MsgCreateDeployment{
		ID:        id,
		Version:   version,
		Groups:    make([]types.GroupSpec, 0, len(groups)),
		Deposit:   deposit,
		Depositor: depositorAcc,
	}

	for _, group := range groups {
		msg.Groups = append(msg.Groups, *group)
	}

	if err := msg.ValidateBasic(); err != nil {
		return err
	}

	return sdkutil.BroadcastTX(cmd.Context(), clientCtx, cmd.Flags(), msg)

}

type MarketOptions struct {
	Owner    sdk.AccAddress
	Provider sdk.AccAddress
}

type MarketOption func(*MarketOptions)

func WithOwner(val sdk.AccAddress) MarketOption {
	return func(opt *MarketOptions) {
		opt.Owner = val
	}
}

// DeploymentIDFromFlags returns DeploymentID with given flags, owner and error if occurred
func DeploymentIDFromFlags(flags *pflag.FlagSet, opts ...MarketOption) (types.DeploymentID, error) {
	var id types.DeploymentID
	opt := &MarketOptions{}

	for _, o := range opts {
		o(opt)
	}

	var owner string
	if flag := flags.Lookup("owner"); flag != nil {
		owner = flag.Value.String()
	}

	// if --owner flag was explicitly provided, use that.
	var err error
	if owner != "" {
		opt.Owner, err = sdk.AccAddressFromBech32(owner)
		if err != nil {
			return id, err
		}
	}

	id.Owner = opt.Owner.String()

	if id.DSeq, err = flags.GetUint64("dseq"); err != nil {
		return id, err
	}
	return id, nil
}

// DepositorFromFlags returns the depositor account if one was specified in flags,
// otherwise it returns the owner's account.
func DepositorFromFlags(flags *pflag.FlagSet, owner string) (string, error) {
	depositorAcc, err := flags.GetString(FlagDepositorAccount)
	if err != nil {
		return "", err
	}

	// if no depositor is specified, owner is the default depositor
	if strings.TrimSpace(depositorAcc) == "" {
		return owner, nil
	}

	_, err = sdk.AccAddressFromBech32(depositorAcc)
	return depositorAcc, err
}
