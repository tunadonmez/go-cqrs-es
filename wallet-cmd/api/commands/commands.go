package commands

import (
	corecommands "github.com/tunadonmez/go-cqrs-es/cqrs-core/commands"
)

type CreateWalletCommand struct {
	corecommands.BaseCommand
	Owner          string  `json:"owner" binding:"required"`
	Currency       string  `json:"currency" binding:"required"`
	OpeningBalance float64 `json:"openingBalance"`
}

type CreditWalletCommand struct {
	corecommands.BaseCommand
	Amount      float64 `json:"amount" binding:"required"`
	Reference   string  `json:"reference"`
	Description string  `json:"description"`
}

type DebitWalletCommand struct {
	corecommands.BaseCommand
	Amount      float64 `json:"amount" binding:"required"`
	Reference   string  `json:"reference"`
	Description string  `json:"description"`
}

type TransferFundsCommand struct {
	corecommands.BaseCommand
	DestinationWalletID string  `json:"destinationWalletId" binding:"required"`
	Amount              float64 `json:"amount" binding:"required"`
	Reference           string  `json:"reference"`
	Description         string  `json:"description"`
}
