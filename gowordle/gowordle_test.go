package gowordle

import (
	"math/rand"
	"sort"
	"strings"
	"testing"

	"github.com/bits-and-blooms/bitset"
	"github.com/stretchr/testify/assert"
)

func TestMany(t *testing.T) {
	set := bitset.New(5)
	set.Set(0)
	set.Set(3)
	indices := make([]uint, set.Count())
	set.NextSetMany(0, indices)
}

func TestBest(t *testing.T) {
	words := StringsToWordleWords([]string{"aaaaa", "abbbb"})
	score := NextGuess1(words, words)
	assert.NotZero(t, score)
	print("testbest")
}

func WW(ins string) WordleWord {
	return WordleWord([]rune(ins))
}
func TestMatching1(t *testing.T) {
	words := StringsToWordleWords([]string{"aaaaa", "abbbb"})
	wds := NewWordleMatcher(words)
	assert := assert.New(t)
	matching := wds.Matching(WW("aazzz"), WW("ggrrr"))
	assert.Equal(matching, StringsToWordleWords([]string{"aaaaa"}))

	matching = wds.Matching(WW("bzzzz"), WW("yrrrr"))
	assert.Equal(matching, StringsToWordleWords([]string{"abbbb"}))
}

func TestMatching2(t *testing.T) {
	words := StringsToWordleWords([]string{"aaaaa", "abbbb"})
	wds := NewWordleMatcher(words)
	assert := assert.New(t)
	matching := wds.Matching(WW("bzzzz"), WW("yrrrr"))
	assert.Equal(matching, StringsToWordleWords([]string{"abbbb"}))
}

func WordSort(ws []WordleWord) []string {
	s := make([]string, len(ws))
	for i, w := range ws {
		s[i] = string(w[:])
	}
	sort.Strings(s)
	return s
}

func testMatching(t *testing.T, words []string, guess string, answer string, expected []string) {
	wwords := StringsToWordleWords(words)
	wds := NewWordleMatcher(wwords)
	matching := wds.Matching(WW(guess), WW(answer))
	assert := assert.New(t)
	sort.Strings(expected)
	matching_s := WordSort(matching)
	assert.Equal(expected, matching_s)
}

func TestMatching3(t *testing.T) {
	testMatching(t,
		[]string{"aaazz", "abbbb", "bcazz"},
		"bxxac", "yrryr", // answer abbbb
		[]string{"abbbb"},
	)
}

func TestMatching4(t *testing.T) {
	testMatching(t,
		[]string{"aaazz", "abbzz", "abczz", "abazz", "bbazz"},
		"xabxx", "ryyrr", // answer abazz
		[]string{"abczz", "abazz", "bbazz"},
	)
}

func TestGreenYellow(t *testing.T) {
	testMatching(t,
		[]string{"aaazz", "abbzz", "abczz", "abazz", "bbazz", "azzza", "azzzz"},
		"axxxa", "grrry", // answer abazz
		[]string{"aaazz", "abazz"},
	)
}

func TestYellowRed(t *testing.T) {
	testMatching(t,
		[]string{"aaazz", "abbzz", "abczz", "abazz", "bbazz", "azzza", "azzzz", "aazzz", "aaazz"},
		"axxaa", "grryr", // answer abazz, two a's, but not 3
		[]string{"abazz", "aazzz"},
	)
}

func TestW1(t *testing.T) {
	testMatching(t,
		WordleDictionary,
		"aaxxd", "yyrry",
		[]string{"drama"},
	)
}

func TestFirst20(t *testing.T) {
	wordList := []string{"cigar", "rebut", "sissy", "humph", "awake", "blush", "focal", "evade", "naval", "serve", "heath", "dwarf", "model", "karma", "stink", "grade", "quiet", "bench", "abate", "feign"}
	simulateWords := Simulate(wordList, "karma", "cigar")
	assert := assert.New(t)
	assert.Greater(float32(5.0), float32(len(simulateWords)))
}

func TestSimple(t *testing.T) {
	possibleWords := StringsToWordleWords([]string{"clack", "clamp", "clank", "cloak", "local", "octal", "vocal"})
	wordList := append(StringsToWordleWords([]string{"thank"}), possibleWords...)
	ScoreAlgorithmTotalMatches1LevelAll(wordList, possibleWords, possibleWords, 1, 1000)
}
func TestFirst(t *testing.T) {
	wordList := SortedWordleDictionary()[0:100]
	// wordList := wordleDictionary
	FirstGuess1(wordList)
}

func TestFirst1(t *testing.T) {
	//wordList := WordleDictionary[0:800]
	wordList := WordleDictionary[0:200]
	score, words := FirstGuess1(wordList)
	print(score)
	PrintWords(words)
}

func TestFirstWithInitialGuesses(t *testing.T) {
	// wordList := WordleDictionary[0:800]
	wordList := WordleDictionary[0:100]
	score, words := FirstGuessProvideInitialGuesses1(wordList, wordList)
	println(score)
	PrintWords(words)
	println("hits:", HitCount)
	println("miss:", MissCount)
}

/*
func BenchmarkFirstN(t *testing.B) {
	wordList := WordleDictionary[0:400]
	FirstGuess1(wordList)
}
*/

func TestAgainstHeron400(t *testing.T) {
	// tested and got cigar/4 for 0:400
	wordList := WordleDictionary[0:400]
	// bug: cigar, rebut, serve, ferry, heron
	guess := "cigar"
	worst := 4
	solution := "heron"
	simulateWords := Simulate(wordList, solution, guess)
	if len(simulateWords) > worst {
		println(len(simulateWords), string(solution))
		worst = len(simulateWords)
	}
}
func TestAgainstHeronArise(t *testing.T) {
	// wordList := WordleDictionary[0:]
	wordList := WordleDictionary[0:200]
	guess := "arise"
	worst := 4
	solution := "heron"
	simulateWords := Simulate(wordList, solution, guess)
	if len(simulateWords) > worst {
		println(len(simulateWords), string(solution))
		worst = len(simulateWords)
	}
}
func TestAgainstServeArise(t *testing.T) {
	wordList := WordleDictionary[0:600] // contains arise
	guess := "arise"
	worst := 4
	solution := "serve"
	assertStringInSlice(t, guess, wordList)
	assertStringInSlice(t, solution, wordList)
	simulateWords := Simulate(wordList, solution, guess)
	if len(simulateWords) > worst {
		println(len(simulateWords), string(solution))
		worst = len(simulateWords)
	}
}

// Real game against online version of Wordl
func TestRealAriseToPetal(t *testing.T) {
	wordList := WordleDictionary[0:]
	guess := "arise"
	solution := "petal"
	assertStringInSlice(t, guess, wordList)
	assertStringInSlice(t, solution, wordList)
	simulateWords := Simulate(wordList, solution, guess)
	assert.Equal(t, 3, len(simulateWords))
}
func TestAgainstBrakeAtone(t *testing.T) {
	wordList := WordleDictionary[0:1200]
	guess := "atone"
	worst := 4
	solution := "break"
	assertStringInSlice(t, guess, wordList)
	assertStringInSlice(t, solution, wordList)
	simulateWords := Simulate(wordList, solution, guess)
	if len(simulateWords) > worst {
		println(len(simulateWords), string(solution))
		worst = len(simulateWords)
	}
}
func assertStringInSlice(t *testing.T, tst string, wordList []string) {
	for _, word := range wordList {
		if word == tst {
			return
		}
	}
	t.Error("word not in list: " + tst)
}

// 1200 guess=atone - 5 brake, craze, frame, grave, crave, grape, brave, graze
func TestAriseAgainstAll(t *testing.T) {
	//func BenchmarkSimulate(t *testing.B) {
	// tested and got cigar/4 for 0:200
	// tested and got cigar/4 for 0:400
	wordList := WordleDictionary[0:200]
	// wordList := WordleDictionary[0:1200] // contains arise
	// wordList := WordleDictionary[0:200]
	// bug: cigar, rebut, serve, ferry, heron
	guess := "atone"
	// not a test: assertStringInSlice(t, guess, wordList)
	worst := 0
	result := []string{}
	for i, solution := range wordList {
		simulateWords := Simulate(wordList, solution, guess)
		// not a test: assert.Equal(t, solution, string(simulateWords[len(simulateWords)-1]))
		if i%10 == 0 {
			println(i)
		}
		if len(simulateWords) == worst {
			result = append(result, string(solution))
		} else if len(simulateWords) > worst {
			println(len(simulateWords), string(solution))
			worst = len(simulateWords)
			result = []string{string(solution)}
		}
	}
	println(worst, strings.Join(result, ", "))
	PrintBetterGuesses()

}

type StringIntSlice []StringInt
type StringInt struct {
	s string
	i int
}

func (sis *StringIntSlice) Len() int {
	return len(*sis)
}

func (sis *StringIntSlice) Less(i, j int) bool {
	return (*sis)[i].i < (*sis)[j].i
}

func (sis *StringIntSlice) Swap(i, j int) {
	(*sis)[i], (*sis)[j] = (*sis)[j], (*sis)[i]
}

func PrintBetterGuesses() {
	var bgs StringIntSlice = make([]StringInt, len(BetterGuesses))
	for guess, times := range BetterGuesses {
		bgs = append(bgs, StringInt{guess, times})
	}
	sort.Sort(&bgs)
	for _, bg := range bgs {
		println(bg.s, " ", bg.i)
	}
}

func TestPlayArise(t *testing.T) {
	wordList := StringsToWordleWords(WordleDictionary[0:])
	guessAnswers := []GuessAnswer{
		{WordleWord([]rune("arise")), WordleWord([]rune("yrrry"))},
		{WordleWord([]rune("metal")), WordleWord([]rune("rgggg"))},
	}
	ww := PlayWordle(wordList, guessAnswers)
	println(string(ww[:]))
	print("done")
}
func TestDifficult(t *testing.T) {
	// ./wdl play raise rryrr hotly yryrr
	// twang: night might ninth wight width fight tight fifth
	// ./wdl play raise rryrr hotly yryrr twang yrrry
	// might: might fight
}

/*
func BenchmarkProof(t *testing.B) {
	wordList := wordleDictionary[0:400]
	_, ret := FirstGuess1(wordList)
	assert := assert.New(t)
	for _, solution := range wordList {
		simulateWords := Simulate(wordList, solution, ret)
		assert.Greater(float32(7.0), float32(len(simulateWords)))
	}
}

*/

var topGuesses = []string{
	"atone", // 20
	"raise", // 0
	"arise", // 1
	"irate", // 2
	"arose", // 3
	"alter", // 4
	"saner", // 5
	"later", // 6
	"snare", // 7
	"stare", // 8
	"slate", // 9
	"alert", // 10
	"crate", // 11
	"trace", // 12
	"stale", // 13
	"aisle", // 14
	"learn", // 15
	"leant", // 16
	"alone", // 17
	"least", // 18
	"crane", // 19
	"trail", // 21
	"react", // 22
	"trade", // 23
}

func BenchmarkFirst1(t *testing.B) {
	BestGuess1 = ScoreAlgorithmRecursive
	// wordList := SortedWordleDictionary()[0:800]
	wordList := SortedWordleDictionary()[:]
	var topGuesses = []string{
		"atone", // 20
		"raise", // 0
		"arise", // 1
	}
	FirstGuessProvideInitialGuesses1(topGuesses, wordList)
	// println(score)
	// PrintWords(words)
	// println("hits:", HitCount)
	// println("miss:", MissCount)
}

var HitmissMap map[string]*Answer = make(map[string]*Answer, 10000)

func testMap(solution, guess WordleWord) *Answer {
	key := string(solution[:]) + string(guess[:])
	if ret, ok := HitmissMap[key]; ok {
		HitCount++
		return ret
	}
	ret := Answer{}
	MissCount++
	HitmissMap[key] = &ret
	return &ret
}

var HitmissInt map[int]*Answer = make(map[int]*Answer, 10000)

func testMapInt(solution, guess int) *Answer {
	//key := string(solution) + string(guess)
	key := solution*10_000 + guess
	if ret, ok := HitmissInt[key]; ok {
		HitCount++
		return ret
	}
	ret := Answer{}
	MissCount++
	HitmissInt[key] = &ret
	return &ret
}

var HitmissIndex []*Answer = make([]*Answer, (Choices*Choices)+Choices)

func testMapIndex(solution, guess int) *Answer {
	//key := string(solution) + string(guess)
	key := solution*Choices + guess
	if ret := HitmissIndex[key]; ret != nil {
		HitCount++
		return ret
	}
	ret := Answer{}
	MissCount++
	HitmissIndex[key] = &ret
	return &ret
}

const L = 1_000_000_000
const Choices = 250

func BenchmarkMapStrings(t *testing.B) {
	wordList := WordleDictionary
	allWords := StringsToWordleWords(wordList)
	l := L
	choices := Choices
	for i := 0; i < l; i++ {
		testMap(allWords[rand.Intn(choices)], allWords[rand.Intn(choices)])
	}
	println("hits:", HitCount)
	println("miss:", MissCount)
}

func BenchmarkMapInt(t *testing.B) {
	l := L
	choices := Choices
	for i := 0; i < l; i++ {
		testMapInt(rand.Intn(choices), rand.Intn(choices))
	}
	println("hits:", HitCount)
	println("miss:", MissCount)
}

func BenchmarkMapIndex(t *testing.B) {
	l := L
	choices := Choices
	for i := 0; i < l; i++ {
		testMapIndex(rand.Intn(choices), rand.Intn(choices))
	}
	println("hits:", HitCount)
	println("miss:", MissCount)
}

func BenchmarkSimulate(t *testing.B) {
	BestGuess1 = ScoreAlgorithmRecursive
	// wordList := SortedWordleDictionary()[0:800]
	wordList := SortedWordleDictionary()
	for _, answer := range wordList[0:3] {
		guesses := Simulate(wordList, answer, "raise")
		println(answer, guesses)
	}
}
