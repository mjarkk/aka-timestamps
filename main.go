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
	"strconv"
	"strings"
	"time"

	"github.com/asticode/go-astisub"
)

type textAtTime struct {
	text string
	at   time.Duration
}

type questionType struct {
	full       string
	searchable string
	shortent   string
}

type detectedTimeStamp struct {
	question questionType
	atStr    string
	found    bool
}

func cleaupAndPrepair() error {
	os.RemoveAll(".vid-meta")
	return os.Mkdir(".vid-meta", 0777)
}

func downloadVideoMeta() error {
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
	if err != nil {
		return err
	}
	cmd.Dir = path.Join(wd, ".vid-meta")

	stdOut, err := cmd.CombinedOutput()
	if err != nil && stdOut != nil && len(stdOut) > 0 {
		err = errors.New(string(stdOut))
	}
	return err
}

func extractSubtitles() ([]textAtTime, map[string][]int, error) {
	elementRegx := regexp.MustCompile(`<(\/)?(\d{1,2}:\d{1,2}:\d{1,2}(\.\d+)?|c)(\/)?>`)

	subtitles, err := astisub.OpenFile(".vid-meta/vid.en.vtt")
	if err != nil {
		return nil, nil, err
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

	return words, wordsMap, nil
}

func extractComments() ([]string, error) {
	descriptionBytes, err := ioutil.ReadFile(".vid-meta/vid.description")
	if err != nil {
		return nil, err
	}

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

	matched := []string{}
	for _, part := range descriptionParts {
		part = strings.TrimSpace(part)
		if len(part) < 10 {
			continue
		}

		if strings.ContainsRune("123456789", rune(part[0])) {
			matched = append(matched, part)
			continue
		}
		parts := strings.SplitN(part, "\n", 2)
		if len(parts) >= 2 && strings.ContainsRune("123456789", rune(parts[1][0])) {
			matched = append(matched, parts[1])
			continue
		}
	}

	filteredList := []string{}
	minNumber := 0
	for _, item := range matched {
		numberStr := []byte{}
		for _, letter := range item {
			if !strings.ContainsRune("123456789", letter) {
				break
			}
			numberStr = append(numberStr, byte(letter))
		}
		if len(numberStr) == 0 {
			continue
		}
		num, err := strconv.Atoi(string(numberStr))
		if err != nil {
			continue
		}
		if num < minNumber-3 || num > minNumber+3 {
			continue
		}

		filteredList = append(filteredList, item)
		minNumber = num
	}

	return filteredList, nil
}

func detectQuestions(linesThatMightBeQuestions []string) []questionType {
	questions := []questionType{}
	for _, match := range linesThatMightBeQuestions {
		question := strings.Split(match, "\n")[0]
		for i, letter := range question {
			if !strings.ContainsRune("1234567890.:= ", letter) {
				question = question[i:]
				break
			}
		}

		maxLen := 120
		shoterQuestion := question
		if len(shoterQuestion) > maxLen {
			shoterQuestion = shoterQuestion[:maxLen-2] + ".."
		}

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
	return questions
}

func detectTimestamps(wordsMap map[string][]int, words []textAtTime, detectedQuestions []questionType) []detectedTimeStamp {
	res := []detectedTimeStamp{}
	for _, question := range detectedQuestions {
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
			} else if lastPair[len(lastPair)-1]+6 > index {
				// Sometimes tweaking this max search offset gives a bit better results
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
		atStr := ""
		found := false
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

			found = true
		}

		res = append(res, detectedTimeStamp{
			question: question,
			atStr:    atStr,
			found:    found,
		})
	}
	return res
}

func main() {
	fmt.Println("Cleaningup and preparing..")
	err := cleaupAndPrepair()
	check(err)

	fmt.Println("Downloading video meta data..")
	err = downloadVideoMeta()
	check(err)

	fmt.Println("Extracting subtitles..")
	words, wordsMap, err := extractSubtitles()
	check(err)

	fmt.Println("Extracting comments..")
	linesThatMightBeQuestions, err := extractComments()
	check(err)

	fmt.Println("Detected questions:")
	detectedQuestions := detectQuestions(linesThatMightBeQuestions)
	for i, question := range detectedQuestions {
		fmt.Printf("%d. %s\n", i+1, question.shortent)
	}

	fmt.Print("\n\nTime stamps:\n\n")
	output := detectTimestamps(wordsMap, words, detectedQuestions)
	for i, detectedItem := range output {
		fmt.Printf("%d. %s %s\n", i+1, detectedItem.atStr, detectedItem.question.shortent)
	}
	fmt.Println("\nPlease correct me if i'm wrong these are auto generated :)")
}
