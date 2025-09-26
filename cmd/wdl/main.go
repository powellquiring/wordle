package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/powellquiring/wordle/wordle"
	"github.com/urfave/cli/v3" // imports as package "cli"
)

func cache(globalConfig GlobalConfiguration) {
	wordList := wordle.StringsToWordleWords(globalConfig.AllWords)
	game := wordle.NewWordleMatcher(wordList)
	result := make(map[GuessSolution]AnswerWords)
	for _, guess := range wordList {
		for _, solution := range wordList {
			answer := wordle.WordleAnswer2(solution, guess)
			matching := game.Matching2(answer)
			answerWords := AnswerWords{answer, matching}
			result[GuessSolution{guess, solution}] = answerWords
		}
	}
	fmt.Println("Done")
}

type GlobalConfiguration struct {
	AllWords  []string
	Recursive bool
	progress  bool
	FirstWord string
}

func globalCofiguration(count int, recursive bool, progress bool, firstWord string) GlobalConfiguration {
	if count == 0 {
		count = len(wordle.SortedWordleDictionary())
	}
	wordle.RECURSIVE = recursive
	if recursive {
		wordle.BestGuess1 = wordle.ScoreAlgorithmRecursive
	}
	if firstWord == "" {
		firstWord = "raise"
	}
	return GlobalConfiguration{
		AllWords:  wordle.SortedWordleDictionary()[0:count],
		Recursive: recursive,
		progress:  progress,
		FirstWord: firstWord,
	}
}

func main() {
	count := 0
	recursive := false
	progress := false
	firstWord := ""
	// going raise blunt
	//server(globalCofiguration(count, recursive, progress, firstWord), "going", []string{"raise", "blunt"})
	// FirstWords(globalCofiguration(count, recursive, progress, firstWord))
	// playWordle(globalCofiguration(count, true, progress, firstWord), []string{"raise", "ryyry"})
	// simulate(globalCofiguration(count, true, progress, firstWord), []string{})
	cmd := &cli.Command{
		Name:  "wdl",
		Usage: "wordle",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:        "count",
				Value:       0,
				Aliases:     []string{"c"},
				Usage:       "number of words, 0 is all words",
				Destination: &count,
			},
			&cli.BoolFlag{
				Name:        "recursive",
				Value:       false,
				Aliases:     []string{"r"},
				Usage:       "turn on recursive flag slower but better",
				Destination: &recursive,
			},
			&cli.BoolFlag{
				Name:        "progress",
				Value:       false,
				Aliases:     []string{"p"},
				Usage:       "show progress bar",
				Destination: &progress,
			},
			&cli.BoolFlag{
				Name:        "progress",
				Value:       false,
				Aliases:     []string{"p"},
				Usage:       "show progress bar",
				Destination: &progress,
			},
			&cli.StringFlag{
				Name:        "first",
				Value:       "",
				Aliases:     []string{"f"},
				Usage:       "first word to guess, default is 'raise', only used with sim command",
				Destination: &firstWord,
			},
		},
		Commands: []*cli.Command{
			{
				Name:  "cache",
				Usage: "build a cache[guess][solution] = Answer",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					cache(globalCofiguration(count, recursive, progress, firstWord))
					return nil
				},
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
