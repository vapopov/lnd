package outputspool

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/roasbeef/btcd/btcec"
	"github.com/roasbeef/btcd/chaincfg"
	"github.com/roasbeef/btcd/chaincfg/chainhash"
	"github.com/roasbeef/btcd/txscript"
	"github.com/roasbeef/btcd/wire"

	"github.com/lightningnetwork/lnd/channeldb"
	"github.com/lightningnetwork/lnd/lnwallet"
	"github.com/lightningnetwork/lnd/lnwallet/test"
)

var (
	testHash = chainhash.Hash{
		0xb7, 0x94, 0x38, 0x5f, 0x2d, 0x1e, 0xf7, 0xab,
		0x4d, 0x92, 0x73, 0xd1, 0x90, 0x63, 0x81, 0xb4,
		0x4f, 0x2f, 0x6f, 0x25, 0x88, 0xa3, 0xef, 0xb9,
		0x6a, 0x49, 0x18, 0x83, 0x31, 0x98, 0x47, 0x53,
	}

	sighDescriptor = lnwallet.SignDescriptor{
		SingleTweak: []byte{
			0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02,
			0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02,
			0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02,
			0x02, 0x02, 0x02, 0x02, 0x02,
		},
		WitnessScript: []byte{
			0x00, 0x14, 0xee, 0x91, 0x41, 0x7e, 0x85, 0x6c, 0xde,
			0x10, 0xa2, 0x91, 0x1e, 0xdc, 0xbd, 0xbd, 0x69, 0xe2,
			0xef, 0xb5, 0x71, 0x48,
		},
		Output: &wire.TxOut{
			Value: 5000000000,
			PkScript: []byte{
				0x41, // OP_DATA_65
				0x04, 0xd6, 0x4b, 0xdf, 0xd0, 0x9e, 0xb1, 0xc5,
				0xfe, 0x29, 0x5a, 0xbd, 0xeb, 0x1d, 0xca, 0x42,
				0x81, 0xbe, 0x98, 0x8e, 0x2d, 0xa0, 0xb6, 0xc1,
				0xc6, 0xa5, 0x9d, 0xc2, 0x26, 0xc2, 0x86, 0x24,
				0xe1, 0x81, 0x75, 0xe8, 0x51, 0xc9, 0x6b, 0x97,
				0x3d, 0x81, 0xb0, 0x1c, 0xc3, 0x1f, 0x04, 0x78,
				0x34, 0xbc, 0x06, 0xd6, 0xd6, 0xed, 0xf6, 0x20,
				0xd1, 0x84, 0x24, 0x1a, 0x6a, 0xed, 0x8b, 0x63,
				0xa6, // 65-byte signature
				0xac, // OP_CHECKSIG
			},
		},
		HashType: txscript.SigHashAll,
	}
)

func TestSerialization(t *testing.T) {
	encodeSample := &strayInput{
		txVSize: 10,
		totalAmt: 201,
		inputs: []lnwallet.SpendableOutput{
			&spendableOutput{
				amt: 100,
				outpoint: *wire.NewOutPoint(&testHash, 0),
				witnessType: lnwallet.CommitmentTimeLock,
				signDesc: sighDescriptor,
			},
			&spendableOutput{
				amt: 101,
				outpoint: *wire.NewOutPoint(&testHash, 1),
				witnessType: lnwallet.CommitmentTimeLock,
				signDesc: sighDescriptor,
			},
		},
	}

	buf := bytes.NewBuffer([]byte{})


	if err := encodeSample.Encode(buf); err != nil {
		t.Fatalf("stray output encode error: %v", err)
	}

	decodeSample := &strayInput{}

	if err := decodeSample.Decode(buf); err != nil {
		t.Fatalf("stray output decode error: %v", err)
	}

	spew.Dump(decodeSample)
}

func TestGenSweepScript(t *testing.T) {
	db, err := initDB()
	if err != nil {
		t.Fatal(err)
	}

	localFundingPrivKey, err := test.PrivkeyFromHex(
		"30ff4956bbdd3222d44cc5e8a1261dab1e07957bdac5ae88fe3261ef321f3749",
	)
	if err != nil {
		t.Fatalf("Failed to parse serialized privkey: %v", err)
	}

	localPaymentPrivKey, err := test.PrivkeyFromHex(
		"bb13b121cdc357cd2e608b0aea294afca36e2b34cf958e2e6451a2f274694491",
	)
	if err != nil {
		t.Fatalf("Failed to parse serialized privkey: %v", err)
	}

	pool := NewDiskStrayOutputsPool(&PoolConfig{
		DB: db,
		Estimator:      &lnwallet.StaticFeeEstimator{FeeRate: 50},
		GenSweepScript: func() ([]byte, error) { return nil, nil },
		Signer: &test.MockSigner{
			Privkeys: []*btcec.PrivateKey{
				localFundingPrivKey, localPaymentPrivKey,
			},
			NetParams: &chaincfg.RegressionNetParams,
		},
	})

	spew.Dump(pool.GenSweepTx())
}

func initDB() (*channeldb.DB, error) {
	tempPath, err := ioutil.TempDir("", "switchdb")
	if err != nil {
		return nil, err
	}

	db, err := channeldb.Open(tempPath)
	if err != nil {
		return nil, err
	}

	return db, nil
}
