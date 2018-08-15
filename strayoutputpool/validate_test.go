package strayoutputpool_test

import (
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/roasbeef/btcd/chaincfg/chainhash"
	"github.com/roasbeef/btcd/wire"
	"github.com/roasbeef/btcutil"

	"github.com/lightningnetwork/lnd/lnwallet"
	"github.com/lightningnetwork/lnd/strayoutputpool"
)

type strayOutputsPoolMock struct {
	// AddSpendableOutputs adds outputs to the pool for late processing.
	AddSpendableOutputFunc func (output lnwallet.SpendableOutput) error

	// GenSweepTx generates transaction for all added outputs.
	GenSweepTxFunc func() (*btcutil.Tx, error)

	// Sweep generates transaction and broadcast it to the network.
	SweepFunc func() error
}

// AddSpendableOutput is mock function.
func (s *strayOutputsPoolMock) AddSpendableOutput (output lnwallet.SpendableOutput) error {
	return s.AddSpendableOutputFunc(output)
}

// GenSweepTx is mock function.
func (s *strayOutputsPoolMock) GenSweepTx() (*btcutil.Tx, error) {
	return s.GenSweepTxFunc()
}

// GenSweepTx is mock function.
func (s *strayOutputsPoolMock) Sweep() error {
	return s.SweepFunc()
}

// Interface match validation.
var _ strayoutputpool.StrayOutputsPool = (*strayOutputsPoolMock)(nil)


func TestCutStrayInput(t *testing.T) {
	sPool := &strayOutputsPoolMock{}

	testCases := []struct {
		inp *lnwallet.BaseOutput
		isMustBeCutted bool
	} {
		{
			lnwallet.NewBaseOutput(50000,
				*wire.NewOutPoint(&chainhash.Hash{}, 0),
				lnwallet.CommitmentNoDelay, lnwallet.SignDescriptor{}),
			false,
		},
		{
			lnwallet.NewBaseOutput(100,
				*wire.NewOutPoint(&chainhash.Hash{}, 0),
				lnwallet.CommitmentNoDelay, lnwallet.SignDescriptor{}),
			true,
		},
	}

	estimator := lnwallet.StaticFeeEstimator{FeeRate: 250}
	feeRate, err := estimator.EstimateFeePerVSize(6)
	if err != nil {
		t.Fatal("couldn't init fee estimator")
	}

	for _, testCase := range testCases {
		var addCounter int
		sPool.AddSpendableOutputFunc = func (output lnwallet.SpendableOutput) error {
			addCounter++

			return nil
		}

		strayoutputpool.CutStrayInput(sPool, feeRate, testCase.inp)

		if testCase.isMustBeCutted && addCounter != 1 ||
			!testCase.isMustBeCutted && addCounter != 0 {
			t.Fatalf("wrong AddSpendableOutput call count for input: %v",
				spew.Sdump(testCase.inp))
		}
	}
}