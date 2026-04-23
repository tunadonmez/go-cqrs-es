package dto

type TransactionType string

const (
	TransactionTypeOpeningBalance TransactionType = "OPENING_BALANCE"
	TransactionTypeCredit         TransactionType = "CREDIT"
	TransactionTypeDebit          TransactionType = "DEBIT"
	TransactionTypeTransferIn     TransactionType = "TRANSFER_IN"
	TransactionTypeTransferOut    TransactionType = "TRANSFER_OUT"
)
