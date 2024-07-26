package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"xabbo.b7c.io/goearth/shockwave/out"
)

// Wait for all dice results and evaluate the poker hand
func evaluatePokerHand() {
	isPokerRolling = false
	if !ChatIsDisabled {
		hand := toPokerString(diceList)
		ext.Send(out.SHOUT, hand)
	}
}

// Wait for all dice results and evaluate the tri hand
func evaluateTriHand() {
	isTriRolling = false
	if !ChatIsDisabled {
		hand := sumHand([]int{
			diceList[0].Value,
			diceList[2].Value,
			diceList[4].Value,
		})
		ext.Send(out.SHOUT, hand)
	}
}

// Sum the values of the dice and return a string representation
func sumHand(values []int) string {
	sum := 0
	for _, val := range values {
		sum += val
	}
	return strconv.Itoa(sum)
}

// Evaluate the hand of dice and return a string representation
// thank you b7 <3 (and me, eduard, selfplug lol)
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
		return "F" + n 
	case "4":
		return "q" + n
	case "3":
		return "t" + n
	case "32":
		return "fh" + n
	case "22":
		return n + "s"
	case "2":
		return n + "s"
	default:
		return n + ""
	}
}