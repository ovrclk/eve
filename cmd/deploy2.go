package cmd

import (
	"bufio"
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path"
	"strings"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	sdkflags "github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/input"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	akashapp "github.com/ovrclk/akash/app"
	"github.com/ovrclk/akash/sdkutil"
	"github.com/ovrclk/akash/sdl"
	cutils "github.com/ovrclk/akash/x/cert/utils"
	types "github.com/ovrclk/akash/x/deployment/types/v1beta2"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/tendermint/tendermint/rpc/client/http"
	ttypes "github.com/tendermint/tendermint/types"

	"github.com/ovrclk/eve/logger"
)

const (
	FlagDepositorAccount = "depositor-account"
	// Only way to detect the timeout error.
	// https://github.com/tendermint/tendermint/blob/46e06c97320bc61c4d98d3018f59d47ec69863c9/rpc/core/mempool.go#L124
	timeoutErrorMessage        = "timed out waiting for tx to be included in a block"
	BroadcastBlockRetryTimeout = 300 * time.Second
	broadcastBlockRetryPeriod  = time.Second

	// Only way to check for tx not found error.
	// https://github.com/tendermint/tendermint/blob/46e06c97320bc61c4d98d3018f59d47ec69863c9/rpc/core/tx.go#L31-L33
	notFoundErrorMessageSuffix = ") not found"
)

// NewDeploy2Cmd returns a new instance of the deploy2 command
func NewDeploy2Cmd(ctx context.Context, cancel context.CancelFunc) *cobra.Command {
	deploy2Cmd := &cobra.Command{
		Use:   "deploy2",
		Short: "Deploy a new application",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := doDeployCreate(ctx, cmd, args[0]); err != nil {
				log.Fatalf("error: %v", err)
			}
		},
	}
	//sdkflags.AddTxFlagsToCmd(deploy2Cmd)
	return deploy2Cmd
}

func doDeployCreate(ctx context.Context, cmd *cobra.Command, sdlPath string) error {
	sdkutil.InitSDKConfig()
	clientCtx, err := client.GetClientTxContext(cmd)
	if err != nil {
		logger.Debug("error getting client context", "err", err)
		return err
	}

	// read user home directory
	home, err := os.UserHomeDir()
	if err != nil {
		logger.Debug("error reading user home directory", "err", err)
		return err
	}
	home = path.Join(home, ".akash")

	// read $AKASH_NODE environment variable
	node := os.Getenv("AKASH_NODE")

	// create a RPC CLient
	rpcClient, err := http.New(node, "/websocket")
	if err != nil {
		logger.Debug("error creating RPC client", "err", err)
		return err
	}
	encodingConfig := akashapp.MakeEncodingConfig()

	clientCtx = clientCtx.
		WithHomeDir(home).
		WithViper("akash").
		WithChainID("akashnet-2").
		WithKeyringDir(home).
		WithNodeURI(node).
		WithOffline(false).
		WithClient(rpcClient).
		WithAccountRetriever(authtypes.AccountRetriever{}).
		WithInterfaceRegistry(encodingConfig.InterfaceRegistry).WithTxConfig(encodingConfig.TxConfig)

	// initiate a new keyring
	kr, err := client.NewKeyringFromBackend(clientCtx, "test")
	if err != nil {
		logger.Debug("error creating keyring", "err", err)
		return err
	}

	clientCtx = clientCtx.WithKeyring(kr)
	addrSrt := "akash1aqnvsas9plseewyu3nt2rtz6ml4aya4s02qm0q"

	// from, _ := flagSet.GetString(flags.FlagFrom)
	// fromAddr, fromName, keyType, err := GetFromFields(clientCtx.Keyring, from, clientCtx.GenerateOnly)
	// if err != nil {
	// 	return clientCtx, err
	// }
	//
	// clientCtx = clientCtx.WithFrom(from).WithFromAddress(fromAddr).WithFromName(fromName)
	addr, err := sdk.AccAddressFromBech32(addrSrt)
	if err != nil {
		logger.Error("error parsing address", "err", err)
	}
	clientCtx = clientCtx.WithFromAddress(addr).WithFromName("deploy")

	if _, err = cutils.LoadAndQueryCertificateForAccount(cmd.Context(), clientCtx, nil); err != nil {
		if os.IsNotExist(err) {

			// generate a new certificate
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

	// create a new deployment ID from unix timestamp

	// create a random number
	rand.Seed(time.Now().UnixNano())
	id := types.DeploymentID{
		Owner: clientCtx.GetFromAddress().String(),
		DSeq:  rand.Uint64(),
	}

	// id, err := DeploymentIDFromFlags(cmd.Flags(), WithOwner(clientCtx.FromAddress))
	// if err != nil {
	// 	return err
	// }

	version, err := sdl.Version(sdlManifest)
	if err != nil {
		return err
	}

	depositVal := "5000000uakt"
	deposit, err := sdk.ParseCoinNormalized(depositVal)
	if err != nil {
		logger.Debug("error parsing deposit: ", err)
		return err
	}

	msg := &types.MsgCreateDeployment{
		ID:        id,
		Version:   version,
		Groups:    make([]types.GroupSpec, 0, len(groups)),
		Deposit:   deposit,
		Depositor: id.Owner,
	}

	for _, group := range groups {
		msg.Groups = append(msg.Groups, *group)
	}

	if err := msg.ValidateBasic(); err != nil {
		return err
	}

	return broadcastDeploymentCreateTX(ctx, clientCtx, msg)

}

func broadcastDeploymentCreateTX(ctx context.Context, clientCtx client.Context, msg sdk.Msg) error {
	from := clientCtx.GetFromAddress()
	logger.Debug("from address ", from)

	txf, err := sdkutil.PrepareFactory(clientCtx, newTxFactory(clientCtx))
	if err != nil {
		logger.Debug("error preparing factory: ", err)
		return err
	}

	// Adjust Gas
	txf, err = sdkutil.AdjustGas(clientCtx, txf, msg)
	if err != nil {
		logger.Debug("error adjusting gas: ", err)
		return err
	}

	// Build Unsigned Transaction
	txb, err := tx.BuildUnsignedTx(txf, msg)
	if err != nil {
		logger.Debug("error building unsigned tx: ", err)
		return err
	}

	ok, err := confirmTx(clientCtx, txb)
	if !ok || err != nil {
		return err
	}

	txb.SetFeeGranter(clientCtx.GetFeeGranterAddress())
	if err = tx.Sign(txf, clientCtx.GetFromName(), txb, true); err != nil {
		logger.Debug("error signing tx: ", err)
		return err
	}

	txBytes, err := clientCtx.TxConfig.TxEncoder()(txb.GetTx())
	if err != nil {
		logger.Debug("error encoding tx: ", err)
		return err
	}

	// Broadcast to a Tendermint node
	res, err := doBroadcast(ctx, clientCtx, BroadcastBlockRetryTimeout, txBytes)
	if err != nil {
		logger.Debug("error broadcasting tx: ", err)
		return err
	}
	return clientCtx.PrintProto(res)
}

func doBroadcast(ctx context.Context, cctx client.Context, timeout time.Duration, txb ttypes.Tx) (*sdk.TxResponse, error) {
	switch cctx.BroadcastMode {
	case sdkflags.BroadcastSync:
		return cctx.BroadcastTxSync(txb)
	case sdkflags.BroadcastAsync:
		return cctx.BroadcastTxAsync(txb)
	}

	hash := hex.EncodeToString(txb.Hash())

	// broadcast-mode=block
	// submit with mode commit/block
	cres, err := cctx.BroadcastTxCommit(txb)
	if err == nil {
		// good job
		return cres, nil
	} else if !strings.HasSuffix(err.Error(), timeoutErrorMessage) {
		return cres, err
	}

	// timeout error, continue on to retry

	// loop
	lctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for lctx.Err() == nil {

		// wait up to one second
		select {
		case <-lctx.Done():
			return cres, err
		case <-time.After(broadcastBlockRetryPeriod):
		}

		// check transaction
		// https://github.com/cosmos/cosmos-sdk/pull/8734
		res, err := authtx.QueryTx(cctx, hash)
		if err == nil {
			return res, nil
		}

		// if it's not a "not found" error, return
		if !strings.HasSuffix(err.Error(), notFoundErrorMessageSuffix) {
			return res, err
		}
	}

	return cres, lctx.Err()
}

func confirmTx(ctx client.Context, txb client.TxBuilder) (bool, error) {
	if ctx.SkipConfirm {
		return true, nil
	}

	out, err := ctx.TxConfig.TxJSONEncoder()(txb.GetTx())
	if err != nil {
		return false, err
	}

	_, _ = fmt.Fprintf(os.Stderr, "%s\n\n", out)

	buf := bufio.NewReader(os.Stdin)
	ok, err := input.GetConfirmation("confirm transaction before signing and broadcasting", buf, os.Stderr)

	if err != nil || !ok {
		_, _ = fmt.Fprintf(os.Stderr, "%s\n", "cancelled transaction")
		return false, err
	}

	return true, nil
}

func newTxFactory(clientCtx client.Context) tx.Factory {
	//gasSetting := sdkflags.GasSetting{Simulate: false, Gas: sdkflags.DefaultGasLimit}

	f := tx.Factory{}
	f = f.
		WithTxConfig(clientCtx.TxConfig).
		WithAccountRetriever(clientCtx.AccountRetriever).
		WithKeybase(clientCtx.Keyring).
		WithChainID(clientCtx.ChainID).
		WithGas(sdkflags.DefaultGasLimit).WithAccountRetriever(clientCtx.AccountRetriever)
	return f
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
