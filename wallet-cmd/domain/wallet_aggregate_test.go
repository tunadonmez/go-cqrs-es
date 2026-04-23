package domain

import (
	"testing"

	"github.com/tunadonmez/go-cqrs-es/wallet-cmd/api/commands"
	"github.com/tunadonmez/go-cqrs-es/wallet-common/dto"
	commonevents "github.com/tunadonmez/go-cqrs-es/wallet-common/events"
)

func TestWalletAggregate_Debit_rejectsInsufficientFunds(t *testing.T) {
	cmd := commands.CreateWalletCommand{
		Owner:          "Alice",
		Currency:       "USD",
		OpeningBalance: 20,
	}
	cmd.SetID("wallet-1")

	wallet, err := NewWalletAggregateFromCommand(&cmd)
	if err != nil {
		t.Fatalf("setup wallet: %v", err)
	}
	wallet.MarkChangesAsCommitted()

	err = wallet.Debit(25, "debit-1", "test debit")
	if err == nil || err.Error() != "debit declined, insufficient funds" {
		t.Fatalf("expected insufficient funds error, got %v", err)
	}
	if got := len(wallet.GetUncommittedChanges()); got != 0 {
		t.Fatalf("expected no uncommitted changes after rejected debit, got %d", got)
	}
}

func TestWalletAggregate_TransferTo_recordsMatchingLedgerEvents(t *testing.T) {
	sourceCmd := commands.CreateWalletCommand{
		Owner:          "Alice",
		Currency:       "USD",
		OpeningBalance: 100,
	}
	sourceCmd.SetID("wallet-source")

	destinationCmd := commands.CreateWalletCommand{
		Owner:          "Bob",
		Currency:       "USD",
		OpeningBalance: 10,
	}
	destinationCmd.SetID("wallet-destination")

	sourceWallet, err := NewWalletAggregateFromCommand(&sourceCmd)
	if err != nil {
		t.Fatalf("setup source wallet: %v", err)
	}
	destinationWallet, err := NewWalletAggregateFromCommand(&destinationCmd)
	if err != nil {
		t.Fatalf("setup destination wallet: %v", err)
	}
	sourceWallet.MarkChangesAsCommitted()
	destinationWallet.MarkChangesAsCommitted()

	if err := sourceWallet.TransferTo(destinationWallet, 25, "transfer-1", "invoice payout"); err != nil {
		t.Fatalf("transfer: %v", err)
	}

	sourceChanges := sourceWallet.GetUncommittedChanges()
	if len(sourceChanges) != 1 {
		t.Fatalf("expected 1 source change, got %d", len(sourceChanges))
	}
	sourceDebit, ok := sourceChanges[0].(*commonevents.WalletDebitedEvent)
	if !ok {
		t.Fatalf("expected WalletDebitedEvent, got %T", sourceChanges[0])
	}
	if sourceDebit.TransactionType != dto.TransactionTypeTransferOut {
		t.Fatalf("expected %s, got %s", dto.TransactionTypeTransferOut, sourceDebit.TransactionType)
	}
	if sourceDebit.CounterpartyWalletID != destinationWallet.ID {
		t.Fatalf("expected destination wallet ID %q, got %q", destinationWallet.ID, sourceDebit.CounterpartyWalletID)
	}

	destinationChanges := destinationWallet.GetUncommittedChanges()
	if len(destinationChanges) != 1 {
		t.Fatalf("expected 1 destination change, got %d", len(destinationChanges))
	}
	destinationCredit, ok := destinationChanges[0].(*commonevents.WalletCreditedEvent)
	if !ok {
		t.Fatalf("expected WalletCreditedEvent, got %T", destinationChanges[0])
	}
	if destinationCredit.TransactionType != dto.TransactionTypeTransferIn {
		t.Fatalf("expected %s, got %s", dto.TransactionTypeTransferIn, destinationCredit.TransactionType)
	}
	if destinationCredit.CounterpartyWalletID != sourceWallet.ID {
		t.Fatalf("expected source wallet ID %q, got %q", sourceWallet.ID, destinationCredit.CounterpartyWalletID)
	}
}
