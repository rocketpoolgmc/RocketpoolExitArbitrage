package arbitrage

import (
	"bufio"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"
	"time"

	"log/slog"

	"github.com/0xtrooper/flashbots_client"
)

func ExecuteDistribute(ctx context.Context, logger *slog.Logger, dataIn *DataIn) error {
	logger.With(slog.String("function", "Simulate"))

	err := VerifyInputData(ctx, logger, dataIn)
	if err != nil {
		return errors.Join(errors.New("failed to verify input data"), err)
	}

	logger.Debug("verified input data")

	if dataIn.RefundAddress != nil {
		err = dataIn.FbClient.UpdateFeeRefundRecipient(*dataIn.RefundAddress)
		if err != nil {
			return errors.Join(errors.New("failed to update flashbots fee refund recipient"), err)
		}

		fmt.Printf("Updated flashbots fee refund recipient to supplied recipient (%s)\n", (*dataIn.RefundAddress).Hex())
	} else if dataIn.RandomPrivateKey {
		err := dataIn.FbClient.UpdateFeeRefundRecipient(*dataIn.NodeAddress)
		if err != nil {
			return errors.Join(errors.New("failed to update flashbots fee refund recipient to node address"), err)
		}

		fmt.Printf("Updated flashbots fee refund recipient to node address (%s)\n", (*dataIn.NodeAddress).Hex())
	}

	bundle, expectedProfit, err := BuildCall(ctx, logger, *dataIn)
	if err != nil {
		return errors.Join(errors.New("failed to build call"), err)
	}

	logger.Debug("created flashbots client")
	res, success, err := dataIn.FbClient.SimulateBundle(bundle, 0)
	if err != nil {
		return errors.Join(errors.New("failed to simulate bundle"), err)
	}

	for index, tx := range res.Results {
		if tx.Error != "" {
			logger.Warn("tx failed", slog.Int("index", index), slog.String("error", tx.Error), slog.String("revertReason", hex.EncodeToString([]byte(tx.RevertReason))))
			success = false
		}
	}

	maxBundleFees, maxArbitrageFees := evalGasPrices(bundle)

	maxBundleFeesFloat, _ := new(big.Float).Quo(new(big.Float).SetInt(maxBundleFees), new(big.Float).SetInt(big.NewInt(1e18))).Float64()
	maxArbitrageFeesFloat, _ := new(big.Float).Quo(new(big.Float).SetInt(maxArbitrageFees), new(big.Float).SetInt(big.NewInt(1e18))).Float64()
	expectedProfitFloat, _ := new(big.Float).Quo(new(big.Float).SetInt(expectedProfit), new(big.Float).SetInt(big.NewInt(1e18))).Float64()
	fmt.Print("Simulated bundle (")
	if success {
		fmt.Print(string(colorGreen), "success", string(colorReset))
	} else {
		fmt.Print(string(colorRed), "failed", string(colorReset))
	}
	fmt.Println("):")
	fmt.Printf("    Expected profit after fees: %.6f, with a tx fee of %.6f\n", expectedProfitFloat-maxBundleFeesFloat, maxBundleFeesFloat)
	fmt.Printf("    Expected profit after arbitrage fees: %.6f, with a tx fee of %.6f (interesting if you want to distribute regardless)\n\n", expectedProfitFloat-maxArbitrageFeesFloat, maxArbitrageFeesFloat)

	if dataIn.DryRun {
		txs := bundle.Transactions()
		fmt.Println("Dry run. Would have sent the following bundle:")
		for i, tx := range txs {
			baseGwei, _ := new(big.Float).Quo(new(big.Float).SetInt(tx.GasFeeCap()), new(big.Float).SetInt(big.NewInt(1e9))).Float64()
			tipGwei, _ := new(big.Float).Quo(new(big.Float).SetInt(tx.GasTipCap()), new(big.Float).SetInt(big.NewInt(1e9))).Float64()

			fmt.Printf("Transaction %d:\n", i+1)
			fmt.Printf("    From: %s\n", dataIn.NodeAddress.Hex())
			fmt.Printf("    To: %s\n", tx.To().Hex())
			fmt.Printf("    Value: %s\n", tx.Value().String())
			fmt.Printf("    Gas Limit: %d\n", tx.Gas())
			fmt.Printf("    Base Fee: %s (%.2f Gwei)\n", tx.GasFeeCap().String(), baseGwei)
			fmt.Printf("    Priority Fee: %s (%.4f Gwei)\n", tx.GasTipCap().String(), tipGwei)
			fmt.Printf("    Nonce: %d\n", tx.Nonce())
			fmt.Printf("    Data: %s\n", hex.EncodeToString(tx.Data()))
		}
		return nil
	}

	// end the attempt if the simulation failed - after printing the dryrun if it was requested!
	if !success {
		return errors.New("bundle simulation failed")
	}

	// this checks if a bundle makes sense to make arbitrage profits
	if dataIn.CheckProfit && !dataIn.CheckProfitIgnoreDistributeCost && expectedProfit.Cmp(maxBundleFees) < 0 {
		return errors.New("expected profit is less than max bundle fees")
	}

	// this checks if a bundle makes sense if the user wants to distribute
	// aka. distribute gas needs to be paid one way or another
	if dataIn.CheckProfit && dataIn.CheckProfitIgnoreDistributeCost && expectedProfit.Cmp(maxArbitrageFees) < 0 {
		return errors.New("expected profit is less than max arbitrage fees")
	}

	if !dataIn.SkipConfirmation && !waitForUserConfirmation() {
		return errors.New("user did not confirm to proceed")
	}

	// add more builders to improve chance to be included
	networkID, err := dataIn.Client.NetworkID(ctx)
	if err != nil {
		return errors.Join(errors.New("failed to get network id"), err)
	}
	bundle.UseAllBuilders(networkID.Uint64())

	// set target block number
	blockNumber, err := dataIn.Client.BlockNumber(ctx)
	if err != nil {
		return errors.Join(errors.New("failed to get block number"), err)
	}
	bundle.SetTargetBlockNumber(blockNumber + 1)

	fmt.Printf("\nSent bundle with hash: %s. Waiting for up to one minute to see if the transaction is included...\n\n", res.BundleHash)

	timeoutContext, cancel := context.WithTimeout(ctx, time.Second*60)
	successfullyIncluded, err := dataIn.FbClient.SendNBundleAndWait(timeoutContext, bundle, 3)
	cancel()
	if err != nil {
		return errors.Join(errors.New("failed to wait for bundle inclusion"), err)
	}

	if !successfullyIncluded {
		return errors.New("bundle was not included in the mempool")
	}

	// print successful inclusion and tx link
	if len(res.Results) == 2 {
		arbTx := res.Results[1]
		fmt.Printf("Distributed minipool! Arbitrage tx: https://etherscan.io/tx/%s\n\n", arbTx.TxHash.Hex())
	} else {
		arbTx := res.Results[len(res.Results)-1]
		fmt.Printf("Distributed minipools! Arbitrage tx: https://etherscan.io/tx/%s\n\n", arbTx.TxHash.Hex())
	}

	return nil
}

func evalGasPrices(bundle *flashbots_client.Bundle) (bundleGasPrice, arbitrageGasPrice *big.Int) {
	bundleGasPrice = bundle.MaximumGasFeePaid()

	txs := bundle.Transactions()
	arbTx := txs[len(txs)-1]

	return bundleGasPrice, arbTx.Cost()
}

func waitForUserConfirmation() bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Do you want to proceed? (y/n): ")
	response, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}
	response = strings.TrimSpace(response)
	switch strings.ToLower(response) {
	case "y", "yes":
		return true
	case "n", "no":
		return false
	default:
		fmt.Println("Invalid input. Please type 'y' or 'n'.")
		return waitForUserConfirmation()
	}
}
