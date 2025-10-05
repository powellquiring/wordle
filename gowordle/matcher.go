package gowordle

import (
	"github.com/bits-and-blooms/bitset"
)

/*
-----------------
Section used for testing/verification, remove after testing
-----------------
*/
func VerifyWordsAreSorted(words []WordleWord) {
	for i := 1; i < len(words); i++ {
		if string(words[i-1][:]) >= string(words[i][:]) {
			panic("words not sorted")
		}
	}
}

/*
letters['a'][0] all words whose first letter is an a, [1] second letter is an a, ...

a word is represented by it's index into words
*/
type WordleWord [5]rune
type WordleMatcher struct {
	words   []WordleWord
	letters [5]map[rune]*bitset.BitSet // letters[0]['a'] set of words with first letter 'a'
	count   map[rune][]*bitset.BitSet  // count['a'][0] set of words with 1 or more a, count['b'][1] words with 2 or more b
	id      int
}

var cachedWordleMatchers map[string]*WordleMatcher = make(map[string]*WordleMatcher)
var wordleMatcherID int = 0

func findWordleMatcher1(words []WordleWord) (*WordleMatcher, bool) {
	keyRune := make([]rune, len(words)*5)
	for _, word := range words {
		keyRune = append(keyRune, word[:]...)
	}
	key := string(keyRune)
	if ret, ok := cachedWordleMatchers[key]; ok {
		return ret, ok
	}
	ret := &WordleMatcher{}
	wordleMatcherID++
	ret.id = wordleMatcherID
	ret.words = words
	cachedWordleMatchers[key] = ret
	return ret, false
}

type WordleWordStruct struct {
	word WordleWord
}
type WordleMatcherAtDepth struct {
	matcher *WordleMatcher
	deeper  map[WordleWordStruct]*WordleMatcherAtDepth
}

var depthMatchers *WordleMatcherAtDepth

var depthMatcherHitCount int

func init() {
	depthMatchers = &WordleMatcherAtDepth{
		deeper: make(map[WordleWordStruct]*WordleMatcherAtDepth),
	}
}

func findWordleMatcher(words []WordleWord) (*WordleMatcher, bool) {
	depth := depthMatchers
	for _, word := range words {
		if deeper, ok := depth.deeper[WordleWordStruct{word}]; !ok {
			// map does not contain the word, so create and add
			nextDeeper := &WordleMatcherAtDepth{
				deeper: make(map[WordleWordStruct]*WordleMatcherAtDepth),
			}
			depth.deeper[WordleWordStruct{word}] = nextDeeper
			depth = nextDeeper
		} else {
			depth = deeper
		}
	}
	if depth.matcher == nil {
		// store the new matcher
		wordleMatcherID++
		depth.matcher = &WordleMatcher{}
		depth.matcher.id = wordleMatcherID
		depth.matcher.words = words
		return depth.matcher, false
	} else {
		depthMatcherHitCount++
		return depth.matcher, true
	}
}

// take a slice of strings and make wordle words
func NewWordleMatcher(words []WordleWord) *WordleMatcher {
	// VerifyWordsAreSorted(words)
	ret, ok := findWordleMatcher(words)
	if ok {
		return ret
	}
	ret.count = make(map[rune][]*bitset.BitSet, 26)
	for w, word := range words {
		word_letters := make(map[rune]int, 5)
		for l, letter := range word {
			// letters
			if ret.letters[l] == nil {
				ret.letters[l] = make(map[rune]*bitset.BitSet)
			}
			if _, ok := ret.letters[l][letter]; !ok {
				ret.letters[l][letter] = bitset.New(uint(len(words)))
			}
			ret.letters[l][letter].Set(uint(w))
			word_letters[letter] = word_letters[letter] + 1
		}
		// count
		for letter, count := range word_letters {
			for c := 0; c < count; c++ {
				if ret.count[letter] == nil {
					ret.count[letter] = make([]*bitset.BitSet, 1) // [0]
					ret.count[letter][0] = bitset.New(uint(len(words)))
				} else if len(ret.count[letter]) <= count {
					ret.count[letter] = append(ret.count[letter], bitset.New(uint(len(words))))
				}
				ret.count[letter][c].Set(uint(w))
			}
		}
	}
	return ret
}

type LetterCount struct {
	letter rune
	count  int
}

type LetterMatch struct {
	must     map[rune]int // only consider words with this many (or more) of the letter, 0 means 1 or more
	must_not map[rune]int // eliminate all words with this many (or more) of the letter, 0 means 1 or more
}

func MakeLetterMatch(guess, answer WordleWord) LetterMatch {
	ret := LetterMatch{}
	yellow_green := make(map[rune]int, 5)
	ret.must_not = make(map[rune]int, 5)
	ret.must = make(map[rune]int, 5)
	for index, letter := range guess {
		if answer[index] == 'g' {
			yellow_green[letter] = yellow_green[letter] + 1
		} else if answer[index] == 'y' {
			yellow_green[letter] = yellow_green[letter] + 1
			ret.must[letter] = ret.must[letter] + 1
		} else { // r
			ret.must_not[letter] = 0
		}
	}
	// The number of red letters not found in the word depends on how many green/yellow
	// aaabb/ryggg means that that all words with 3 or more a's can be eliminated
	for red, _ := range ret.must_not {
		ret.must_not[red] = yellow_green[red]
	}

	// The number of yellow letters that must be in the word, more is good
	// aaabb/ygggg menas that there must be 3 a's in the word
	for yellow, _ := range ret.must {
		ret.must[yellow] = yellow_green[yellow] - 1 // 0 is 1 or more letter, 1 is 2 or more, ....
	}
	return ret
}
func MakeLetterMatch2(guess, answer WordleWord) (must, mustNot []LetterCount) {
	ret := MakeLetterMatch(guess, answer)
	retMust := []LetterCount{}
	retMustNot := []LetterCount{}
	for letter, count := range ret.must {
		retMust = append(retMust, LetterCount{letter, count})
	}
	for letter, count := range ret.must_not {
		retMustNot = append(retMustNot, LetterCount{letter, count})
	}
	return retMust, retMustNot
}

type Answer struct {
	guess   WordleWord
	Colors  WordleWord
	must    []LetterCount
	mustNot []LetterCount
}

var Hitmiss map[string]Answer = make(map[string]Answer, 10000)
var HitCount int
var MissCount int

func WordleAnswer2(solution, guess WordleWord) Answer {
	key := string(solution[:]) + string(guess[:])
	if ret, ok := Hitmiss[key]; ok {
		HitCount++
		return ret
	}

	ret := Answer{
		guess: guess,
		// must:    []LetterCount{},
		// mustNot: []LetterCount{},
		must:    make([]LetterCount, 0, 5),
		mustNot: make([]LetterCount, 0, 5),
		Colors:  WordleWord([]rune{'r', 'r', 'r', 'r', 'r'}),
	}
	solutionNotGreenCount := [26]int{}
	guessYellowGreenCount := [26]int{}
	must := [26]bool{}
	mustNot := [26]bool{}
	for i, solutionLetter := range solution {
		guessLetter := guess[i]
		if solutionLetter == guessLetter {
			ret.Colors[i] = 'g'
			guessYellowGreenCount[guessLetter-'a'] = guessYellowGreenCount[guessLetter-'a'] + 1
		} else {
			// answer[i] = 'r'
			solutionNotGreenCount[solutionLetter-'a'] = solutionNotGreenCount[solutionLetter-'a'] + 1
		}
	}
	// turn the red to yellow if in the word but not green
	for i, guessLetter := range guess {
		if ret.Colors[i] == 'r' {
			if solutionNotGreenCount[guessLetter-'a'] > 0 {
				ret.Colors[i] = 'y'
				solutionNotGreenCount[guessLetter-'a'] = solutionNotGreenCount[guessLetter-'a'] - 1
				guessYellowGreenCount[guessLetter-'a'] = guessYellowGreenCount[guessLetter-'a'] + 1
			}
		}
	}
	for i, guessLetter := range guess {
		if ret.Colors[i] == 'r' {
			if !mustNot[guessLetter-'a'] {
				ret.mustNot = append(ret.mustNot, LetterCount{guessLetter, guessYellowGreenCount[guessLetter-'a']})
				mustNot[guessLetter-'a'] = true
			}
		} else if ret.Colors[i] == 'y' {
			if !must[guessLetter-'a'] {
				// add one for each red letter
				ret.must = append(ret.must, LetterCount{guessLetter, guessYellowGreenCount[guessLetter-'a'] - 1})
				must[guessLetter-'a'] = true
			}
		}
	}
	MissCount++
	Hitmiss[key] = ret
	return ret
}

// new try
// Matching returns the set of Matching words from the game's dictionary
func (wd *WordleMatcher) Matching(guess, answer WordleWord) []WordleWord {
	must, must_not := MakeLetterMatch2(guess, answer)
	return wd.matchingWorker(guess, answer, must, must_not)
}

func (wd *WordleMatcher) Matching2(answer Answer) []WordleWord {
	return wd.matchingWorker(answer.guess, answer.Colors, answer.must, answer.mustNot)
}

func (wd *WordleMatcher) MatchingWithCache(solution, guess WordleWord) []WordleWord {
	return wd.Matching2(WordleAnswer2(solution, guess))
}

//var Compliment []uint64

// [0] is BitSet for 1 bit.  Index off by 1
var bitsetAllSetPreAllocated []*bitset.BitSet = make([]*bitset.BitSet, 0)

// Length is 1..N
func NewBitsetAllSet(length int) *bitset.BitSet {
	if length < 1 {
		panic("bad length")
	}
	for i := len(bitsetAllSetPreAllocated); i < length; i++ {
		bitsetAllSetPreAllocated = append(bitsetAllSetPreAllocated, bitset.New(uint(i+1)).Complement())
	}
	set := make([]uint64, ((length-1)/64)+1)

	copy(set, bitsetAllSetPreAllocated[length-1].Bytes())
	ret := bitset.FromWithLength(uint(length), set)
	return ret
}

func (wd *WordleMatcher) matchingWorker(guess, answer WordleWord, must, must_not []LetterCount) []WordleWord {
	if len(guess) != 5 {
		panic("not 5 letter word:" + string(guess[:]))
	}
	if len(answer) != 5 {
		panic("not 5 letter word:" + string(answer[:]))
	}
	ret := NewBitsetAllSet(len(wd.words))
	// if there are greens then the starting point only contains words with matching letter
	for i, color := range answer {
		if color == 'g' {
			set := wd.letters[i][guess[i]]
			ret.InPlaceIntersection(set)
		}
	}

	// must letter is for yellow letters.  It indicates how many of these letters
	// must be in the word
	for _, letterCount := range must {
		yellow := letterCount.letter
		count := letterCount.count
		if counts, ok := wd.count[yellow]; ok {
			if len(counts) > count {
				set := counts[count]
				ret.InPlaceIntersection(set)
			}
		}
	}

	// red letters removes words that do not contain the required count of matching letters
	for _, letterCount := range must_not {
		red := letterCount.letter
		count := letterCount.count
		if counts, ok := wd.count[red]; ok {
			if len(counts) > count {
				set := counts[count]
				ret.InPlaceDifference(set)
			}
		}
	}

	// if there are yellow remove the words with matching letters - those would have been green
	// also remove any words that have the red letter in the same index
	for l, color := range answer {
		if color == 'y' {
			// words may not exist with this letter
			if wd.letters[l] != nil {
				if set, ok := wd.letters[l][guess[l]]; ok {
					ret.InPlaceDifference(set)
				}
			}
		}
		if color == 'r' {
			// words may not exist with this letter
			if wd.letters[l] != nil {
				if set, ok := wd.letters[l][guess[l]]; ok {
					ret.InPlaceDifference(set)
				}
			}
		}
	}
	indices := make([]uint, ret.Count())
	ret.NextSetMany(0, indices)
	retSlice := make([]WordleWord, len(indices))
	for i, index := range indices {
		retSlice[i] = wd.words[index]
	}
	return retSlice
}

func (wd *WordleMatcher) matchingWords(guess, answer string) []string {
	return []string{"todo", "todo2"}
}

// return the wordle answer for the quess given the solution
func WordleAnswer(solution, guess WordleWord) WordleWord {
	answer := WordleAnswer2(solution, guess)
	return answer.Colors
}

func WordleAnswerOrig(solution, guess WordleWord) WordleWord {
	var answer WordleWord
	var solution_not_green WordleWord
	for i, letter := range solution {
		if letter == guess[i] {
			answer[i] = 'g'
		} else {
			answer[i] = 'r'
			solution_not_green[letter] = solution_not_green[letter] + 1
		}
	}
	// turn the red to yellow if in the word but not green
	for i, letter := range guess {
		if answer[i] == 'r' {
			if solution_not_green[letter] > 0 {
				answer[i] = 'y'
				solution_not_green[letter] = solution_not_green[letter] - 1
			}
		}
	}
	return answer
}
