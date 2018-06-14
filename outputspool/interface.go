package outputspool

import (
	"github.com/lightningnetwork/lnd/chainntnfs"
	"github.com/lightningnetwork/lnd/channeldb"
	"github.com/lightningnetwork/lnd/lnwallet"
	"github.com/roasbeef/btcd/wire"
	"github.com/roasbeef/btcutil"
)

// TODO(roasbeef)
// Finally, in the case that the output is simply too small to sweep at a target fee rate,
// it should be added to a "stray output" pool which is stored persistently.
// The items in this pool (on disk, but it should be interacted with via an interface,
// should include complete details w.r.t to how to sweep this output. In many cases,
// the SignDescriptor for the output should be sufficient.


// A new sub-system should be erected to manage the stray output pool, and periodically
// batch these outputs into single transaction to sweep back into the wallet.
// A later extension can allow other sub-systems to use this as an extension point
// to implement further batching when sweeping outputs from various contracts.


// StrayOutputsPool
type StrayOutputsPool interface {
	// AddInputs
	AddInputs(txVSize int64, inputs ...lnwallet.SpendableOutput) error

	// GenSweepTx
	GenSweepTx() (*btcutil.Tx, error)

	// Sweep
	Sweep() error
}


// PoolConfig
type PoolConfig struct {
	// DB provides access to the user's channels, allowing the breach
	// arbiter to determine the current state of a user's channels, and how
	// it should respond to channel closure.
	DB *channeldb.DB

	// Estimator is used by the breach arbiter to determine an appropriate
	// fee level when generating, signing, and broadcasting sweep
	// transactions.
	Estimator lnwallet.FeeEstimator

	// GenSweepScript generates the receiving scripts for swept outputs.
	GenSweepScript func() ([]byte, error)

	// Notifier provides a publish/subscribe interface for event driven
	// notifications regarding the confirmation of txids.
	Notifier chainntnfs.ChainNotifier

	// PublishTransaction facilitates the process of broadcasting a
	// transaction to the network.
	PublishTransaction func(*wire.MsgTx) error

	// Signer is used by the breach arbiter to generate sweep transactions,
	// which move coins from previously open channels back to the user's
	// wallet.
	Signer lnwallet.Signer
}