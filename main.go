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
	number   int
	atStr    string
	found    bool
}

func parseArgs() (youtubeURL string, showAll bool, err error) {
	for _, arg := range os.Args[1:] {
		if arg == "-a" || arg == "--all" {
			showAll = true
		} else if strings.Contains(arg, "youtube.com") || strings.Contains(arg, "youtu.be") {
			youtubeURL = arg
		}
	}
	if youtubeURL == "" {
		err = errors.New("No video url provided")
	}
	return
}

func cleaupAndPrepair() error {
	os.RemoveAll(".vid-meta")
	return os.Mkdir(".vid-meta", 0777)
}

func downloadVideoMeta(url string) error {
	cmd := exec.Command(
		"../youtube-dl",
		url,
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
			if !strings.ContainsRune("1234567890", letter) {
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
		questionLines := strings.Split(match, "\n")
		question := questionLines[0]
		if len(questionLines) >= 3 {
			question += " " + questionLines[1]
		}

		for i, letter := range question {
			if !strings.ContainsRune("1234567890.:=() ", letter) {
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

func detectTimestamps(wordsMap map[string][]int, words []textAtTime, detectedQuestions []questionType, allTimes bool) []detectedTimeStamp {
	res := []detectedTimeStamp{}

	for questionIdx, question := range detectedQuestions {
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
		longestPairIdx := []int{0}
		longestPairIdxLast := func() int {
			return longestPairIdx[len(longestPairIdx)-1]
		}

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

			if pairsIdx != longestPairIdxLast() && len(lastPair) > len(pairs[longestPairIdxLast()]) {
				longestPairIdx = append(longestPairIdx, pairsIdx)
			}

			pairs[len(pairs)-1] = lastPair
		}

		if !allTimes {
			longestPairIdx = []int{longestPairIdx[len(longestPairIdx)-1]}
		} else if len(longestPairIdx) > 3 {
			longestPairIdx = longestPairIdx[len(longestPairIdx)-3:]
		}

		for _, pairIdx := range longestPairIdx {
			atStr := ""
			found := false

			resultIndexes := pairs[pairIdx]
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
				number:   questionIdx + 1,
			})
		}
	}

	return res
}

func main() {
	fmt.Println("Parsing args..")
	youtubeURL, showAll, err := parseArgs()
	check(err)

	fmt.Println("Cleaning up and preparing..")
	err = cleaupAndPrepair()
	check(err)

	fmt.Println("Downloading video meta data..")
	err = downloadVideoMeta(youtubeURL)
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
	output := detectTimestamps(wordsMap, words, detectedQuestions, showAll)
	for _, detectedItem := range output {
		if detectedItem.found {
			fmt.Printf("%d. %s %s\n", detectedItem.number, detectedItem.atStr, detectedItem.question.shortent)
		} else {
			fmt.Printf("%d. ??:?? %s\n", detectedItem.number, detectedItem.question.shortent)
		}
	}
	fmt.Println("\nPlease correct me if i'm wrong these are auto generated :)")
}
