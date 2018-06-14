package outputspool

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/coreos/bbolt"
	"github.com/lightningnetwork/lnd/lnwallet"
	"github.com/roasbeef/btcd/txscript"
	"github.com/roasbeef/btcd/wire"
	"github.com/roasbeef/btcutil"
)

var (
	// strayInputBucket is the name of the bucket within the database that
	// stores all data related to small outputs that better to combine to
	// minimise fee rate to proceed the transaction
	strayInputBucket = []byte("stray-input")

	// ErrNoStrayInputsCreated is returned when bucket of stray inputs
	// hasn't been created.
	ErrNoStrayInputsCreated = fmt.Errorf("there are no existing stray inputs")

	byteOrder = binary.BigEndian
)

type spendableOutput struct {
	amt         btcutil.Amount
	outpoint    wire.OutPoint
	witnessType lnwallet.WitnessType
	signDesc    lnwallet.SignDescriptor
}

func (s *spendableOutput) Amount() btcutil.Amount {
	return s.amt
}

func (s *spendableOutput) OutPoint() *wire.OutPoint {
	return &s.outpoint
}

func (s *spendableOutput) WitnessType() lnwallet.WitnessType {
	return s.witnessType
}

func (s *spendableOutput) SignDesc() *lnwallet.SignDescriptor {
	return &s.signDesc
}

func (s *spendableOutput) BuildWitness(signer lnwallet.Signer, txn *wire.MsgTx,
	hashCache *txscript.TxSigHashes, txinIdx int) ([][]byte, error) {
	return nil, nil
}


// AddStrayInput saves serialized stray input to database in order to combine
// them to one transaction to pay fee for one transaction.
func (d *DiskStrayOutputsPool) AddStrayInput(strayInput *strayInput) error {
	var b bytes.Buffer
	if err := strayInput.Encode(&b); err != nil {
		return err
	}

	return d.cfg.DB.Batch(func(tx *bolt.Tx) error {
		inputs, err := tx.CreateBucketIfNotExists(strayInputBucket)
		if err != nil {
			return err
		}

		inputID, err := inputs.NextSequence()
		if err != nil {
			return err
		}

		paymentIDBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(paymentIDBytes, inputID)

		return inputs.Put(paymentIDBytes, b.Bytes())
	})
}

// FetchAllStrayInputs returns all stray inputs in DB.
func (d *DiskStrayOutputsPool) FetchAllStrayInputs() ([]*strayInput, error) {
	var inputs []*strayInput
	err := d.cfg.DB.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(strayInputBucket)
		if bucket == nil {
			return ErrNoStrayInputsCreated
		}

		return bucket.ForEach(func(k, v []byte) error {
			input := &strayInput{}
			if err := input.Decode(bytes.NewReader(v)); err != nil {
				return err
			}

			inputs = append(inputs, input)

			return nil
		})
	})
	if err != nil {
		return nil, err
	}

	return inputs, nil
}

// StrayInput
type strayInput struct {
	//
	txVSize int64

	//
	totalAmt btcutil.Amount

	//
	inputs []lnwallet.SpendableOutput
}


// Encode
func (s *strayInput) Encode(w io.Writer) error {
	var scratch [8]byte

	byteOrder.PutUint64(scratch[:], uint64(s.txVSize))
	if _, err := w.Write(scratch[:]); err != nil {
		return err
	}

	byteOrder.PutUint64(scratch[:], uint64(s.totalAmt))
	if _, err := w.Write(scratch[:]); err != nil {
		return err
	}

	for _, input := range s.inputs {
		byteOrder.PutUint64(scratch[:], uint64(input.Amount()))
		if _, err := w.Write(scratch[:]); err != nil {
			return err
		}

		if err := writeOutpoint(w, input.OutPoint()); err != nil {
			return err
		}

		byteOrder.PutUint16(scratch[:2], uint16(input.WitnessType()))
		if _, err := w.Write(scratch[:2]); err != nil {
			return err
		}

		if err := lnwallet.WriteSignDescriptor(w, input.SignDesc()); err != nil {
			return err
		}
	}

	return nil
}

// Decode
func (s *strayInput) Decode(r io.Reader) error {
	var scratch [8]byte

	if _, err := r.Read(scratch[:]); err != nil {
		return err
	}
	s.txVSize = int64(byteOrder.Uint64(scratch[:]))

	if _, err := r.Read(scratch[:]); err != nil {
		return err
	}
	s.totalAmt = btcutil.Amount(byteOrder.Uint64(scratch[:]))

	var err error
	for {
		sOutput := &spendableOutput{}

		if _, err = r.Read(scratch[:]); err != nil {
			break
		}
		sOutput.amt = btcutil.Amount(byteOrder.Uint64(scratch[:]))

		if err = readOutpoint(io.LimitReader(r, 40),
			&sOutput.outpoint); err != nil {
			break
		}

		if _, err = r.Read(scratch[:2]); err != nil {
			break
		}
		sOutput.witnessType = lnwallet.WitnessType(
			byteOrder.Uint16(scratch[:2]),
		)

		if err = lnwallet.ReadSignDescriptor(r, &sOutput.signDesc); err != nil {
			break
		}

		s.inputs = append(s.inputs, sOutput)
	}
	if err != nil && err != io.EOF {
		return err
	}

	return nil
}

// writeOutpoint writes an outpoint to the passed writer using the minimal
// amount of bytes possible.
func writeOutpoint(w io.Writer, o *wire.OutPoint) error {
	if _, err := w.Write(o.Hash[:]); err != nil {
		return err
	}
	if err := binary.Write(w, byteOrder, o.Index); err != nil {
		return err
	}

	return nil
}

// readOutpoint reads an outpoint from the passed reader that was previously
// written using the writeOutpoint struct.
func readOutpoint(r io.Reader, o *wire.OutPoint) error {
	if _, err := io.ReadFull(r, o.Hash[:]); err != nil {
		return err
	}
	if err := binary.Read(r, byteOrder, &o.Index); err != nil {
		return err
	}

	return nil
}