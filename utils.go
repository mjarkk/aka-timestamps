package main

import (
	"fmt"
	"os"
	"strings"
)

func check(err error) {
	if err != nil {
		fmt.Println("Err:", err)
		os.Exit(1)
	}
}

func toSearchable(text string) string {
	text = strings.TrimSpace(text)
	text = strings.ToLower(text)

	newText := ""
	for _, letter := range text {
		repalceWith, found := map[rune]rune{
			'\'': 0,
			'"':  0,
			'“':  0,
			'”':  0,
			')':  0,
			'(':  0,
			'.':  0,
			';':  0,
			':':  0,
			',':  0,
			'?':  0,
			'!':  0,
			'@':  0,
			'#':  0,
			'$':  0,
			'%':  0,
			'^':  0,
			'&':  0,
			'*':  0,
			'_':  0,
			'+':  0,
			']':  0,
			'[':  0,
			'}':  0,
			'{':  0,
			'/':  ' ',
			'\t': ' ',
			'\n': ' ',
		}[letter]
		if !found {
			newText += string(letter)
			continue
		}

		if repalceWith != 0 {
			newText += string(repalceWith)
		}
	}

	words := strings.Split(newText, " ")
	newWords := []string{}
	for _, word := range words {
		_, match := map[string]uint8{
			"i":       0,
			"a":       0,
			"was":     0,
			"and":     0,
			"it":      0,
			"of":      0,
			"like":    0,
			"do":      0,
			"to":      0,
			"you":     0,
			"as":      0,
			"have":    0,
			"when":    0,
			"the":     0,
			"because": 0,
			"in":      0,
			"is":      0,
			"that":    0,
		}[word]
		if match {
			continue
		}

		if len(newWords) > 0 && newWords[len(newWords)-1] == word {
			continue
		}
		newWords = append(newWords, word)
	}

	return strings.Join(newWords, " ")
}
