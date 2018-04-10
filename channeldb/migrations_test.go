package channeldb

import (
	"crypto/sha256"
	"testing"
)

func TestPaymentStatusesMigration(t *testing.T) {
	t.Parallel()

	fakePayment := makeFakePayment()
	paymentHash := sha256.Sum256(fakePayment.PaymentPreimage[:])

	// add fake payment to test database and verify that it was created and we
	// have only one payment and theirs status is not "Completed"
	beforeMigrationFunc := func(d *DB) {
		if err := d.AddPayment(fakePayment); err != nil {
			t.Fatalf("unable to add payment: %v", err)
		}

		payments, err := d.FetchAllPayments()
		if err != nil {
			t.Fatalf("unable to fetch payments: %v", err)
		}

		if len(payments) != 1 {
			t.Fatalf("wrong qty of paymets: expected 1, got %v",
				len(payments))
		}

		paymentStatus, err := d.FetchPaymentStatus(paymentHash)
		if err != nil {
			t.Fatalf("unable to fetch payment status: %v", err)
		}

		// we should receive default status if we have any in database
		if paymentStatus != StatusGrounded {
			t.Fatalf("wrong payment status: expected %v, got %v",
				StatusGrounded.String(), paymentStatus.String())
		}
	}

	// verify that was created payment status "Completed" for our one fake
	// payment
	afterMigrationFunc := func(d *DB) {
		meta, err := d.FetchMeta(nil)
		if err != nil {
			t.Fatal(err)
		}

		if meta.DbVersionNumber != 1 {
			t.Fatal("migration 'paymentStatusesMigration' wasn't applied")
		}

		paymentStatus, err := d.FetchPaymentStatus(paymentHash)
		if err != nil {
			t.Fatalf("unable to fetch payment status: %v", err)
		}

		if paymentStatus != StatusCompleted {
			t.Fatalf("wrong payment status: expected %v, got %v",
				StatusCompleted.String(), paymentStatus.String())
		}
	}

	applyMigration(t,
		beforeMigrationFunc,
		afterMigrationFunc,
		paymentStatusesMigration,
		false)
}
