package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/asticode/go-astisub"
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

func main() {
	fmt.Println("Cleaningup and preparing..")
	err := os.RemoveAll(".vid-meta")
	check(err)
	err = os.Mkdir(".vid-meta", 0777)
	check(err)

	fmt.Println("Downloading video meta data..")
	cmd := exec.Command(
		"../youtube-dl",
		os.Args[len(os.Args)-1],
		"--write-auto-sub",
		"--write-description",
		"--output",
		"vid",
		"--skip-download",
	)
	wd, err := os.Getwd()
	check(err)
	cmd.Dir = path.Join(wd, ".vid-meta")

	stdOut, err := cmd.CombinedOutput()
	if err != nil {
		if stdOut != nil && len(stdOut) > 0 {
			err = errors.New(string(stdOut))
		}
		check(err)
	}

	fmt.Println("Extracting subtitles..")
	elementRegx := regexp.MustCompile(`<(\/)?(\d{1,2}:\d{1,2}:\d{1,2}(\.\d+)?|c)(\/)?>`)

	subtitles, err := astisub.OpenFile(".vid-meta/vid.en.vtt")
	check(err)

	type textAtTime struct {
		text string
		at   time.Duration
	}

	lines := []textAtTime{}
	for _, item := range subtitles.Items {
		for _, line := range item.Lines {
			text := line.String()
			if text == "" {
				continue
			}

			text = elementRegx.ReplaceAllString(text, "")
			text = toSearchable(text)

			if len(lines) > 1 && lines[len(lines)-1].text == text {
				continue
			}

			lines = append(lines, textAtTime{
				text: text,
				at:   item.StartAt,
			})
		}
	}

	words := []textAtTime{}
	for _, line := range lines {
		for _, word := range strings.Split(line.text, " ") {
			words = append(words, textAtTime{
				text: word,
				at:   line.at,
			})
		}
	}

	wordsMap := map[string][]int{}
	for i, word := range words {
		indexes, found := wordsMap[word.text]
		if !found {
			wordsMap[word.text] = []int{i}
			continue
		}
		indexes = append(indexes, i)
		wordsMap[word.text] = indexes
	}

	fmt.Println("Extracting comments..")
	descriptionBytes, err := ioutil.ReadFile(".vid-meta/vid.description")
	check(err)
	description := string(descriptionBytes)
	for i := 0; i < 10; i++ {
		toReplace := ""
		for j := 0; j < i+1; j++ {
			toReplace += " "
		}
		description = strings.ReplaceAll(description, "\n"+toReplace+"\n", "\n\n")
	}
	for i := 0; i < 4; i++ {
		description = strings.ReplaceAll(description, "\n\n\n", "\n\n")
	}

	descriptionParts := strings.Split(description, "\n\n")

	var matched []string = nil
	for _, part := range descriptionParts {
		part = strings.TrimSpace(part)
		if len(part) < 10 {
			continue
		}

		if matched == nil {
			if part[0] == '1' {
				matched = []string{part}
				continue
			}
			parts := strings.SplitN(part, "\n", 2)
			if len(parts) >= 2 && parts[1][0] == '1' {
				matched = []string{parts[1]}
				continue
			}
			continue
		}

		if strings.Contains("123456789", string(part[0])) {
			matched = append(matched, part)
		} else {
			break
		}
	}

	fmt.Println("Detected questions:")
	type questionType struct {
		full       string
		searchable string
		shortent   string
	}
	questions := []questionType{}
	for i, match := range matched {
		question := strings.Split(match, "\n")[0]
		for i, letter := range question {
			if !strings.ContainsRune("1234567890.:= ", letter) {
				question = question[i:]
				break
			}
		}

		maxLen := 140
		shoterQuestion := question
		if len(shoterQuestion) > maxLen {
			shoterQuestion = shoterQuestion[:maxLen-2] + ".."
		}
		fmt.Printf("%d. %s\n", i+1, shoterQuestion)

		maxLen = 170
		searchableQuestion := question
		removeLastWord := false
		if len(searchableQuestion) > maxLen {
			searchableQuestion = searchableQuestion[:maxLen]
			removeLastWord = true
		}

		// remove the text in between of qoutes: (these kinds of things)
		for {
			parts := strings.SplitN(searchableQuestion, "(", 2)
			searchableQuestion = parts[0]
			if len(parts) != 2 {
				break
			}

			parts = strings.SplitN(parts[1], ")", 2)
			if len(parts) != 2 {
				break
			}

			searchableQuestion += " " + parts[1]
		}

		searchableQuestion = toSearchable(searchableQuestion)
		if removeLastWord {
			words := strings.Split(searchableQuestion, " ")
			if len(words) >= 2 {
				searchableQuestion = strings.Join(words[:len(words)-1], " ")
			}
		}

		questions = append(questions, questionType{
			full:       question,
			searchable: searchableQuestion,
			shortent:   shoterQuestion,
		})
	}
	matched = nil

	fmt.Print("\n\nTime stamps:\n\n")

	for questionIdx, question := range questions {
		indexes := []int{}
		for _, word := range strings.Split(question.searchable, " ") {
			match, ok := wordsMap[word]
			if ok {
				for _, item := range match {
					indexes = append(indexes, item)
				}
			}
		}

		sort.Ints(indexes)

		pairs := [][]int{{}}
		longestPairIdx := 0
		for i, index := range indexes {
			pairsIdx := len(pairs) - 1
			lastPair := pairs[pairsIdx]
			if i == 0 {
				lastPair = []int{index}
			} else if lastPair[len(lastPair)-1]+15 > index {
				lastPair = append(lastPair, index)
			} else {
				pairs = append(pairs, []int{index})
				continue
			}

			if pairsIdx != longestPairIdx && len(lastPair) > len(pairs[longestPairIdx]) {
				longestPairIdx = pairsIdx
			}

			pairs[len(pairs)-1] = lastPair
		}

		resultIndexes := pairs[longestPairIdx]
		atStr := "QUESTION NOT FOUND"
		if len(resultIndexes) > 5 {
			at := words[resultIndexes[0]].at - (time.Second * 3)

			hours := int(at.Hours())
			minutes := int(at.Minutes()) % 60
			seconds := int(at.Seconds()) % 60

			minutesStr := fmt.Sprintf("%d", minutes)
			if len(minutesStr) == 1 {
				minutesStr = "0" + minutesStr
			}

			secondsStr := fmt.Sprintf("%d", seconds)
			if len(secondsStr) == 1 {
				secondsStr = "0" + secondsStr
			}

			atStr = fmt.Sprintf("%s:%s", minutesStr, secondsStr)
			if hours > 0 {
				atStr = fmt.Sprintf("%d:%s", hours, atStr)
			}
		}

		fmt.Printf("%d. %s %s\n", questionIdx+1, atStr, question.shortent)
	}
}
