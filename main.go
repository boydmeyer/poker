package main

import (
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
	Version:     "1.1",
})

// Dice struct represents a dice with its ID, value, and packets for throwing and turning off
type Dice struct {
	DiceID    int
	Value     int
	ThrowDice *g.Packet
	DiceOff   *g.Packet
}

// Variables to manage dice, rolling state, mutex for setup, and wait group for results
var (
	diceArray        []*Dice
	rolling          bool
	closing		     bool
	setupMutex       sync.Mutex
	resultsWaitGroup sync.WaitGroup
)

// Entry point of the application
func main() {
	ext.Initialized(onInitialized)
	ext.Connected(onConnected)
	ext.Disconnected(onDisconnected)
	ext.Intercept(out.CHAT, out.SHOUT, out.WHISPER).With(handleChat)
	ext.Intercept(out.THROW_DICE).With(handleThrowDice)
	ext.Intercept(out.DICE_OFF).With(handleDiceOff)
	ext.Intercept(in.DICE_VALUE).With(handleDiceResult)
	ext.Run()
}

func onInitialized(e g.InitArgs) {
	log.Println("Extension initialized")
}

func onConnected(e g.ConnectArgs) {
	log.Printf("Game connected (%s)\n", e.Host)
}

func onDisconnected() {
	log.Println("Game disconnected")
}

// Handle chat messages to trigger dice actions
func handleChat(e *g.Intercept) {
	msg := e.Packet.ReadString()
	if strings.Contains(msg, ":close") {
		e.Block()
		log.Println(msg)

		if rolling  {
			log.Println("Already rolling...")
			return
		}

		if closing  {
			log.Println("Already closing...")
			return
		}

		go closeDice()
	} else if strings.Contains(msg, ":reset") {
		e.Block()
		log.Println(msg)
		resetPackets()
	} else if strings.Contains(msg, ":roll") {
		e.Block()
		log.Println(msg)

		if rolling {
			log.Println("Already rolling...")
			return
		}

		if closing  {
			log.Println("Already closing...")
			return
		}

		rolling = true
		go rollDice()
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
		if d != nil && d.DiceID == diceID {
			dice = d
			if dice.ThrowDice == nil {
				dice.ThrowDice = packet.Copy()
			}
			break
		}
	}

	if dice == nil && len(diceArray) < 5 {
		dice = &Dice{
			DiceID:    diceID,
			ThrowDice: packet.Copy(),
		}
		diceArray = append(diceArray, dice)
		log.Printf("Dice %d added\n", dice.DiceID)
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
		if d.DiceID == diceID {
			if d.DiceOff == nil {
				d.DiceOff = packet.Copy()
			}
			log.Printf("Dice %d off added\n", diceID)
			dice = d
			break
		}
	}

	if dice == nil && len(diceArray) < 5 {
		dice = &Dice{
			DiceID:  diceID,
			DiceOff: packet.Copy(),
		}
		diceArray = append(diceArray, dice)
		log.Printf("Dice %d added\n", dice.DiceID)
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
		if dice.DiceID == diceID {
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
	setupMutex.Unlock()
	closing = true
	for _, dice := range diceArray {
		if dice.DiceOff != nil {
			ext.SendPacket(dice.DiceOff)
			time.Sleep(500 * time.Millisecond)
		}
	}
	time.Sleep(1500 * time.Millisecond)
	closing = false
}

// Roll the dice by sending packets and waiting for results
func rollDice() {
	setupMutex.Lock()

	if len(diceArray) < 5 {
		setupMutex.Unlock()
		log.Println("Not enough dice to roll")
		return
	}

	validDiceCount := 0
	for _, dice := range diceArray {
		if dice.ThrowDice != nil {
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
		if dice.ThrowDice != nil {
			ext.SendPacket(dice.ThrowDice)
			time.Sleep(500 * time.Millisecond)
		}
	}

	time.Sleep(1000 * time.Millisecond)
	resultsWaitGroup.Wait()
	waitForAllResults()
}

// Wait for all dice results and evaluate the hand
func waitForAllResults() {
	rolling = false
	hand := evaluateHand([]int{diceArray[0].Value, diceArray[1].Value, diceArray[2].Value, diceArray[3].Value, diceArray[4].Value})
	log.Printf("Hand: %s\n", hand)
	ext.Send(out.SHOUT, hand)
}

// Evaluate the hand of dice and return a string representation
func evaluateHand(dice []int) string {
	if len(dice) != 5 {
		return "Invalid input"
	}

	sort.Ints(dice)
	counts := make(map[int]int)

	for _, die := range dice {
		counts[die]++
	}

	if fullHouseValues, isFullHouse := getFullHouse(counts); isFullHouse {
		return "Full House: " + strconv.Itoa(fullHouseValues[0]) + " over " + strconv.Itoa(fullHouseValues[1])
	} else if fiveOfAKindValue, isFiveOfAKind := getFiveOfAKind(counts); isFiveOfAKind {
		return "5 of a Kind: " + strconv.Itoa(fiveOfAKindValue)
	} else if fourOfAKindValue, isFourOfAKind := getFourOfAKind(counts); isFourOfAKind {
		return "4 of a Kind: " + strconv.Itoa(fourOfAKindValue)
	} else if isStraight(dice) {
		return "Straight: " + getStraightString(dice)
	} else if threeOfAKindValue, isThreeOfAKind := getThreeOfAKind(counts); isThreeOfAKind {
		return "3 of a Kind: " + strconv.Itoa(threeOfAKindValue)
	} else if pairValues, isTwoPair := getTwoPair(counts); isTwoPair {
		return "2 Pair: " + strconv.Itoa(pairValues[0]) + " and " + strconv.Itoa(pairValues[1])
	} else if pairValue, isOnePair := getOnePair(counts); isOnePair {
		return "1 Pair: " + strconv.Itoa(pairValue)
	} else {
		return "No Pair"
	}
}

// Get the value if there are five of a kind
func getFiveOfAKind(counts map[int]int) (int, bool) {
	for value, count := range counts {
		if count == 5 {
			return value, true
		}
	}
	return 0, false
}

// Get the value if there are four of a kind
func getFourOfAKind(counts map[int]int) (int, bool) {
	for value, count := range counts {
		if count == 4 {
			return value, true
		}
	}
	return 0, false
}

// Get the values for a full house
func getFullHouse(counts map[int]int) ([2]int, bool) {
	var threeValue, twoValue int
	for value, count := range counts {
		if count == 3 {
			threeValue = value
		} else if count == 2 {
			twoValue = value
		}
	}
	if threeValue != 0 && twoValue != 0 {
		return [2]int{threeValue, twoValue}, true
	}
	return [2]int{}, false
}

// Check if the dice form a straight
func isStraight(dice []int) bool {
	if (dice[0] == 1 && dice[1] == 2 && dice[2] == 3 && dice[3] == 4 && dice[4] == 5) ||
		(dice[0] == 2 && dice[1] == 3 && dice[2] == 4 && dice[3] == 5 && dice[4] == 6) {
		return true
	}
	for i := 0; i < 4; i++ {
		if dice[i+1] != dice[i]+1 {
			return false
		}
	}
	return true
}

// Get a string representation of the straight
func getStraightString(dice []int) string {
	strValues := []string{}
	for _, value := range dice {
		strValues = append(strValues, strconv.Itoa(value))
	}
	return strings.Join(strValues, ", ")
}

// Get the value if there are three of a kind
func getThreeOfAKind(counts map[int]int) (int, bool) {
	for value, count := range counts {
		if count == 3 {
			return value, true
		}
	}
	return 0, false
}

// Get the values if there are two pairs
func getTwoPair(counts map[int]int) ([]int, bool) {
	pairs := []int{}
	for value, count := range counts {
		if count == 2 {
			pairs = append(pairs, value)
		}
	}
	if len(pairs) == 2 {
		return pairs, true
	}
	return nil, false
}

// Get the value if there is one pair
func getOnePair(counts map[int]int) (int, bool) {
	for value, count := range counts {
		if count == 2 {
			return value, true
		}
	}
	return 0, false
}
