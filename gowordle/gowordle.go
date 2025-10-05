package gowordle

import (
	"container/heap"
	"fmt"
	"sort"

	mapset "github.com/deckarep/golang-set"
)

// MinHeap is a generic min-heap that can store any type T.
type MinHeap[T any] struct {
	data []T
	less func(a, b T) bool
}

func (h *MinHeap[T]) Len() int           { return len(h.data) }
func (h *MinHeap[T]) Less(i, j int) bool { return h.less(h.data[i], h.data[j]) }
func (h *MinHeap[T]) Swap(i, j int)      { h.data[i], h.data[j] = h.data[j], h.data[i] }

// Push adds an element to the heap.
func (h *MinHeap[T]) Push(x any) {
	h.data = append(h.data, x.(T))
}

// Pop removes the highest-priority element.
func (h *MinHeap[T]) Pop() any {
	n := len(h.data)
	item := h.data[n-1]
	h.data = h.data[0 : n-1]
	return item
}

// An Item is something we manage in a priority queue.
type Item struct {
	Value WordleWord // The value of the item; arbitrary.
	Score int        // The priority of the item in the queue.
	// The index is needed by update and is maintained by the heap.Interface methods.
}

func NewMinHeapWordleWordPriority() *MinHeap[Item] {
	ret := &MinHeap[Item]{
		data: []Item{},
		less: func(a, b Item) bool {
			// The priority queue will be based on the 'priority' field.
			return a.Score < b.Score
		},
	}
	heap.Init(ret)
	return ret
}

var s mapset.Set = nil

// WordleWordMap represents a map with ordered integer keys and values
// that are a slice of WordleWord.
type WordleWordMap struct {
	keys   []int
	values map[int][]WordleWord
}

// NewWordleWordMap creates and initializes a new WordleWordMap.
func NewWordleWordMap() *WordleWordMap {
	return &WordleWordMap{
		keys:   make([]int, 0),
		values: make(map[int][]WordleWord),
	}
}

// Set adds or updates a key-value pair.
// It appends a WordleWord to the slice associated with the given key.
func (om *WordleWordMap) Set(key int, word WordleWord) {
	if _, ok := om.values[key]; !ok {
		// Key does not exist, so add it to the keys slice.
		om.keys = append(om.keys, key)
		om.values[key] = make([]WordleWord, 0)
	}
	// Append the new word to the existing slice.
	om.values[key] = append(om.values[key], word)
}

// Get retrieves the slice of WordleWord for a given key.
func (om *WordleWordMap) Get(key int) ([]WordleWord, bool) {
	value, ok := om.values[key]
	return value, ok
}

// Len returns the number of unique keys in the map.
func (om *WordleWordMap) Len() int {
	return len(om.keys)
}

// Keys returns the ordered slice of keys.
func (om *WordleWordMap) Keys() []int {
	return om.keys
}

// Iterate provides a way to loop through the key-value pairs in order.
func (om *WordleWordMap) Iterate(f func(key int, values []WordleWord)) {
	for _, key := range om.keys {
		if values, ok := om.values[key]; ok {
			f(key, values)
		}
	}
}

// SortKeys sorts the keys in ascending order.
func (om *WordleWordMap) SortKeys() {
	sort.Ints(om.keys)
}

func wwsToString(ww []WordleWord) string {
	ret := ""
	sep := ""
	for _, w := range ww {
		ret = ret + sep + string(w[:])
		sep = ","
	}
	return ret
}

func StringsToWordleWords(words []string) []WordleWord {
	ret := make([]WordleWord, 0, len(words))
	for _, word := range words {
		rune_word := []rune(word)
		if len(rune_word) != 5 {
			panic("not 5 letter word:" + word)
		}
		ww := WordleWord(rune_word)
		ret = append(ret, ww)
	}
	return ret
}

func WordleWordsToStrings(words []WordleWord) []string {
	ret := make([]string, 0, len(words))
	for _, word := range words {
		ret = append(ret, string(word[:]))
	}
	return ret
}

func PrintWords(words []WordleWord) {
	for _, word := range words {
		fmt.Println(string(word[:]))
	}
}

type GuessAnswer struct {
	Guess  WordleWord
	Answer WordleWord
}

// play wordle against the computer providing the current board state
// return the next best answer
func PlayWorldReturnPossible(allWordleWords []WordleWord, guessAnswers []GuessAnswer) (WordleWord, []WordleWord) {
	possibleAnswers := allWordleWords

	for _, guessAnswer := range guessAnswers {
		game := NewWordleMatcher(possibleAnswers)
		possibleAnswers = game.Matching(guessAnswer.Guess, guessAnswer.Answer)
	}
	//ret := NextGuess(allWordleWords, possibleAnswers)
	ret := NextGuess1(allWordleWords, possibleAnswers)
	return ret, possibleAnswers
}

func PlayWordle(allWordleWords []WordleWord, guessAnswers []GuessAnswer) WordleWord {
	ret, _ := PlayWorldReturnPossible(allWordleWords, guessAnswers)
	return ret
}

func NextGuess1(allWords, possibleAnswers []WordleWord) WordleWord {
	_, wordsPossible := BestGuess1(allWords, possibleAnswers, possibleAnswers, 1, len(possibleAnswers)+1)
	return wordsPossible[0]
}

func FirstGuess1(allWords []string) (float32, []WordleWord) {
	wws := StringsToWordleWords(allWords)
	score, ret := BestGuess1(wws, wws, wws, 1, len(allWords))
	return float32(score), ret
}

func FirstGuessProvideInitialGuesses1(initialGuesses_s, allWords_s []string) (float32, []WordleWord) {
	allWords := StringsToWordleWords(allWords_s)
	initialGuesses := StringsToWordleWords(initialGuesses_s)
	// score, ret := BestGuess1(allWords, allWords, initialGuesses, 1, len(allwords))
	score, ret := BestGuess1(allWords, allWords, initialGuesses, 1, 10)
	return float32(score), ret
}

type WordFloat struct {
	word WordleWord
	flt  float32
}
type WordFloatByFloat []WordFloat

func (wf WordFloatByFloat) Len() int           { return len(wf) }
func (wf WordFloatByFloat) Swap(i, j int)      { wf[i], wf[j] = wf[j], wf[i] }
func (wf WordFloatByFloat) Less(i, j int) bool { return wf[i].flt < wf[j].flt }

// configurable from command line
var RECURSIVE bool = false

var matching2 bool = true
var Logging bool = false
var BetterGuesses map[string]int = make(map[string]int)

type gameCache struct {
	gameId int
	score  int
	words  []WordleWord
}

var gameCacheMap map[int]gameCache = make(map[int]gameCache)

func scoreForPossibleWords(gameId int) (int, []WordleWord, bool) {
	if ret, ok := gameCacheMap[gameId]; ok {
		return ret.score, ret.words, true
	}
	return 0, nil, false
}
func rememberScoreForPossibleWords(gameId int, score int, words []WordleWord) (int, []WordleWord) {
	if _, ok := gameCacheMap[gameId]; ok {
		panic("already have score for " + fmt.Sprintf("%d", gameId))
	}
	gameCacheMap[gameId] = gameCache{gameId, score, words}
	return score, words
}

// find best next guess, return the low score and the slice of words that have that score
// The score will be the average number of guesses it will take to solve if one the best guesses is used
func ScoreAlgorithmRecursive(allWords, possibleWords, _initialGuesses []WordleWord, depth int, bestScoreSoFar int) (int, []WordleWord) {
	const INIFINITY_SCORE = 1000000
	if len(possibleWords) == 0 {
		panic("possibleWords is empty")
	}
	if len(possibleWords) == 1 {
		return 100, possibleWords // just guess it
	}
	if len(possibleWords) == 2 {
		// if there are two words choose either of the words and the guesses will be 1 if the right guess and 2 if the wrong guess
		return 150, possibleWords
	}

	// Using the possible words the best guess is the matching one for one solution and 2 for the rest of the solutions.
	game := NewWordleMatcher(possibleWords)
	if retScore, retWordsWithScore, ok := scoreForPossibleWords(game.id); ok {
		return retScore, retWordsWithScore
	}
	possibleWordsSet := make(map[string]bool)
	for _, guess := range possibleWords {
		possibleWordsSet[string(guess[:])] = true
	}
	guessesInPossibleWords := make([]WordleWord, 0)
	guessesNotInPossibleWords := make([]WordleWord, 0)
	flagGuessCountCutOff := 0
	if true {
		maxGuessCount := 300
		flagGuessCountCutOff = maxGuessCount - 100
		sortedScores := ScoreAlgorithmTotalMatches1LevelAll(allWords, possibleWords, _initialGuesses, depth, bestScoreSoFar)
		for guessCount := 0; (sortedScores.Len() > 0) && (guessCount < maxGuessCount); guessCount++ {
			item := heap.Pop(sortedScores).(Item)
			guess := item.Value
			if _, ok := possibleWordsSet[string(guess[:])]; ok {
				guessesInPossibleWords = append(guessesInPossibleWords, guess)
			} else {
				guessesNotInPossibleWords = append(guessesNotInPossibleWords, guess)
			}
		}
	} else {
		flagGuessCountCutOff = 1000000
		for _, guess := range allWords {
			if _, ok := possibleWordsSet[string(guess[:])]; ok {
				guessesInPossibleWords = append(guessesInPossibleWords, guess)
			} else {
				guessesNotInPossibleWords = append(guessesNotInPossibleWords, guess)
			}
		}
	}

	bestScore := 10000 // start with a high score
	// assume best possible score is a correct guess (100) and getting all the rest of the solutions in 2 guesses
	bestPossibleScore := (100 + 200*(len(possibleWords)-1)) / len(possibleWords)
	var bestGuess []WordleWord
	for guessCount, guess := range append(guessesInPossibleWords, guessesNotInPossibleWords...) {
		score := 0 // running average
		guessInPossibleWordsRemaining := false
		if guessCount < len(guessesInPossibleWords) {
			// this is a guess that is in the possible words
			guessInPossibleWordsRemaining = true
		} else {
			// the guess is not in the possible words, so the best possible score is a guess (100) that narrows it down to 1 quess (100)
			bestPossibleScore = 200
		}
		if bestScore <= bestPossibleScore {
			break
		}
		for count, solution := range possibleWords {
			matching := game.Matching2(WordleAnswer2(solution, guess))
			/*
				if len(matching) == len(possibleWords) {
					// not narrowing it down any this solution so it is a bad guess, go to next guess
					score = INIFINITY_SCORE
					break
				}
			*/
			if depth > 5 || len(matching) == len(possibleWords) {
				score = INIFINITY_SCORE
				break // this guess is bad move to the next guess
			}
			// Score the guess for this solution
			guessSolutionScore := 100 // one guess is 100 points
			if (len(matching) == 1) && (string(matching[0][:]) == string(guess[:])) {
				guessInPossibleWordsRemaining = false // this is the correct guess
			} else {
				subscore, _ := ScoreAlgorithmRecursive(allWords, matching, matching, depth+1, bestScoreSoFar)
				guessSolutionScore += subscore
			}
			score = score + ((guessSolutionScore - score) / (count + 1)) // running average

			// 200 is the best for the remaining words, if the current average plus best possible result for the remaining words
			// is alread over that may as well quit
			bestPossibleScoreForThisGuess := ((score * (count + 1)) + (200 * (len(possibleWords) - (count + 1)))) / len(possibleWords)
			if guessInPossibleWordsRemaining {
				// if a correct guess is coming up then the score is 100 for the matching guess and 200 for the rest
				if len(possibleWords) < (count + 2) {
					panic("bad count")
				}
				bestPossibleScoreForThisGuess = ((score * (count + 1)) + 100 + (200 * (len(possibleWords) - (count + 2)))) / len(possibleWords)
			}
			if bestPossibleScoreForThisGuess > bestScore {
				score = bestPossibleScoreForThisGuess // greater then bestScore is all that matters
				break
			}
		}
		if score == INIFINITY_SCORE {
			continue // this guess is bad move to the next guess
		}

		if score < bestScore {
			if guessCount > flagGuessCountCutOff {
				fmt.Println("old/new:", bestScore, score, "depth:", depth, "guessCount:", guessCount, "guess:", string(guess[:]), "possibleWords:", len(possibleWords), WordleWordsToStrings(possibleWords))
			}
			bestScore = score
			bestGuess = []WordleWord{guess}

		} else if score == bestScore {
			bestGuess = append(bestGuess, guess)
		}
	}
	return rememberScoreForPossibleWords(game.id, bestScore, bestGuess)
}

/*************
// find best next guess, return the low score and the slice of words that have that score
// The score will be the average number of guesses it will take to solve if one the best guesses is used
func ScoreAlgorithmRecursive_try1(allWords, possibleWords, _initialGuesses []WordleWord, depth int, bestScoreSoFar int) (int, []WordleWord) {
	// Using the possible words the best guess is the matching one for one solution and 2 for the rest of the solutions.
	game := NewWordleMatcher(possibleWords)
	if score, possibleWords, ok := scoreForPossibleWords(game.id); ok {
		return score, possibleWords
	}

	const INIFINITY_SCORE = 1000000
	if len(possibleWords) == 0 {
		panic("possibleWords is empty")
	}
	if len(possibleWords) == 1 {
		return 100, possibleWords // just guess it
	}
	if len(possibleWords) == 2 {
		// if there are two words choose either of the words and the guesses will be 1 if the right guess and 2 if the wrong guess
		return 150, possibleWords
	}
	bestScore := 1000
	var bestGuess []WordleWord
	possibleWordsSet := make(map[string]bool)
	for _, guess := range possibleWords {
		score := 0 // running average
		for count, solution := range possibleWords {
			matching := game.MatchingWithCache(solution, guess)
			// matching := game.Matching2(WordleAnswer2(solution, guess))
			subscore, _ := ScoreAlgorithmRecursive_try1(allWords, matching, matching, depth+1, bestScoreSoFar)
			if subscore == INIFINITY_SCORE {
				score = INIFINITY_SCORE
				break // this guess is bad move to the next guess
			}
			if !((len(matching) == 1) && (string(matching[0]) == string(guess))) {
				subscore += 100 // if the guess is the solution then the subscore will be 100, otherwise it will be the current guess pluss the recursive guess
			}
			score = score + ((subscore - score) / (count + 1))
		}
		if score == INIFINITY_SCORE {
			continue // this guess is bad move to the next guess
		}
		// all but one of the guesses is incorrect (hence the -1) then each of the second guesses takes only 1 guess
		if score <= (((len(possibleWords)-1)+len(possibleWords))*100)/len(possibleWords) {
			// found this guess compared to all the possible words returns the correct answer (1) or narrows down to a single answer for the correct guess
			return rememberScoreForPossibleWords(game.id, score, []WordleWord{guess})
		}
		possibleWordsSet[string(guess)] = true
		// although not best possible it could still be the best so far
		if score < bestScore {
			bestScore = score
			bestGuess = []WordleWord{guess}
		} else if score == bestScore {
			bestGuess = append(bestGuess, guess)
		}
	}
	// bestScorePossibleWord := bestScore

	// the best that can be done by using a non possible word is finding a guess that narrows it down to 1 in all cases, thus taking 2 guesses total
	// so if that has already been achieved use the best guess
	if bestScore <= 200 {
		return rememberScoreForPossibleWords(game.id, bestScore, bestGuess)
	}

	// did not find an optimal answer using just the possible words, try all the words
	for guessCount, guess := range allWords {
		if _, ok := possibleWordsSet[string(guess)]; ok {
			// already tried this guess
			continue
		}
		score := 0 // running average
		for count, solution := range possibleWords {
			matching := game.Matching2(WordleAnswer2(solution, guess))
			if depth > 5 || len(matching) == len(possibleWords) {
				score = INIFINITY_SCORE
				break // this guess is bad move to the next guess
			}
			subscore, _ := ScoreAlgorithmRecursive_try1(allWords, matching, matching, depth+1, bestScoreSoFar)
			subscore += 100                                    // current score plus the recursive score
			score = score + ((subscore - score) / (count + 1)) // running average

			// 200 is the best for the remaining words, if the current average plus best possible result for the remaining words
			// is alread over that may as well quit
			bestPossibleScore := ((score * (count + 1)) + (200 * (len(possibleWords) - (count + 1)))) / len(possibleWords)
			if bestPossibleScore > bestScore {
				score = bestPossibleScore
				break
			}
		}
		if score == INIFINITY_SCORE {
			continue // this guess is bad move to the next guess
		}

		if score < bestScore {
			bestScore = score
			bestGuess = []WordleWord{guess}
			if bestScore == 200 {
				// the best score possible is 200, so not going to get all of them but this may save time.
				break
			}
		} else if score == bestScore {
			bestGuess = append(bestGuess, guess)
		}
		// dead code
		if depth == 7 {
			fmt.Println("guessCount:", guessCount, "guess:", string(guess), "bestScore:", bestScore, "bestGuess:", WordleWordsToStrings(bestGuess))
		}
	}
	return rememberScoreForPossibleWords(game.id, bestScore, bestGuess)
}
***************/

func ScoreAlgorithmTotalMatches1Level(allWords, possibleWords, initialGuesses []WordleWord, depth int, bestScoreSoFar int) (int, []WordleWord) {
	minHeap := ScoreAlgorithmTotalMatches1LevelAll(allWords, possibleWords, initialGuesses, depth, bestScoreSoFar)
	ret := heap.Pop(minHeap).(Item)
	return ret.Score, []WordleWord{ret.Value}
}

// total number of words
func GuessScore(guess WordleWord, possibleWords []WordleWord, allWords []WordleWord, depth int) int {
	game := NewWordleMatcher(possibleWords)
	score := 0
	guessInPossibleWords := false
	for _, solution := range possibleWords {
		if string(solution[:]) == string(guess[:]) {
			guessInPossibleWords = true
		}
		matching := game.Matching2(WordleAnswer2(solution, guess))
		score += len(matching)
	}
	if guessInPossibleWords && score >= 2 {
		score -= 2
	}
	return score
}

// try all the guesses and return a map of score to guess.
func ScoreAlgorithmTotalMatches1LevelAll(allWords, possibleWords, initialGuesses []WordleWord, depth int, bestScoreSoFar int) *MinHeap[Item] {
	ret := NewMinHeapWordleWordPriority()
	if len(possibleWords) == 0 {
		panic("possibleWords is empty")
	}
	if len(possibleWords) == 1 {
		heap.Push(ret, Item{Value: possibleWords[0], Score: 1})
		return ret
	}

	// use possible words first - then rest of the words
	initialGuessMap := make(map[string]bool, len(initialGuesses))
	orderedGuesses := make([]WordleWord, len(initialGuesses))
	copy(orderedGuesses, initialGuesses)

	for _, guess := range initialGuesses {
		initialGuessMap[string(guess[:])] = true
	}
	for _, guess := range allWords {
		if _, ok := initialGuessMap[string(guess[:])]; !ok {
			orderedGuesses = append(orderedGuesses, guess)
		}
	}
	for _, guess := range orderedGuesses {
		score := GuessScore(guess, possibleWords, allWords, depth)
		heap.Push(ret, Item{Value: guess, Score: score})
	}
	return ret
}

var BestGuess1 func([]WordleWord, []WordleWord, []WordleWord, int, int) (int, []WordleWord) = ScoreAlgorithmTotalMatches1Level

// Simulate a game of wordle.
// words_s - dictionary of words
// solution - answer
// first_guess - first guess
func Simulate(words_s []string, solution_s string, first_guess_s string) []string {
	words := StringsToWordleWords(words_s)
	solution := WordleWord([]rune(solution_s))
	guess := WordleWord([]rune(first_guess_s))
	guesses := []string{}
	gas := make([]GuessAnswer, 0)
	for guessCount := 0; guessCount < 6; guessCount++ {
		guesses = append(guesses, string(guess[:]))
		answer := WordleAnswer(solution, guess)
		if string(answer[:]) == "ggggg" {
			return guesses
		}
		gas = append(gas, GuessAnswer{guess, WordleAnswer(solution, guess)})
		guess = PlayWordle(words, gas)
	}
	panic("unexpected Simulate end")
}

type SolutionsAnswers struct {
	Solutions    []string
	AnswerColors []string
}

// for the given guess return a map key = matching possible solutions, value = slice of Solutions and answerColors
func UniqueGuessResults(wordListStrings []string, guess string) map[string]SolutionsAnswers {
	guessWW := WordleWord([]rune(guess))
	wordList := StringsToWordleWords(wordListStrings)
	game := NewWordleMatcher(wordList)
	sortedSolutions := make(map[string]SolutionsAnswers)
	for _, solution := range wordList {
		answer := WordleAnswer2(solution, guessWW)
		possibleSolutions := game.Matching(guessWW, answer.Colors)
		sort.Slice(possibleSolutions, func(i, j int) bool {
			return string(possibleSolutions[i][:]) < string(possibleSolutions[j][:])
		})
		allSolutions := ""
		for _, solution := range possibleSolutions {
			allSolutions += string(solution[:]) + " "
		}
		if solutionAnswers, ok := sortedSolutions[allSolutions]; ok {
			newSolutionAnswers := sortedSolutions[allSolutions]
			newSolutionAnswers.Solutions = append(solutionAnswers.Solutions, string(solution[:]))
			if solutionAnswers.AnswerColors[0] != string(answer.Colors[:]) {
				newSolutionAnswers.AnswerColors = append(solutionAnswers.AnswerColors, "BUG-"+string(solution[:])+"-"+string(answer.Colors[:]))
			}
			sortedSolutions[allSolutions] = newSolutionAnswers
		} else {
			sortedSolutions[allSolutions] = SolutionsAnswers{Solutions: []string{string(solution[:])}, AnswerColors: []string{string(answer.Colors[:])}}
		}
	}
	return sortedSolutions
}

// for the given guess return a map key = answer colors value = matching solutions
func UniqueAnswerResults(wordListStrings []string, guess string) map[string][]string {
	wordList := StringsToWordleWords(wordListStrings)
	guessWW := WordleWord([]rune(guess))
	answerSolutions := make(map[string][]string)
	for _, solutionWW := range wordList {
		answer := WordleAnswer2(solutionWW, guessWW)
		answerColors := string(answer.Colors[:])
		if answers, ok := answerSolutions[string(answerColors)]; ok {
			answerSolutions[answerColors] = append(answers, string(solutionWW[:]))
		} else {
			answerSolutions[answerColors] = []string{string(solutionWW[:])}
		}
	}
	return answerSolutions
}
