package commands

// BaseCommand is the base for all commands.
type BaseCommand struct {
	ID string `json:"id"`
}

func (b *BaseCommand) GetID() string   { return b.ID }
func (b *BaseCommand) SetID(id string) { b.ID = id }
