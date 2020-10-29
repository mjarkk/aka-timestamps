package main

import (
	"bytes"
	"encoding/json"
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

type QuestionType struct {
	Full       string `json:"full"`
	Searchable string `json:"searchable"`
	Shortent   string `json:"shortent"`
}

type DetectedTimeStamp struct {
	QuestionIdx int    `json:"questionIdx"`
	AtStr       string `json:"atStr"`
	Found       bool   `json:"found"`
}

func startup() (useSystemYoutubeDL, fetchOnStartup bool, err error) {
	_, err = ioutil.ReadFile("./youtube-dl")
	if err != nil {
		useSystemYoutubeDL = true
	}

	err = os.Mkdir(".vid-meta", 0777)
	if os.IsExist(err) {
		err = nil
	} else {
		fetchOnStartup = true
	}
	return
}

func downloadLatestVideosMeta(useSystemYoutubeDL bool, firstTime bool) error {
	youtubedl := "../youtube-dl"
	if useSystemYoutubeDL {
		youtubedl = "youtube-dl"
	}
	max := "5"
	if firstTime {
		max = "10"
	}

	cmd := exec.Command(
		youtubedl,
		"-4",
		"--yes-playlist",
		"--ignore-errors",
		"--output", "%(playlist_index)s.%(title)s.vid",
		"--write-auto-sub",
		"--write-description",
		"--max-downloads", max,
		"--skip-download", "https://www.youtube.com/playlist?list=UUs58xfxPpjVARRuwjH8usfw",
	)

	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	cmd.Dir = path.Join(wd, ".vid-meta")

	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	err = cmd.Run()
	if output.Len() == 0 {
		if err != nil {
			return fmt.Errorf("youtube-dl exitted with error: %v", err)
		}
		return errors.New("youtube-dl exitted without any output nor error code")
	}

	fmt.Println("youtube-dl output:")
	fmt.Println(output.String())

	if err == nil {
		return nil
	}

	if strings.Contains(err.Error(), "101") {
		return nil
	}

	return errors.New(output.String())
}

func extractSubtitles(ep *FoundEp) ([]textAtTime, map[string][]int, error) {
	elementRegx := regexp.MustCompile(`<(\/)?(\d{1,2}:\d{1,2}:\d{1,2}(\.\d+)?|c)(\/)?>`)

	subtitles, err := astisub.OpenFile(".vid-meta/" + ep.BaseName() + "en.vtt")
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

func extractComments(ep *FoundEp) ([]string, error) {
	descriptionBytes, err := ioutil.ReadFile(path.Join(".vid-meta", ep.BaseName()+"description"))
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

func detectQuestions(linesThatMightBeQuestions []string) []QuestionType {
	questions := []QuestionType{}
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

		questions = append(questions, QuestionType{
			Full:       question,
			Searchable: searchableQuestion,
			Shortent:   shoterQuestion,
		})
	}
	return questions
}

func detectTimestamps(wordsMap map[string][]int, words []textAtTime, detectedQuestions []QuestionType) []DetectedTimeStamp {
	res := []DetectedTimeStamp{}

	for questionIdx, question := range detectedQuestions {
		indexes := []int{}
		for _, word := range strings.Split(question.Searchable, " ") {
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

		if len(longestPairIdx) > 3 {
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

			res = append(res, DetectedTimeStamp{
				QuestionIdx: questionIdx,
				AtStr:       atStr,
				Found:       found,
			})
		}
	}

	return res
}

type FoundEps []FoundEp

func (s FoundEps) Len() int      { return len(s) }
func (s FoundEps) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

type FoundEpsByEpNumb struct{ FoundEps }

func (s FoundEpsByEpNumb) Less(i, j int) bool { return s.FoundEps[i].Number < s.FoundEps[j].Number }

type FoundEp struct {
	Number           int           `json:"number"`
	RawNumber        string        `json:"rawNumber"`
	Name             string        `json:"name"`
	FoundDescription bool          `json:"foundDescription"`
	FoundVTT         bool          `json:"foundVTT"`
	FoundResults     *FoundResults `json:"foundResults"`
}

func (ep *FoundEp) BaseName() string {
	return fmt.Sprintf("%s.%s.vid.", ep.RawNumber, ep.Name)
}

type FoundResults struct {
	Questions []QuestionType      `json:"questions"`
	TimeStamp []DetectedTimeStamp `json:"timeStamp"`
	Err       string              `json:"err"`
}

func checkDownloadedVideos() error {
	items, err := ioutil.ReadDir(".vid-meta")
	if err != nil {
		return err
	}

	foundEps := map[string]FoundEp{}

	for _, item := range items {
		name := item.Name()
		nameParts := strings.Split(name, ".")
		if len(nameParts) < 4 {
			// File has invalid name
			continue
		}
		epNumberStr := nameParts[0]
		epNumber, err := strconv.Atoi(epNumberStr)
		if err != nil {
			// File has invalid name
			continue
		}
		ext := ""
		for {
			part := nameParts[len(nameParts)-1]
			nameParts = nameParts[:len(nameParts)-1]
			if part == "vid" {
				break
			}

			if ext == "" {
				ext = part
			} else {
				ext = part + "." + ext
			}
		}

		epName := strings.Join(nameParts[1:], ".")
		lowerEpName := strings.ToLower(epName)
		if strings.Contains(lowerEpName, "otdm") {
			// Wrong series
			continue
		}
		if !strings.Contains(lowerEpName, "ask kati anything") && !strings.Contains(lowerEpName, "aka") {
			// Wrong series
			continue
		}

		ep, ok := foundEps[epName]
		if !ok {
			ep = FoundEp{
				RawNumber: epNumberStr,
				Number:    epNumber,
				Name:      epName,
			}
		}

		if ext == "description" {
			ep.FoundDescription = true
		} else if ext == "en.vtt" {
			ep.FoundVTT = true
		} else if ext == "anylize.results" {
			data, err := ioutil.ReadFile(path.Join(".vid-meta", name))
			var results FoundResults
			if err != nil {
				results.Err = err.Error()
			} else {
				err = json.Unmarshal(data, &results)
				if err != nil {
					results.Err = err.Error()
				}
			}
			ep.FoundResults = &results
		} else {
			// Unknown file extension go to next file
			continue
		}

		foundEps[epName] = ep
	}

	eps := FoundEps{}
	for _, value := range foundEps {
		eps = append(eps, value)
	}
	sort.Sort(FoundEpsByEpNumb{eps})

	for idx, ep := range eps {
		if ep.FoundResults != nil || !ep.FoundDescription || !ep.FoundVTT {
			continue
		}
		questions, timeStamps, err := checkVid(&ep)
		errStr := ""
		if err != nil {
			errStr = err.Error()
		}
		results := FoundResults{
			Questions: questions,
			TimeStamp: timeStamps,
			Err:       errStr,
		}
		toSafe, err := json.Marshal(results)
		if err != nil {
			results = FoundResults{Err: err.Error()}
			toSafe, err = json.Marshal(results)
			if err != nil {
				continue
			}
		}

		err = ioutil.WriteFile(".vid-meta/"+ep.BaseName()+"anylize.results", toSafe, 0777)
		if err != nil {
			continue
		}
		ep.FoundResults = &results
		eps[idx] = ep
	}

	videosLock.Lock()
	videos = eps
	videosLock.Unlock()

	return nil
}

func checkVid(ep *FoundEp) ([]QuestionType, []DetectedTimeStamp, error) {
	words, wordsMap, err := extractSubtitles(ep)
	if err != nil {
		return nil, nil, err
	}

	linesThatMightBeQuestions, err := extractComments(ep)
	if err != nil {
		return nil, nil, err
	}

	detectedQuestions := detectQuestions(linesThatMightBeQuestions)
	detectedTimeStamps := detectTimestamps(wordsMap, words, detectedQuestions)
	return detectedQuestions, detectedTimeStamps, nil
}

func main() {
	fmt.Println("Staring..")
	useSystemYoutubeDL, fetchOnStartup, err := startup()
	check(err)

	go func(useSystemYoutubeDL bool) {
		downloadingLock.Lock()
		downloading = true
		downloadingLock.Unlock()

		defer func() {
			downloadingLock.Lock()
			downloading = false
			downloadingLock.Unlock()
		}()

		if fetchOnStartup {
			fmt.Println("Downloading latest video meta data..")
			err = downloadLatestVideosMeta(useSystemYoutubeDL, true)
			if err != nil {
				fmt.Println("downloading videos error:", err)
				return
			}
		}

		fmt.Println("Check downloaded videos..")
		err = checkDownloadedVideos()
		if err != nil {
			fmt.Println("checking downloaded videos error:", err)
		} else {
			fmt.Println("Comleted search for videos")
		}
	}(useSystemYoutubeDL)

	check(serve(useSystemYoutubeDL))
}
