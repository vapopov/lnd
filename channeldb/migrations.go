package channeldb

import (
	"bytes"
	"crypto/sha256"

	"github.com/coreos/bbolt"
)

// paymentStatusesMigration is a database migration intended for adding payment
// statuses for each existing payment entity in bucket to be able control
// transitions of statuses and prevent cases such as double payment
func paymentStatusesMigration(tx *bolt.Tx) error {
	// Get the bucket dedicated to storing payments
	bucket := tx.Bucket(paymentBucket)
	if bucket == nil {
		return ErrNoPaymentsCreated
	}

	// Get the bucket dedicated to storing statuses of payments,
	// where a key is payment hash, value is payment status
	paymentStatuses, err := tx.CreateBucketIfNotExists(paymentStatusBucket)
	if err != nil {
		return err
	}

	log.Infof("Migration database adds to all existing payments " +
		"statuses as Completed")

	// For each payment in the bucket, fetch all data.
	return bucket.ForEach(func(k, v []byte) error {
		// ignores if it is sub-bucket
		if v == nil {
			return nil
		}

		r := bytes.NewReader(v)
		payment, err := deserializeOutgoingPayment(r)
		if err != nil {
			return err
		}

		// calculate payment hash for current payment
		paymentHash := sha256.Sum256(payment.PaymentPreimage[:])

		// tries to update status for current payment to completed
		// if it fails - migration abort transaction and return payment bucket
		// to previous state
		return paymentStatuses.Put(paymentHash[:], StatusCompleted.Bytes())
	})
}
