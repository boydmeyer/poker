package main

import (
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	g "xabbo.b7c.io/goearth"
	"xabbo.b7c.io/goearth/shockwave/in"
	"xabbo.b7c.io/goearth/shockwave/out"
)

// Initialize the extension with metadata
var ext = g.NewExt(g.ExtInfo{
	Title:       "Poker",
	Description: "An extension for managing, rolling, and resetting dice with automated poker hand evaluation and game interaction.",
	Author:      "Nanobyte", // "Nanobyte, Eduard"
	Version:     "1.3.1",
})

// Global variables for dice management, rolling state, mutex, and wait group
var (
	diceList         []*Dice
	isPokerRolling   bool
	isTriRolling     bool
	isClosing        bool
	ChatIsDisabled	 bool
	mutex            sync.Mutex
	resultsWaitGroup sync.WaitGroup
	rollDelay        = 800 * time.Millisecond
)

// Entry point of the application
func main() {
	ext.Intercept(out.CHAT, out.SHOUT, out.WHISPER).With(onChatMessage)
	ext.Intercept(out.THROW_DICE).With(handleThrowDice)
	ext.Intercept(out.DICE_OFF).With(handleDiceOff)
	ext.Intercept(in.DICE_VALUE).With(handleDiceResult)
	ext.Run()
}

// onChatMessage processes chat commands to trigger dice actions
func onChatMessage(e *g.Intercept) {
    msg := e.Packet.ReadString()
    

    // Process commands based on the message prefix and suffix
    if strings.HasPrefix(msg, ":") {		
		// Check if already rolling or closing
		if isPokerRolling || isTriRolling || isClosing {
			log.Println("Already rolling or closing...")
			e.Block()
			return
		}

		command := strings.TrimPrefix(msg, ":")
        switch {
        case strings.HasSuffix(command, "reset"):
			e.Block()
			resetDiceState()
        case strings.HasSuffix(command, "roll"):
			e.Block()
			isPokerRolling = true
			go rollPokerDice()
        case strings.HasSuffix(command, "close"):
			e.Block()
			go closeAllDice()
        case strings.HasSuffix(command, "tri"):
			e.Block()
			isTriRolling = true
			go rollTriDice()
		case strings.HasSuffix(command, "chaton"):
			e.Block()
			ChatIsDisabled = false
		case strings.HasSuffix(command, "chatoff"):
			e.Block()
			ChatIsDisabled = true
        }
    }
}

// Reset all saved dice states
func resetDiceState() {
	mutex.Lock()
	defer mutex.Unlock()
	resultsWaitGroup.Wait() // Ensure all dice roll results are processed
	diceList = []*Dice{}
	isPokerRolling, isTriRolling, isClosing = false, false, false
}

// Handle the throwing of a dice
func handleThrowDice(e *g.Intercept) {
	packet := e.Packet
	rawData := string(packet.Data)
	logrus.WithFields(logrus.Fields{"raw_data": rawData}).Debug("Raw packet data")

	diceData := strings.Fields(rawData)
	diceIDStr := diceData[0]
	diceID, err := strconv.Atoi(diceIDStr)
	if err != nil {
		logrus.WithFields(logrus.Fields{"dice_id_str": diceIDStr, "error": err}).Warn("Failed to parse dice ID")
		return
	}

	mutex.Lock()
	defer mutex.Unlock()

	// Search for a dice with the given ID in the list
	var existingDice *Dice
	for _, dice := range diceList {
		if dice != nil && dice.ID == diceID {
			existingDice = dice
			break
		}
	}

	// If not found and the list has fewer than 5 dice, create and add a new one
	if existingDice == nil && len(diceList) < 5 {
		newDice := &Dice{ID: diceID, IsRolling: true, IsClosed: false}
		diceList = append(diceList, newDice)
		log.Printf("Dice %d added\n", diceID)
	}
}

// Handle the turning off of a dice
func handleDiceOff(e *g.Intercept) {
	packet := e.Packet
	diceIDStr := string(packet.Data)

	diceID, err := strconv.Atoi(diceIDStr)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"dice_id_str": diceIDStr,
			"error":       err,
		}).Warn("Failed to parse dice ID")
		return
	}

	mutex.Lock()
	defer mutex.Unlock()

	// Search for a dice with the given ID in the list
	var existingDice *Dice
	for _, dice := range diceList {
		if dice != nil && dice.ID == diceID {
			existingDice = dice
			break
		}
	}

	// If not found and the list has fewer than 5 dice, create and add a new one
	if existingDice == nil && len(diceList) < 5 {
		newDice := &Dice{ID: diceID, IsRolling: false, IsClosed: true}
		diceList = append(diceList, newDice)
		log.Printf("Dice %d added\n", diceID)
	}
}

// Handle the result of a dice roll
func handleDiceResult(e *g.Intercept) {
	packet := e.Packet
	rawData := string(packet.Data)
	logrus.WithFields(logrus.Fields{"raw_data": rawData}).Debug("Raw packet data")

	diceData := strings.Fields(rawData)
	if len(diceData) < 2 {
		return
	}

	diceIDStr := diceData[0]
	diceID, err := strconv.Atoi(diceIDStr)
	if err != nil {
		logrus.WithFields(logrus.Fields{"dice_id_str": diceIDStr, "error": err}).Warn("Failed to parse dice ID")
		return
	}

	diceValueStr := diceData[1]
	diceValue, err := strconv.Atoi(diceValueStr)
	if err != nil {
		logrus.WithFields(logrus.Fields{"dice_value_str": diceValueStr, "error": err}).Warn("Failed to parse dice value")
		return
	}
	adjustedDiceValue := diceValue - (diceID * 38)


	mutex.Lock()
	for i, dice := range diceList {
		if dice.ID == diceID {
			if dice.IsRolling && (isPokerRolling || isTriRolling) {
				dice.IsRolling = false
				resultsWaitGroup.Done()
			}
			diceList[i].Value = adjustedDiceValue
			diceList[i].IsClosed = diceList[i].Value == 0

			if isPokerRolling || isTriRolling {
				log.Printf("Dice %d rolled: %d\n", diceID, adjustedDiceValue)
			}
			break
		}
	}
	mutex.Unlock()
}

// Close the dice and send the packets to the game server
func closeAllDice() {
	mutex.Lock()
	defer mutex.Unlock()
	isClosing = true
	resultsWaitGroup.Add(len(diceList))
	for _, dice := range diceList {
		dice.Close()
		time.Sleep(rollDelay)
		resultsWaitGroup.Done()
	}
	resultsWaitGroup.Wait()
	isClosing = false
}

// Roll the poker dice by sending packets and waiting for results
func rollPokerDice() {
	mutex.Lock()

	if len(diceList) < 5 {
		mutex.Unlock()
		log.Println("Not enough dice to roll")
		isPokerRolling = false
		return
	}

	resultsWaitGroup.Add(len(diceList))
	mutex.Unlock()

	for _, dice := range diceList {
		dice.Roll()
		time.Sleep(rollDelay)
	}

	time.Sleep(1000 * time.Millisecond)
	resultsWaitGroup.Wait()
	evaluatePokerHand()
	isPokerRolling = false
}

// Evaluate the poker hand and send the result to the chat
func rollTriDice() {
	mutex.Lock()

	if len(diceList) < 5 {
		mutex.Unlock()
		log.Println("Not enough dice to roll")
		isTriRolling = false
		return
	}

	resultsWaitGroup.Add(3)
	mutex.Unlock()

	for _, index := range []int{0, 2, 4} {
		diceList[index].Roll()
		time.Sleep(rollDelay)
	}

	time.Sleep(1000 * time.Millisecond)
	resultsWaitGroup.Wait()

	evaluateTriHand()
	isTriRolling = false
}