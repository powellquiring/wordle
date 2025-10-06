package wordle

import (
	"container/heap"
	"fmt"
	"sort"

	//	"github.com/bits-and-blooms/bitset"
	"github.com/powellquiring/wordle/bitset"
	"github.com/powellquiring/wordle/gowordle"
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

type WordScore struct {
	Value WordleWord
	Score uint16 // The priority of the item lower is better.
}

type WordScoreSorter []WordScore

/*
	func NewWordScoreHeap(len int) *WordScoreSorter {
		wordScores := make([]WordScore, 0, len)
		return (*WordScoreSorter)(&wordScores)
	}
*/
func (h *WordScoreSorter) Reset() {
	*h = (*h)[:0]
}

func (h *WordScoreSorter) Push(x WordScore) {
	if len(*h) >= cap(*h) {
		panic("WordScoreHeap is full")
	}
	*h = append(*h, x)
}

var fullAnswerCacheHitCount int
var fullAnswerCacheMissCount int
var cacheSolutionGuess bool = true

func (d *Dictionary) GoWordleSliceToWordList(goMatching []gowordle.WordleWord) *WordList {
	possibleWords := d.WordlistEmpty()
	for _, goWord := range goMatching {
		wordString := string(goWord[:])
		word, ok := d.Word(wordString)
		if !ok {
			panic("word not in dictionary: " + wordString)
		}
		possibleWords.Insert(word)
	}
	return possibleWords
}

func (d *Dictionary) GetFullAnswerForDictionary(wordlist *WordList, solution WordleWord, guess WordleWord) FullAnswer {
	goAnswer := gowordle.WordleAnswer2(gowordle.WordleWord([]rune(d.String(solution))), gowordle.WordleWord([]rune(d.String(guess))))
	goColor := string(goAnswer.Colors[:])
	goMatching := d.matcher.Matching2(goAnswer)
	color, ok := StringToAnswer(goColor)
	if !ok {
		panic("Color not valid: " + goColor)
	}
	possibleWords := d.GoWordleSliceToWordList(goMatching)
	return FullAnswer{AnswerColor: color, AnswerMatching: possibleWords}
}

const fullAnswerCachePrintCount int = 0

// given a wordlist a solution and a guess return the answer and new wordlist
func (d *Dictionary) GetFullAnswerForDictionaryWithCache(wordlist *WordList, solution WordleWord, guess WordleWord) FullAnswer {
	if (fullAnswerCachePrintCount > 0) && (fullAnswerCacheHitCount+fullAnswerCacheMissCount)%10_000_000 == 0 {
		fmt.Println("fullAnswerCacheHitCount:", fullAnswerCacheHitCount, "fullAnswerCacheMissCount:", fullAnswerCacheMissCount)
	}
	if d.fullAnswerCache[solution][guess].AnswerMatching != nil {
		fullAnswerCacheHitCount++
		return d.fullAnswerCache[solution][guess]
	} else {
		fullAnswerCacheMissCount++
		fullAnswer := d.GetFullAnswerForDictionary(wordlist, solution, guess)
		d.fullAnswerCache[solution][guess] = fullAnswer
		return fullAnswer
	}
}

func (d *Dictionary) GetFullAnswer(wordlist *WordList, solution WordleWord, guess WordleWord, possibleWords *WordList) FullAnswer {
	var fullAnswer FullAnswer
	if cacheSolutionGuess {
		fullAnswer = d.GetFullAnswerForDictionaryWithCache(wordlist, solution, guess)
	} else {
		fullAnswer = d.GetFullAnswerForDictionary(wordlist, solution, guess)
	}
	bs := (*bitset.BitSet)(fullAnswer.AnswerMatching)
	bs.IntersectionInPlace((*bitset.BitSet)(wordlist), (*bitset.BitSet)(possibleWords))
	return FullAnswer{AnswerColor: fullAnswer.AnswerColor, AnswerMatching: (*WordList)(possibleWords)}
}

var subscoreCacheHit int
var subscoreCacheMiss int
var subscoreCache = make(map[WordList](int))

func subscoreCacheGet(matching *WordList) (int, bool) {
	if false && ((subscoreCacheHit+subscoreCacheMiss)%10_000_000 == 0) {
		fmt.Println("Subscore cache hit/miss: ", subscoreCacheHit, subscoreCacheMiss)
	}
	ret, ok := subscoreCache[*matching]
	if ok {
		subscoreCacheHit++
	} else {
		subscoreCacheMiss++
	}
	return ret, ok
}

func subscoreCacheSet(matching *WordList, subscore int) {
	subscoreCache[*matching] = subscore
}

var wordScoreSorter WordScoreSorter

func (d *Dictionary) SortedGuesses(possibleWords *WordList, depth int) *WordScoreSorter {
	// possible words are allocated here to minimize the number of initializations
	var fullanswerPossibleWords WordList
	wordScoreSorter := &wordScoreSorter
	if len(*wordScoreSorter) == 0 {
		*wordScoreSorter = make([]WordScore, d.Len())
	}
	wordScoreSorter.Reset()
	for _, guess := range d.WordlistAll().Range {
		score := 0
		guessInPossibleWords := false
		for _, solution := range possibleWords.Range {
			fullAnswer := d.GetFullAnswer(possibleWords, solution, guess, &fullanswerPossibleWords)
			matching := fullAnswer.AnswerMatching
			matchingLen := matching.Len()
			score += matchingLen
			if guess == solution {
				guessInPossibleWords = true
			}
		}
		if guessInPossibleWords == true {
			score -= 2
		}
		wordScoreSorter.Push(WordScore{Value: guess, Score: uint16(score)})
	}
	sort.Slice(*wordScoreSorter, func(i, j int) bool {
		return (*wordScoreSorter)[i].Score < (*wordScoreSorter)[j].Score
	})
	return wordScoreSorter
}

var depthExceededCount int

// starting with a subset of the dictionary words (wordlist) give a score to each word in the dictionary
// based on "guess score" for that word.
func (d *Dictionary) NextGuessSearch(possibleWords *WordList, depth int) (int, WordleWord) {
	const INIFINITY_SCORE = 1000000

	// possible words are allocated here to minimize the number of initializations
	var fullanswerPossibleWords WordList

	if depth > 14 {
		depthExceededCount++
		fmt.Println("depth:", depth, "possibleWords:", depthExceededCount, possibleWords.Len(), d.WordlistStrings(possibleWords))
	}
	possibleWordsLen := possibleWords.Len()
	if possibleWordsLen == 0 {
		panic("possibleWords is empty")
	}
	if possibleWordsLen == 1 {
		return 100, possibleWords.FirstWord() // just guess it
	}
	if possibleWordsLen == 2 {
		// if there are two words choose either of the words and the guesses will be 1 if the right guess and 2 if the wrong guess
		return 150, possibleWords.FirstWord()
	}

	// Using the possible words the best guess is the matching one for one solution and 2 for the rest of the solutions.
	// if retScore, retWordsWithScore, ok := scoreForPossibleWords(game.id); ok {
	// 	return retScore, retWordsWithScore
	// }
	possibleWordsSet := make(map[WordleWord]bool)
	for _, guess := range possibleWords.Range {
		possibleWordsSet[guess] = true
	}
	guessesInPossibleWords := make([]WordleWord, 0)
	guessesNotInPossibleWords := make([]WordleWord, 0)
	flagGuessCountCutOff := 0
	if false {
		maxGuessCount := 300
		flagGuessCountCutOff = maxGuessCount - 100
		// sortedScores := ScoreAlgorithmTotalMatches1LevelAll(allWords, possibleWords, _initialGuesses, depth, bestScoreSoFar)
		/*
			sortedScores := d.WordlistAll()
			for guessCount := 0; (sortedScores.Len() > 0) && (guessCount < maxGuessCount); guessCount++ {
				item := heap.Pop(sortedScores).(Item)
				guess := item.Value
				if _, ok := possibleWordsSet[string(guess[:])]; ok {
					guessesInPossibleWords = append(guessesInPossibleWords, guess)
				} else {
					guessesNotInPossibleWords = append(guessesNotInPossibleWords, guess)
				}
			}
		*/
	} else {
		flagGuessCountCutOff = 140
		maxGuessCount := 200
		if maxGuessCount > d.Len() {
			maxGuessCount = d.Len()
		}
		wordScoreSorter := d.SortedGuesses(possibleWords, depth)
		// sortedScores := d.SortedGuesses(possibleWords, wordScoreSorter)
		for guessCount := 0; (len(*wordScoreSorter) > 0) && (guessCount < maxGuessCount); guessCount++ {
			item := (*wordScoreSorter)[guessCount]
			guess := item.Value
			if _, ok := possibleWordsSet[guess]; ok {
				guessesInPossibleWords = append(guessesInPossibleWords, guess)
			} else {
				guessesNotInPossibleWords = append(guessesNotInPossibleWords, guess)
			}
		}
	}

	bestScore := INIFINITY_SCORE
	// assume best possible score is a correct guess (100) and getting all the rest of the solutions in 2 guesses
	var bestGuess WordleWord
	// TODO
	for guessCount, guess := range append(guessesInPossibleWords, guessesNotInPossibleWords...) {
		//guessesLen := len(guessesInPossibleWords)
		score := 0 // running average
		guessInPossibleWordsRemaining := false
		bestPossibleScore := (100 + 200*(possibleWordsLen-1)) / possibleWordsLen
		if guessCount < len(guessesInPossibleWords) {
			// this is a guess that is in the possible words
			guessInPossibleWordsRemaining = true
		} else {
			bestPossibleScore = 200
		}
		if bestScore <= bestPossibleScore {
			// not going to add any more identical scores to the best guess list
			break
		}
		for count, solution := range possibleWords.Range {
			fullAnswer := d.GetFullAnswer(possibleWords, solution, guess, &fullanswerPossibleWords)
			matching := fullAnswer.AnswerMatching
			matchingLen := matching.Len()
			//	if len(matching) == possibleWordsLen {
			//		// not narrowing it down any this solution so it is a bad guess, go to next guess
			//		score = INIFINITY_SCORE
			//		break
			//	}
			if depth > 17 || matchingLen == possibleWordsLen {
				score = INIFINITY_SCORE
				break // this guess is bad move to the next guess
			}
			// Score the guess for this solution
			guessSolutionScore := 100 // one guess is 100 points
			if (matchingLen == 1) && (matching.FirstWord() == guess) {
				guessInPossibleWordsRemaining = false // this is the correct guess
			} else {
				var subscore int
				if subscoreCached, ok := subscoreCacheGet(matching); ok {
					subscore = subscoreCached
				} else {
					subscore, _ = d.NextGuessSearch(matching, depth+1)
					if subscore != INIFINITY_SCORE {
						subscoreCacheSet(matching, subscore)
					} else {
						fmt.Println("subscor cache miss")
					}
				}
				guessSolutionScore += subscore
			}
			score = score + ((guessSolutionScore - score) / (count + 1)) // running average

			// 200 is the best for the remaining words, if the current average plus best possible result for the remaining words
			// is alread over that may as well quit
			bestPossibleScoreForThisGuess := ((score * (count + 1)) + (200 * (possibleWordsLen - (count + 1)))) / possibleWordsLen
			if guessInPossibleWordsRemaining {
				// if a correct guess is coming up then the score is 100 for the matching guess and 200 for the rest
				if possibleWordsLen < (count + 2) {
					panic("bad count")
				}
				bestPossibleScoreForThisGuess = ((score * (count + 1)) + 100 + (200 * (possibleWordsLen - (count + 2)))) / possibleWordsLen
			}
			if bestPossibleScoreForThisGuess > bestScore {
				score = bestPossibleScoreForThisGuess // greater then bestScore is all that matters
				break
			}
		}
		if score == INIFINITY_SCORE {
			//if (bestGuess == nil) && (guessCount == (guessesLen - 1)) {
			//	fmt.Println("no score. depth:", depth, "guessCount:", guessCount, "possibleWords:", possibleWordsLen, d.WordlistStrings(possibleWords))
			//}
			continue // this guess is bad move to the next guess
		}

		if score < bestScore {
			if false && guessCount > flagGuessCountCutOff {
				fmt.Println("old/new:", bestScore, score, "depth:", depth, "guessCount:", guessCount, "guess:", d.String(guess), "possibleWords:", possibleWordsLen, d.WordlistStrings(possibleWords))
			}
			bestScore = score
			bestGuess = guess

			//} else if score == bestScore {
			//	bestGuess.Insert(guess)
		}
		if false && (depth == 0) {
			fmt.Println("depth guessCount score", depth, guessCount, score)
		}
	}
	return bestScore, bestGuess
}

// simulate one game given the first word and the solution
func SimulateOneGameGivenFirstWord(dictionary *Dictionary, solution WordleWord, initialGuesses []WordleWord) []WordleWord {
	// possible words are allocated here to minimize the number of initializations
	var fullanswerPossibleWords WordList
	guesses := []WordleWord{}
	matchingWords := dictionary.WordlistAll()
	for guessCount := range 8 {
		var nextGuess WordleWord
		if guessCount < len(initialGuesses) {
			nextGuess = initialGuesses[guessCount]
		} else {
			nextGuess = dictionary.NextGuess(matchingWords)
		}
		guesses = append(guesses, nextGuess)
		fullAnswer := dictionary.GetFullAnswer(matchingWords, solution, nextGuess, &fullanswerPossibleWords)
		matchingWords = fullAnswer.AnswerMatching
		if matchingWords.Len() == 1 {
			words := matchingWords.Words()
			if words[0] == solution {
				if words[0] != nextGuess {
					// if the solution is the nextGuess it was already added, do not need to add it again
					guesses = append(guesses, words[0])
				}
				return guesses
			} else {
				panic("unexpected Simulate end")
			}
		}
	}
	panic("unexpected Simulate end")
}

type GuessAnswer struct {
	Guess  string
	Answer string
}

// play wordle against the computer providing the current board state
// return the next best answer
func (d *Dictionary) PlayWorldReturnPossible(guessAnswers []GuessAnswer) (WordleWord, *WordList) {
	goMatching := []gowordle.WordleWord{}

	var game *gowordle.WordleMatcher
	for guessCount, guessAnswer := range guessAnswers {
		if guessCount == 0 {
			game = d.matcher
		} else {
			game = gowordle.NewWordleMatcher(goMatching)
		}
		goGuess := gowordle.WordleWord([]rune(guessAnswer.Guess))
		goAnswer := gowordle.WordleWord([]rune(guessAnswer.Answer))
		goMatching = game.Matching(goGuess, goAnswer)
	}
	possibleAnswers := d.GoWordleSliceToWordList(goMatching)
	ret := d.NextGuess(possibleAnswers)
	return ret, possibleAnswers
}
