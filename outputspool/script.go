package outputspool

import (
	"github.com/lightningnetwork/lnd/lnwallet"
	"github.com/roasbeef/btcd/txscript"
	"github.com/roasbeef/btcd/wire"
	"github.com/roasbeef/btcutil"
)

func (d *DiskStrayOutputsPool) AddInputs(txVSize int64,
	inputs ...lnwallet.SpendableOutput) error {

	// Compute the total amount contained in the inputs.
	var totalAmt btcutil.Amount
	for _, input := range inputs {
		totalAmt += input.Amount()
	}

	return nil
}

// Sweep generates transaction for all added previously inputs to the wallet
// output address and broadcast it to the network
func (d *DiskStrayOutputsPool) Sweep() error {
	btx, err := d.GenSweepTx()
	if err != nil {
		return err
	}

	return d.cfg.PublishTransaction(btx.MsgTx())
}

// GenSweepTx
func (d *DiskStrayOutputsPool) GenSweepTx() (*btcutil.Tx, error) {
	// First, we obtain a new public key script from the wallet which we'll
	// sweep the funds to.
	pkScript, err := d.cfg.GenSweepScript()
	if err != nil {
		return nil, err
	}

	strayInputs, err := d.FetchAllStrayInputs()
	if err != nil {
		return nil, err
	}

	return d.genSweepTx(pkScript, strayInputs...)
}


// genSweepTx
func (d *DiskStrayOutputsPool) genSweepTx(pkScript []byte,
	strayInputs ...*strayInput) (*btcutil.Tx, error) {
	// We'll actually attempt to target inclusion within the next two
	// blocks as we'd like to sweep these funds back into our wallet ASAP.
	feePerVSize, err := d.cfg.Estimator.EstimateFeePerVSize(2)
	if err != nil {
		return nil, err
	}

	// With the fee calculated, we can now create the transaction using the
	// information gathered above and the provided retribution information.
	txn := wire.NewMsgTx(2)

	// Compute the total amount contained in the inputs.
	var totalAmt btcutil.Amount

	for _, sInput := range strayInputs {
		txFee := feePerVSize.FeeForVSize(sInput.txVSize)
		totalAmt += sInput.totalAmt - txFee

		// Next, we add all of the spendable outputs as inputs to the
		// transaction.
		for _, input := range sInput.inputs {
			txn.AddTxIn(&wire.TxIn{
				PreviousOutPoint: *input.OutPoint(),
			})
		}

		// Create a sighash cache to improve the performance of hashing and
		// signing SigHashAll inputs.
		hashCache := txscript.NewTxSigHashes(txn)

		// Create a closure that encapsulates the process of initializing a
		// particular output's witness generation function, computing the
		// witness, and attaching it to the transaction. This function accepts
		// an integer index representing the intended txin index, and the
		// breached output from which it will spend.
		addWitness := func(idx int, so lnwallet.SpendableOutput) error {
			// First, we construct a valid witness for this outpoint and
			// transaction using the SpendableOutput's witness generation
			// function.
			witness, err := so.BuildWitness(d.cfg.Signer, txn, hashCache, idx)
			if err != nil {
				return err
			}

			// Then, we add the witness to the transaction at the
			// appropriate txin index.
			txn.TxIn[idx].Witness = witness

			return nil
		}

		// Generate a witness for each input of the transaction.
		for i, input := range sInput.inputs {
			if err := addWitness(i, input); err != nil {
				return nil, err
			}
		}
	}

	// We begin by adding the output to which our funds will be deposited.
	txn.AddTxOut(&wire.TxOut{
		PkScript: pkScript,
		Value:    int64(totalAmt),
	})

	// Before signing the transaction, check to ensure that it meets some
	// basic validity requirements.
	btx := btcutil.NewTx(txn)
	if err := lnwallet.CheckTransactionSanity(btx); err != nil {
		return nil, err
	}

	return btx, nil
}
