package commands

import (
	"github.com/techbank/account-common/dto"
	"github.com/techbank/cqrs-core/commands"
)

type OpenAccountCommand struct {
	commands.BaseCommand
	AccountHolder  string          `json:"accountHolder" binding:"required"`
	AccountType    dto.AccountType `json:"accountType" binding:"required"`
	OpeningBalance float64         `json:"openingBalance" binding:"required"`
}

type DepositFundsCommand struct {
	commands.BaseCommand
	Amount float64 `json:"amount" binding:"required"`
}

type WithdrawFundsCommand struct {
	commands.BaseCommand
	Amount float64 `json:"amount" binding:"required"`
}

type CloseAccountCommand struct {
	commands.BaseCommand
}
