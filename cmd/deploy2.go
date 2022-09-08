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
	"strconv"
	"strings"
	"time"

	sdkclient "github.com/cosmos/cosmos-sdk/client"
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
	akttypes "github.com/ovrclk/akash/x/deployment/types/v1beta2"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	tmhttpclient "github.com/tendermint/tendermint/rpc/client/http"
	ttypes "github.com/tendermint/tendermint/types"

	"github.com/ovrclk/eve/logger"
)

const (
	// Only way to detect the timeout error.
	// https://github.com/tendermint/tendermint/blob/46e06c97320bc61c4d98d3018f59d47ec69863c9/rpc/core/mempool.go#L124
	timeoutErrorMessage = "timed out waiting for tx to be included in a block"

	// Only way to check for tx not found error.
	// https://github.com/tendermint/tendermint/blob/46e06c97320bc61c4d98d3018f59d47ec69863c9/rpc/core/tx.go#L31-L33
	notFoundErrorMessageSuffix = ") not found"

	BroadcastBlockRetryTimeout = 300 * time.Second
	broadcastBlockRetryPeriod  = time.Second
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
	return deploy2Cmd
}

func doDeployCreate(ctx context.Context, cmd *cobra.Command, sdlPath string) error {
	sdkutil.InitSDKConfig()
	clientCtx, err := sdkclient.GetClientTxContext(cmd)
	if err != nil {
		return errors.Wrap(err, "error getting client context")
	}

	// read user home directory
	home, err := os.UserHomeDir()
	if err != nil {
		return errors.Wrap(err, "error getting user home directory")
	}
	home = path.Join(home, ".akash")

	// read $AKASH_NODE environment variable
	node := os.Getenv("AKASH_NODE")

	// create a RPC CLient
	rpcClient, err := tmhttpclient.New(node, "/websocket")
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
		WithInterfaceRegistry(encodingConfig.InterfaceRegistry).
		WithTxConfig(encodingConfig.TxConfig).
		WithCodec(encodingConfig.Marshaler).
		WithSkipConfirmation(true)

	// initiate a new keyring
	kr, err := sdkclient.NewKeyringFromBackend(clientCtx, "test")
	if err != nil {
		logger.Debug("error creating keyring", "err", err)
		return err
	}

	clientCtx = clientCtx.WithKeyring(kr)
	addrSrt := "akash1aqnvsas9plseewyu3nt2rtz6ml4aya4s02qm0q"
	addr, err := sdk.AccAddressFromBech32(addrSrt)

	if err != nil {
		return errors.Wrap(err, "error getting address")
	}
	clientCtx = clientCtx.WithFromAddress(addr).WithFromName("deploy")

	if _, err = cutils.LoadAndQueryCertificateForAccount(cmd.Context(), clientCtx, nil); err != nil {
		if os.IsNotExist(err) {
			// TODO: generate a new certificate
			err = errors.Errorf("no certificate file found for account %q.\n"+
				"consider creating it as certificate required to submit manifest", clientCtx.FromAddress.String())
		}
		logger.Debug("error loading certificate: ", err)
		return err
	}

	sdlManifest, err := sdl.ReadFile(sdlPath)
	if err != nil {
		return errors.Wrap(err, "error reading manifest")
	}

	groups, err := sdlManifest.DeploymentGroups()
	if err != nil {
		return errors.Wrap(err, "error reading groups from the manifest")
	}

	// create a new deployment ID from a random number with unix timestamp as the seed
	rand.Seed(time.Now().UnixNano())
	deployID := akttypes.DeploymentID{Owner: clientCtx.GetFromAddress().String(), DSeq: rand.Uint64()}
	version, err := sdl.Version(sdlManifest)
	if err != nil {
		return errors.Wrap(err, "error reading version from the manifest")
	}

	depositVal := "5000000uakt"
	deposit, err := sdk.ParseCoinNormalized(depositVal)
	if err != nil {
		logger.Debug("error parsing deposit: ", err)
		return err
	}

	msg := &akttypes.MsgCreateDeployment{
		ID:        deployID,
		Version:   version,
		Groups:    make([]akttypes.GroupSpec, 0, len(groups)),
		Deposit:   deposit,
		Depositor: deployID.Owner,
	}

	for _, group := range groups {
		msg.Groups = append(msg.Groups, *group)
	}

	if err := msg.ValidateBasic(); err != nil {
		return errors.Wrap(err, "basic validation failed")
	}

	return broadcastDeploymentCreateTX(ctx, clientCtx, msg)
}

func broadcastDeploymentCreateTX(ctx context.Context, clientCtx sdkclient.Context, msg sdk.Msg) error {
	txf, err := sdkutil.PrepareFactory(clientCtx, newTxFactory(clientCtx))
	if err != nil {
		logger.Debug("error preparing factory: ", err)
		return err
	}

	// Adjust Gas
	txf, err = sdkutil.AdjustGas(clientCtx, txf, msg)
	if err != nil {
		return errors.Wrap(err, "error adjusting gas")
	}

	// Build Unsigned Transaction
	txb, err := tx.BuildUnsignedTx(txf, msg)
	if err != nil {
		return errors.Wrap(err, "error building unsigned transaction")
	}

	ok, err := confirmTx(clientCtx, txb)
	if !ok || err != nil {
		return errors.Wrap(err, "error confirming transaction")
	}

	txb.SetFeeGranter(clientCtx.GetFeeGranterAddress())
	if err = tx.Sign(txf, clientCtx.GetFromName(), txb, true); err != nil {
		return errors.Wrap(err, "error signing transaction")
	}

	txBytes, err := clientCtx.TxConfig.TxEncoder()(txb.GetTx())
	if err != nil {
		return errors.Wrap(err, "error encoding transaction")
	}

	// Broadcast to a Tendermint node
	res, err := doBroadcast(ctx, clientCtx, BroadcastBlockRetryTimeout, txBytes)
	if err != nil {
		return errors.Wrap(err, "error broadcasting transaction")
	}
	return clientCtx.PrintProto(res)
}

func newTxFactory(clientCtx sdkclient.Context) tx.Factory {
	gasStr := os.Getenv("AKASH_GAS")
	gastSetting, err := sdkflags.ParseGasSetting(gasStr)
	if err != nil {
		logger.Debug("error parsing gas: ", err)
	}

	gasAdjustStr := os.Getenv("AKASH_GAS_ADJUSTMENT")
	gasAdjustment, err := strconv.ParseFloat(gasAdjustStr, 64)
	if err != nil {
		logger.Debug("error parsing gas adjustment: ", err)
	}

	gasPricesStr := os.Getenv("AKASH_GAS_PRICES")

	f := tx.Factory{}
	return f.
		WithTxConfig(clientCtx.TxConfig).
		WithAccountRetriever(clientCtx.AccountRetriever).
		WithKeybase(clientCtx.Keyring).
		WithChainID(clientCtx.ChainID).
		WithSimulateAndExecute(gastSetting.Simulate).
		WithGas(gastSetting.Gas).
		WithGasAdjustment(gasAdjustment).
		WithAccountRetriever(clientCtx.AccountRetriever).
		WithGasPrices(gasPricesStr)
}

func doBroadcast(ctx context.Context, cctx sdkclient.Context, timeout time.Duration, txb ttypes.Tx) (*sdk.TxResponse, error) {
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

func confirmTx(ctx sdkclient.Context, txb sdkclient.TxBuilder) (bool, error) {
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
