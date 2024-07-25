package main

import (
	"fmt"
	"log"
	"sort"
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
	Author:      "Nanobyte",
	Version:     "1.2",
})

// Dice struct represents a dice with its ID, value, and packets for throwing and turning off
type Dice struct {
	Id    int
	Value int
}

// Variables to manage dice, rolling state, mutex for setup, and wait group for results
var (
	diceArray        []*Dice
	rolling          bool
	closing          bool
	setupMutex       sync.Mutex
	resultsWaitGroup sync.WaitGroup
	delay            = 600 * time.Millisecond
)

// Entry point of the application
func main() {
	ext.Intercept(out.CHAT, out.SHOUT, out.WHISPER).With(handleChat)
	ext.Intercept(out.THROW_DICE).With(handleThrowDice)
	ext.Intercept(out.DICE_OFF).With(handleDiceOff)
	ext.Intercept(in.DICE_VALUE).With(handleDiceResult)
	ext.Run()
}

// Handle chat messages to trigger dice actions
func handleChat(e *g.Intercept) {
	msg := e.Packet.ReadString()
	if strings.HasPrefix(msg, ":") {
		e.Block()
		if strings.HasSuffix(msg, "roll") {
			e.Block()
			log.Println(msg)

			if rolling || closing {
				log.Println("Busy...")
				return
			}

			rolling = true
			go rollDice()
		} else if strings.HasSuffix(msg, "close") {
			log.Println(msg)

			if rolling || closing {
				log.Println("Busy...")
				return
			}

			go closeDice()
		} else if strings.HasSuffix(msg, "reset") {
			e.Block()
			log.Println(msg)
			resetPackets()
		}
	}
}

// Reset the saved dice packets
func resetPackets() {
	// Remove unnecessary locking and unlocking of the mutex
	diceArray = []*Dice{}
	rolling = false
	closing = false
	log.Println("All saved packets reset")
}

// Handle the throwing of a dice
func handleThrowDice(e *g.Intercept) {
	packet := e.Packet
	rawString := string(packet.Data)
	logrus.WithFields(logrus.Fields{"raw_data": rawString}).Debug("Raw packet data")

	diceValueStrings := strings.Fields(rawString)
	diceIDStr := diceValueStrings[0]
	diceID, err := strconv.Atoi(diceIDStr)
	if err != nil {
		logrus.WithFields(logrus.Fields{"dice_id_str": diceIDStr, "error": err}).Warn("Failed to parse dice ID")
		return
	}

	setupMutex.Lock()
	defer setupMutex.Unlock()

	var dice *Dice
	for _, d := range diceArray {
		if d != nil && d.Id == diceID {
			dice = d
			break
		}
	}

	if dice == nil && len(diceArray) < 5 {
		dice = &Dice{
			Id: diceID,
		}
		diceArray = append(diceArray, dice)
		log.Printf("Dice %d added\n", dice.Id)
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

	setupMutex.Lock()
	defer setupMutex.Unlock()

	var dice *Dice
	for _, d := range diceArray {
		if d.Id == diceID {
			log.Printf("Dice %d off added\n", diceID)
			dice = d
			break
		}
	}

	if dice == nil && len(diceArray) < 5 {
		dice = &Dice{
			Id: diceID,
		}
		diceArray = append(diceArray, dice)
		log.Printf("Dice %d added\n", dice.Id)
	}
}

// Handle the result of a dice roll
func handleDiceResult(e *g.Intercept) {
	packet := e.Packet
	rawString := string(packet.Data)
	logrus.WithFields(logrus.Fields{"raw_data": rawString}).Debug("Raw packet data")

	diceValueStrings := strings.Fields(rawString)
	if len(diceValueStrings) < 2 {
		return
	}

	diceIDStr := diceValueStrings[0]
	diceID, err := strconv.Atoi(diceIDStr)
	if err != nil {
		logrus.WithFields(logrus.Fields{"dice_id_str": diceIDStr, "error": err}).Warn("Failed to parse dice ID")
		return
	}

	diceValueStr := diceValueStrings[1]
	diceValue, err := strconv.Atoi(diceValueStr)
	if err != nil {
		logrus.WithFields(logrus.Fields{"dice_value_str": diceValueStr, "error": err}).Warn("Failed to parse dice value")
		return
	}
	diceRollValue := diceValue - (diceID * 38)

	setupMutex.Lock()
	for i, dice := range diceArray {
		if dice.Id == diceID {
			diceArray[i].Value = diceRollValue
			if rolling {
				log.Printf("Dice %d rolled: %d\n", diceID, diceRollValue)
				resultsWaitGroup.Done()
			}
			break
		}
	}
	setupMutex.Unlock()
}

// Close the dice and send the packets to the game server
func closeDice() {
	setupMutex.Lock()

	if len(diceArray) < 5 {
		setupMutex.Unlock()
		rolling = false
		log.Println("Not enough dice to close")
		return
	}
	closing = true
	for _, dice := range diceArray {
		if dice != nil {
			ext.Send(out.DICE_OFF, []byte(strconv.Itoa(dice.Id)))
			time.Sleep(delay)
		}
	}
	closing = false
}

// Roll the dice by sending packets and waiting for results
func rollDice() {
	setupMutex.Lock()

	if len(diceArray) < 5 {
		setupMutex.Unlock()
		rolling = false
		log.Println("Not enough dice to roll")
		return
	}

	validDiceCount := 0
	for _, dice := range diceArray {
		if dice != nil {
			validDiceCount++
		}
	}
	if validDiceCount == 0 {
		setupMutex.Unlock()
		log.Println("No valid dice to roll")
		return
	}

	resultsWaitGroup.Add(validDiceCount)
	setupMutex.Unlock()

	for _, dice := range diceArray {
		if dice != nil {
			ext.Send(out.THROW_DICE, []byte(strconv.Itoa(dice.Id)))
			time.Sleep(delay)
		}
	}

	time.Sleep(1000 * time.Millisecond)
	resultsWaitGroup.Wait()
	waitForAllResults()
}

// Wait for all dice results and evaluate the hand
func waitForAllResults() {
	rolling = false
	hand := toPokerString(diceArray)
	log.Printf("Hand: %s\n", hand)
	ext.Send(out.SHOUT, hand)
}

// Evaluate the hand of dice and return a string representation
func toPokerString(dices []*Dice) string {
	s := ""
	for _, dice := range dices {
		s += strconv.Itoa(dice.Value)
	}
	runes := []rune(s)
	sort.Slice(runes, func(i, j int) bool {
		return runes[i] < runes[j]
	})
	s = string(runes)

	if s == "12345" {
		return "LS"
	}
	if s == "23456" {
		return "HS"
	}

	mapCount := make(map[int]int)
	for _, c := range s {
		mapCount[int(c-'0')]++
	}

	keys := []int{}
	values := []int{}
	for k, v := range mapCount {
		if v > 1 {
			keys = append(keys, k)
			values = append(values, v)
		}
	}

	if len(keys) == 0 {
		return "nothing"
	}

	sort.Slice(keys, func(i, j int) bool { return keys[i] > keys[j] })
	sort.Slice(values, func(i, j int) bool { return values[i] > values[j] })

	n := strings.Trim(strings.Replace(fmt.Sprint(keys), " ", "", -1), "[]")
	c := strings.Trim(strings.Replace(fmt.Sprint(values), " ", "", -1), "[]")

	switch c {
	case "5":
		return n + "F"
	case "4":
		return n + "q"
	case "3":
		return n + "t"
	case "32":
		return n + "fh"
	case "22":
		return n + "s"
	case "2":
		return n + "s"
	default:
		return n + ""
	}
}
