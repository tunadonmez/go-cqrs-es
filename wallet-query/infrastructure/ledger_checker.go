package infrastructure

import (
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strings"

	"github.com/tunadonmez/go-cqrs-es/wallet-common/dto"
	"github.com/tunadonmez/go-cqrs-es/wallet-query/domain"
)

const ledgerCheckTolerance = 0.000001

type LedgerConsistencyChecker struct {
	repository *WalletRepository
}

func NewLedgerConsistencyChecker(repository *WalletRepository) *LedgerConsistencyChecker {
	return &LedgerConsistencyChecker{repository: repository}
}

type LedgerCheckReport struct {
	Status               string
	WalletsChecked       int
	MovementsChecked     int
	LedgerEntriesChecked int
	WalletMismatches     []WalletBalanceMismatch
	MovementIssues       []MovementIssue
	TransferIssues       []TransferIssue
	GlobalSummary        GlobalLedgerSummary
	Warnings             []string
}

type WalletBalanceMismatch struct {
	WalletID        string
	StoredBalance   float64
	ComputedBalance float64
	Difference      float64
}

type MovementIssue struct {
	MovementID         string
	StoredStatus       string
	StoredDebitTotal   float64
	StoredCreditTotal  float64
	DerivedDebitTotal  float64
	DerivedCreditTotal float64
	StoredEntryCount   int
	DerivedEntryCount  int
	Currencies         []string
	Reason             string
}

type TransferIssue struct {
	MovementID string
	Reason     string
	EntryCount int
}

type GlobalLedgerSummary struct {
	DebitTotal                   float64
	CreditTotal                  float64
	Difference                   float64
	ExternalMoneyFlowDetected    bool
	ExternalMoneyFlowExplanation string
}

func (c *LedgerConsistencyChecker) Run() (*LedgerCheckReport, error) {
	slog.Info("Ledger consistency check started", "component", "ledger-checker")

	wallets, err := c.repository.FindAllWalletsForCheck()
	if err != nil {
		return nil, err
	}
	entries, err := c.repository.FindAllLedgerEntriesForCheck()
	if err != nil {
		return nil, err
	}
	movements, err := c.repository.FindAllLedgerMovementsForCheck()
	if err != nil {
		return nil, err
	}

	report := &LedgerCheckReport{
		WalletsChecked:       len(wallets),
		MovementsChecked:     len(movements),
		LedgerEntriesChecked: len(entries),
	}

	balancesByWallet := make(map[string]float64)
	movementGroups := make(map[string][]*domain.LedgerEntry)
	missingMovementIDCount := 0
	for _, entry := range entries {
		switch entry.EntryType {
		case domain.LedgerEntryTypeCredit:
			balancesByWallet[entry.WalletID] += entry.Amount
			report.GlobalSummary.CreditTotal += entry.Amount
		case domain.LedgerEntryTypeDebit:
			balancesByWallet[entry.WalletID] -= entry.Amount
			report.GlobalSummary.DebitTotal += entry.Amount
		}

		if isExternalMoneyFlowType(entry.TransactionType) {
			report.GlobalSummary.ExternalMoneyFlowDetected = true
		}
		if entry.MovementID == "" {
			missingMovementIDCount++
			continue
		}
		movementGroups[entry.MovementID] = append(movementGroups[entry.MovementID], entry)
	}
	if missingMovementIDCount > 0 {
		report.Warnings = append(report.Warnings, fmt.Sprintf("%d ledger entrie(s) have no movement_id; run replay to rebuild explicit journal rows", missingMovementIDCount))
	}

	for _, wallet := range wallets {
		computed := balancesByWallet[wallet.ID]
		diff := wallet.Balance - computed
		if math.Abs(diff) > ledgerCheckTolerance {
			report.WalletMismatches = append(report.WalletMismatches, WalletBalanceMismatch{
				WalletID:        wallet.ID,
				StoredBalance:   wallet.Balance,
				ComputedBalance: computed,
				Difference:      diff,
			})
		}
	}

	for _, movement := range movements {
		group := movementGroups[movement.ID]
		var debitTotal, creditTotal float64
		currencies := map[string]struct{}{}
		debitCount := 0
		creditCount := 0

		for _, entry := range group {
			currencies[entry.Currency] = struct{}{}
			if entry.EntryType == domain.LedgerEntryTypeDebit {
				debitTotal += entry.Amount
				debitCount++
			}
			if entry.EntryType == domain.LedgerEntryTypeCredit {
				creditTotal += entry.Amount
				creditCount++
			}
		}

		if len(group) == 0 {
			report.MovementIssues = append(report.MovementIssues, MovementIssue{
				MovementID:        movement.ID,
				StoredStatus:      movement.Status,
				StoredDebitTotal:  movement.TotalDebit,
				StoredCreditTotal: movement.TotalCredit,
				StoredEntryCount:  movement.EntryCount,
				DerivedEntryCount: 0,
				Currencies:        nil,
				Reason:            "movement row has no linked ledger entries",
			})
			continue
		}

		if math.Abs(movement.TotalDebit-debitTotal) > ledgerCheckTolerance ||
			math.Abs(movement.TotalCredit-creditTotal) > ledgerCheckTolerance ||
			movement.EntryCount != len(group) ||
			len(currencies) > 1 {
			reportMovementMismatch(movement, debitTotal, creditTotal, len(group), sortedCurrencyList(currencies))
			report.MovementIssues = append(report.MovementIssues, MovementIssue{
				MovementID:         movement.ID,
				StoredStatus:       movement.Status,
				StoredDebitTotal:   movement.TotalDebit,
				StoredCreditTotal:  movement.TotalCredit,
				DerivedDebitTotal:  debitTotal,
				DerivedCreditTotal: creditTotal,
				StoredEntryCount:   movement.EntryCount,
				DerivedEntryCount:  len(group),
				Currencies:         sortedCurrencyList(currencies),
				Reason:             "stored movement totals do not match linked ledger entries",
			})
		}

		if movement.MovementType != domain.LedgerMovementTypeTransfer {
			continue
		}

		if len(group) != 2 || debitCount != 1 || creditCount != 1 {
			report.TransferIssues = append(report.TransferIssues, TransferIssue{
				MovementID: movement.ID,
				Reason:     "expected exactly one debit entry and one credit entry",
				EntryCount: len(group),
			})
			continue
		}

		first := group[0]
		second := group[1]
		if first.CounterpartyWalletID != second.WalletID || second.CounterpartyWalletID != first.WalletID {
			report.TransferIssues = append(report.TransferIssues, TransferIssue{
				MovementID: movement.ID,
				Reason:     "counterparty wallet pairing is inconsistent",
				EntryCount: len(group),
			})
		}
		if movement.SourceWalletID == "" || movement.DestinationWalletID == "" {
			report.TransferIssues = append(report.TransferIssues, TransferIssue{
				MovementID: movement.ID,
				Reason:     "movement row is missing source or destination wallet metadata",
				EntryCount: len(group),
			})
		}
	}

	keys := make([]string, 0, len(movementGroups))
	for key := range movementGroups {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		found := false
		for _, movement := range movements {
			if movement.ID == key {
				found = true
				break
			}
		}
		if found {
			continue
		}
		group := movementGroups[key]
		report.MovementIssues = append(report.MovementIssues, MovementIssue{
			MovementID:         key,
			DerivedDebitTotal:  sumEntriesByType(group, domain.LedgerEntryTypeDebit),
			DerivedCreditTotal: sumEntriesByType(group, domain.LedgerEntryTypeCredit),
			DerivedEntryCount:  len(group),
			Currencies:         entryCurrencies(group),
			Reason:             "ledger entries reference a movement_id that has no ledger_movements row",
		})
		reportMovementMismatch(&domain.LedgerMovement{ID: key}, sumEntriesByType(group, domain.LedgerEntryTypeDebit), sumEntriesByType(group, domain.LedgerEntryTypeCredit), len(group), entryCurrencies(group))
	}

	report.GlobalSummary.Difference = report.GlobalSummary.CreditTotal - report.GlobalSummary.DebitTotal
	if report.GlobalSummary.ExternalMoneyFlowDetected {
		report.GlobalSummary.ExternalMoneyFlowExplanation = "global debit/credit equality is not required because external deposits, withdrawals, or opening balances add or remove money from the wallet system"
		if math.Abs(report.GlobalSummary.Difference) > ledgerCheckTolerance {
			report.Warnings = append(report.Warnings, report.GlobalSummary.ExternalMoneyFlowExplanation)
		}
	} else if math.Abs(report.GlobalSummary.Difference) > ledgerCheckTolerance {
		report.MovementIssues = append(report.MovementIssues, MovementIssue{
			MovementID:         "GLOBAL",
			DerivedDebitTotal:  report.GlobalSummary.DebitTotal,
			DerivedCreditTotal: report.GlobalSummary.CreditTotal,
			DerivedEntryCount:  report.LedgerEntriesChecked,
			Reason:             "global debit/credit totals do not balance",
		})
	}

	report.Status = "OK"
	if len(report.Warnings) > 0 {
		report.Status = "WARNING"
	}
	if report.HasFailures() {
		report.Status = "FAILED"
	}

	attrs := []any{
		"component", "ledger-checker",
		"status", report.Status,
		"walletsChecked", report.WalletsChecked,
		"movementsChecked", report.MovementsChecked,
		"ledgerEntriesChecked", report.LedgerEntriesChecked,
		"walletMismatches", len(report.WalletMismatches),
		"movementIssues", len(report.MovementIssues),
		"transferIssues", len(report.TransferIssues),
		"warnings", len(report.Warnings),
	}
	if report.HasFailures() {
		slog.Error("Ledger consistency check failed", attrs...)
	} else {
		slog.Info("Ledger consistency check completed", attrs...)
	}

	return report, nil
}

func (r *LedgerCheckReport) HasFailures() bool {
	return len(r.WalletMismatches) > 0 || len(r.MovementIssues) > 0 || len(r.TransferIssues) > 0
}

func (r *LedgerCheckReport) ExitCode() int {
	if r.HasFailures() {
		return 1
	}
	return 0
}

func (r *LedgerCheckReport) String() string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Ledger Consistency Check: %s\n", r.Status))
	builder.WriteString(fmt.Sprintf("Wallets checked: %d\n", r.WalletsChecked))
	builder.WriteString(fmt.Sprintf("Movements checked: %d\n", r.MovementsChecked))
	builder.WriteString(fmt.Sprintf("Ledger entries checked: %d\n", r.LedgerEntriesChecked))
	builder.WriteString(fmt.Sprintf("Wallet mismatches: %d\n", len(r.WalletMismatches)))
	builder.WriteString(fmt.Sprintf("Movement issues: %d\n", len(r.MovementIssues)))
	builder.WriteString(fmt.Sprintf("Transfer issues: %d\n", len(r.TransferIssues)))
	builder.WriteString(fmt.Sprintf("Warnings: %d\n", len(r.Warnings)))
	builder.WriteString("\n")

	builder.WriteString("Global totals\n")
	builder.WriteString(fmt.Sprintf("- total debit: %.2f\n", r.GlobalSummary.DebitTotal))
	builder.WriteString(fmt.Sprintf("- total credit: %.2f\n", r.GlobalSummary.CreditTotal))
	builder.WriteString(fmt.Sprintf("- difference: %.2f\n", r.GlobalSummary.Difference))
	if r.GlobalSummary.ExternalMoneyFlowExplanation != "" {
		builder.WriteString(fmt.Sprintf("- note: %s\n", r.GlobalSummary.ExternalMoneyFlowExplanation))
	}

	appendSection(&builder, "Wallet balance mismatches", len(r.WalletMismatches) == 0, func() {
		for _, mismatch := range r.WalletMismatches {
			builder.WriteString(fmt.Sprintf(
				"- walletId=%s stored=%.2f computed=%.2f difference=%.2f\n",
				mismatch.WalletID,
				mismatch.StoredBalance,
				mismatch.ComputedBalance,
				mismatch.Difference,
			))
		}
	})

	appendSection(&builder, "Movement issues", len(r.MovementIssues) == 0, func() {
		for _, issue := range r.MovementIssues {
			builder.WriteString(fmt.Sprintf(
				"- movement=%s storedStatus=%s storedDebit=%.2f storedCredit=%.2f derivedDebit=%.2f derivedCredit=%.2f storedEntries=%d derivedEntries=%d currencies=%s reason=%s\n",
				issue.MovementID,
				issue.StoredStatus,
				issue.StoredDebitTotal,
				issue.StoredCreditTotal,
				issue.DerivedDebitTotal,
				issue.DerivedCreditTotal,
				issue.StoredEntryCount,
				issue.DerivedEntryCount,
				strings.Join(issue.Currencies, ","),
				issue.Reason,
			))
		}
	})

	appendSection(&builder, "Transfer issues", len(r.TransferIssues) == 0, func() {
		for _, issue := range r.TransferIssues {
			builder.WriteString(fmt.Sprintf(
				"- movement=%s entries=%d reason=%s\n",
				issue.MovementID,
				issue.EntryCount,
				issue.Reason,
			))
		}
	})

	appendSection(&builder, "Warnings", len(r.Warnings) == 0, func() {
		for _, warning := range r.Warnings {
			builder.WriteString(fmt.Sprintf("- %s\n", warning))
		}
	})

	return builder.String()
}

func appendSection(builder *strings.Builder, title string, empty bool, render func()) {
	builder.WriteString("\n")
	builder.WriteString(title)
	builder.WriteString("\n")
	if empty {
		builder.WriteString("- none\n")
		return
	}
	render()
}

func reportMovementMismatch(movement *domain.LedgerMovement, derivedDebit, derivedCredit float64, derivedEntries int, currencies []string) {
	slog.Error("Ledger movement consistency mismatch",
		"component", "ledger-checker",
		"movementId", movement.ID,
		"storedStatus", movement.Status,
		"storedDebitTotal", movement.TotalDebit,
		"storedCreditTotal", movement.TotalCredit,
		"storedEntryCount", movement.EntryCount,
		"derivedDebitTotal", derivedDebit,
		"derivedCreditTotal", derivedCredit,
		"derivedEntryCount", derivedEntries,
		"currencies", strings.Join(currencies, ","))
}

func sortedCurrencyList(values map[string]struct{}) []string {
	items := make([]string, 0, len(values))
	for value := range values {
		items = append(items, value)
	}
	sort.Strings(items)
	return items
}

func entryCurrencies(entries []*domain.LedgerEntry) []string {
	values := make(map[string]struct{})
	for _, entry := range entries {
		values[entry.Currency] = struct{}{}
	}
	return sortedCurrencyList(values)
}

func sumEntriesByType(entries []*domain.LedgerEntry, entryType string) float64 {
	var total float64
	for _, entry := range entries {
		if entry.EntryType == entryType {
			total += entry.Amount
		}
	}
	return total
}

func isExternalMoneyFlowType(transactionType dto.TransactionType) bool {
	return transactionType == dto.TransactionTypeOpeningBalance ||
		transactionType == dto.TransactionTypeCredit ||
		transactionType == dto.TransactionTypeDebit
}
