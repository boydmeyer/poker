package main

import (
	"errors"
	"fmt"

	"xabbo.b7c.io/goearth/shockwave/out"
)

// Dice struct represents a dice with its ID, value
type Dice struct {
	ID    int
	Value     int
	IsClosed  bool
	IsRolling bool
}

// Roll the dice
func (d *Dice) Roll() error {
	if d.ID == 0 {
		return errors.New("no dice id")
	}

	ext.Send(out.THROW_DICE, []byte(fmt.Sprintf("%d", d.ID)))
	d.IsRolling = true
	d.IsClosed = false
	return nil
}

// Close the dice
func (d *Dice) Close() error {
	if d.ID == 0 {
		return errors.New("no dice id")
	}

	// Send the throw dice packet
	ext.Send(out.DICE_OFF, []byte(fmt.Sprintf("%d", d.ID)))
	d.IsClosed = true
	return nil
}

// IsValid checks if the dice is valid
func (d *Dice) IsValid() bool {
	return d.ID != 0
}
