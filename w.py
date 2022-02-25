#!/usr/bin/env python
import requests
import re
import json
from bs4 import BeautifulSoup
import pathlib
import collections
import itertools

def wordle_answer(solution, guess):
  "return the wordle answer for the quess given the solution"
  answer = ""
  solution_not_green = ""
  for i, letter in enumerate(guess):
    if letter == solution[i]:
      answer = answer + "g"
    else:
      answer = answer + "r"
      solution_not_green = solution_not_green + solution[i]
  for i, letter in enumerate(guess):
    if answer[i] == "r":
      if (found := solution_not_green.find(letter)) >= 0:
        answer = answer[0:i] + "y" + answer[i+1:]
        solution_not_green = solution_not_green[0:found] + solution_not_green[found+1:]
  return answer

class Letters:
  def __init__(self, guess: list[str], answer: list[str]):
    GREEN = "g"
    YELLOW = "y"
    self.green = list()
    self.yellow = list()
    self.red = list()
    self.yellow_letters = ""
    self.red_letters = ""
    for i,l in enumerate(guess):
      if answer[i] == GREEN:
        self.green.append((i,l))
      elif answer[i] == YELLOW:
        self.yellow.append((i,l))
        self.yellow_letters = self.yellow_letters + l
      else:
        self.red.append((i,l))
        self.red_letters = self.red_letters + l

class Lettersorig:
  def __init__(self, guess: list[str], answer: list[str]):
    GREEN = "g"
    YELLOW = "y"
    self.green = ""
    self.yellow = ""
    self.red = ""
    for i in range(len(guess)):
      if answer[i] == GREEN:
        self.green = self.green + guess[i]
      elif answer[i] == YELLOW:
        self.yellow = self.yellow + guess[i]
      else:
        self.red = self.red + guess[i]

class Words:
  """Dictionay and wordle"""
  def __init__(self, words:set=None):
    self.unique_letters_in_word = dict()
    if words == None:
      self.words = set()
    else:
      self.words = words
      #for word in words:
      #  word = word.lower()
      #  self.words.add(word)

  def word_list(self):
     return sorted(list(self.words))

  def read_file(self, dir_s, file):
    dir = pathlib.Path(dir_s)
    assert (words := dir / file).exists()
    with words.open("r") as f:
      word_list = json.load(f)
      for word in word_list:
        self.add(word)

  def add(self, word):
    word = word.lower()
    self.words.add(word)
    unique_letters = set()
    for letter in word:
      unique_letters.add(letter)
    for letter in unique_letters:
      self.unique_letters_in_word[letter] = self.unique_letters_in_word.get(letter, 0) + 1
  def popular_letters(self):
    ret = list()
    for letter, count in sorted(self.unique_letters_in_word.items(), key=lambda item: item[1], reverse=True):
      ret.append(letter)
    return ret
  def find(self, letters):
    """Find words that conain these letters once in the word"""
    ret = set()
    for word in self.words:
      shrinking_word = word
      count = 0
      for letter in letters:
        if (i := shrinking_word.find(letter)) >= 0:
          shrinking_word = shrinking_word[:i] + shrinking_word[i+1:]
          count = count + 1
      if count >= 5:
        ret.add(word)
    return ret
  
  def words_containing_letter(self) -> dict:
    """
    set of words that contain a letter. 
    ['a'][0] - words that contain 1,2,3,4,5 a's
    ['z'][1] - words that contain 1,2,3,4,5 z
    ['z'][1] - words that contain 1 or 2 z
    """
    if not hasattr(self, "_words_containing_letter"):
      self._words_containing_letter = dict()
      for word in self.words:
        word_letter_set = set(word)
        for l in word_letter_set:
          if l not in self._words_containing_letter:
            self._words_containing_letter[l] = list([set(),set(),set(),set(),set()])
          index = word.count(l)
          for s in self._words_containing_letter[l][:index]:
            s.add(word)
    return self._words_containing_letter

  def words_filter(self, guess: str, answer:str, guess_letters:Letters) -> set:
    "filter outthe words that can not possibly match by using indexed sets"
    def words_containing_letter(l, count):
      "the letter may not come from the dictionary so may not exist"
      words_containing_l = self.words_containing_letter()
      if l in words_containing_l:
        return words_containing_l[l][count]
      else:
        return set()

    green = ''.join([l for i, l in guess_letters.green])
    yellow_green = guess_letters.yellow_letters + green
    def count_of_red_in_green_yellow(r):
      return yellow_green.count(r)

    # words must have has many matching letters that are green and yellow, and more
    # babab/rgryr must only include words with 2 or more a's
    ret = None
    green_and_yellow = [l for i,l in guess_letters.green + guess_letters.yellow]
    for l in set(green_and_yellow):
      letter_count = green_and_yellow.count(l) - 1
      words_containing_l = words_containing_letter(l, letter_count)
      ret = words_containing_l if ret == None else ret.intersection(words_containing_l)

    if ret == None:
      # no green or yellow need to start with all of the words
      ret = self.words

    # remove the words with red letters abcde/ryyyy means any word containing an 'a' can be removed
    # bbaaa/ryggg says you can remove words that have
    for r in set(guess_letters.red_letters):
      letter_count = count_of_red_in_green_yellow(r)
      ret = ret - words_containing_letter(r, letter_count)
    return ret

  def wordle(self, guesses: dict[str, str]) -> list[str]:
    "return a set of words"
    words = self
    for guess, answer in guesses.items():
      words = words.wordle1(guess, answer)
    return words.words

  def wordle1(self, guess, answer) -> set[str]:
    """key - guess, value - spot light: g - right letter right spot, y - right letter wrong spot, r - wrong letter
    abcde grrrr
    return - list or words in the dictionary that will work
    """
    GREEN = "g"
    YELLOW = "y"

    if len(guess) != len(answer):
      raise Exception(f"Guess and answer not same length, guess:{guess}, answer:{answer}")
    guess_letters = Letters(guess, answer)
    ret = set()
    #for word in self.words:
    for word in self.words_filter(guess, answer, guess_letters):
      for i,l in guess_letters.green:
        if l != word[i]:
          break # green must match exactly
      else:
        yellow_and_red = ""
        for i,l in guess_letters.yellow:
          if word[i] == l:
            break # it was yellow (not green) so it can not be in the suggested place
          #yellow_and_red = yellow_and_red + word[i]
        else:
          #for i,l in guess_letters.red:
          #  yellow_and_red = yellow_and_red + word[i]
          #(subset, red_letters) = subset_bool(guess_letters.yellow_letters, yellow_and_red)
          #(subset, red_letters) = (True, yellow_and_red)
          #if not subset:
          #  continue # all of the yellow guess letters must be in the word
          #if False and intersect(red_letters, guess_letters.red_letters):
          #  continue # word contains red letters known not to be in the answer
          ret.add(word)
    return Words(ret)

  def enter_guess(self, solution, guess):
    """
    This is a wordle server playing a game wth the hidden solution.  The user types in the guess and this function return
    a wordle answer.  The answer is checked against the words to return a new word subset

    key - guess, value - traffic light colors: g - right letter right spot, y - right letter wrong spot, r - wrong letter
    abcde grrrr
    """
    answer = wordle_answer(solution, guess)
    words = self.wordle({guess: answer})
    return Words(words)


def subset_bool(left: str, right_in: str):
  """Return true if the left is completely contained in right"""
  right = str(right_in)
  for l in left:
    if (i := right.find(l)) >= 0:
      right = right[:i] + right[i+1:]
    else:
      return (False, "") # left item not in right 
  return (True, right)

#def subset(left: str, right_in: str) -> bool:
#  """Return true if the left is completely contained in right"""
#  (subset_bool, subset_contents) = subset_bool(left, right)
#  return subset_bool

def intersect(left, right):
  """Return true if any elements of left are in right"""
  for l in left:
    if right.find(l) >= 0:
      return True
  return False


def popular_letters(word_sets):
  for name, words in word_sets.items():
    print(f"set: {name}")
    for l in range(5,7):
      print(f"words with most popular letters: {l}")
      letters = words.popular_letters()[0:l]
      wds = words.find(letters)
      if len(wds) > 0:
        for word in wds:
          print(word)

class GuessReport:
  def __init__(self, guess, words):
    self.guess = guess
    self.words = words
    self.solutions = dict()
    for solution in self.words.words:
      self.solutions[solution] = words.enter_guess(solution, guess)
  def average(self):
    sum = 0
    for solution, words in self.solutions.items():
      l = len(words.words)
      #print(f" {l}", end="")
      sum = sum + l
    #print("")
    return sum / len(self.solutions)
  def __str__(self):
    return f"guess:{self.guess} avg:{self.average()}"

def create_words_json(dir_s):
  """Create words.json from an index.html file that was creeated from:
  wget -v https://7esl.com/5-letter-words/"""
  dir = pathlib.Path(dir_s)
  word_list = list()
  with (dir / "index.html").open("r") as f:
    soup = BeautifulSoup(f, 'html.parser')
    for h in soup.findAll("h4"):
      if str(h.text).startswith("5 Letter"):  
        ul = h.find_next_sibling("ul")
        for li in ul.findAll("li"):
          word_list.append(str(li.text))
    with (dir / "words.json").open("w") as f:
      f.write(json.dumps(word_list))

def create_wordle_json(dir_s, file_s):
  """
  Create wordle.json from a previously downloaded file
  """
  dir = pathlib.Path(dir_s)
  with (dir / file_s).open("r") as f:
    s = f.read()
    left, sep, right = s.partition("Ma=")
    words_json_str, sep, _unused = right.partition("]")
    word_list = json.loads(words_json_str +"]")
    with (dir / "wordle.json").open("w") as f:
      f.write(json.dumps(word_list))


def report(words:Words, word_list:list=None):
  """
  report on the words.
  rank quesses.  Try all the words as a guess.  For each solution get a wordl answer and resulting words
  word_list is a list of words to use as possible guesses, if not provided the word list to report on comes from words
  """
  word_reports = []
  #for guess in list(sorted(words.words)):
  for guess in word_list if word_list != None else words.word_list():
    guess_report = GuessReport(guess, words)
    #print(guess_report.average())
    word_reports.append(guess_report)
  ret = None
  sorted_list = sorted(word_reports, key=lambda gr: 0.0 - gr.average())
  for guess_report in sorted_list[-10:]:
    print(guess_report)
    ret = guess_report
  return ret

def report_choose_best_small_large(words: Words, word_set:set):
    guess_small = report(words)
    print("")
    guess_large = report(words, word_set)
    if (guess_small.average() - 0.1) < guess_large.average():
      guess = guess_small
    else:
      guess = guess_large
    return guess

def simulate(words: Words, solution:str, first_guess:str) -> list[dict]:
  """Simulate a game of wordle.
  words - dictionary
  solution - answer
  first_guess - first guess
  """
  guesses = {first_guess: wordle_answer(solution, first_guess)}
  guess_number = 1
  print(f"{guess_number}: {first_guess}")
  while True:
    guess_number = guess_number + 1
    wds = words.wordle(guesses)
    #for wd in wds:
    #  print(wd)
    if len(wds) == 1:
      print(f"final answer:{guess_number} - {wds}")
      break
    if len(wds) == 0:
      print(f"bug")
      break
    guess = report_choose_best_small_large(Words(set(wds)), words.words)
    answer = wordle_answer(solution, guess.guess)
    print(f"{guess_number}: {guess}/{answer}")
    guesses[guess.guess] = answer
  return guesses

def play_wordle(word_sets, narrow_search=True):
  for name, words in word_sets.items():
    print(f"set: {name}")
    guesses = {
      "arise": "yrryg",
      "moved": "rrryr",
    }
    guesses = {
      "arise": wordle_answer("caulk", "arise"),
    }
    guesses = {
      "arise": "rrrrg",
      "mould": "rgrry",
      "weber": "ryrrr",
    }
    guesses = {
      "arise": "rrrrg",
      "mould": "rgrry",
    }
    guesses = {
      "arise": "rrgyr",
      "stink": "grgrr",
    }
    guesses = {
      "raise": "rgyrr",
      "panic": "rgrgy",
    }
    guesses = {
      "raise": "yrrry",
      "outer": "grygg",
    }
    guesses = {
      "raise": "yrrrr",
      "count": "ryryy",
    }
    guesses = {
      "raise": "yrrrg",
      "prong": "rggrr",
      "vodka": "yyrrr",
      "trove": "ggggg",
    }
    guesses = {
      "raise": "rrrrg",
      "could": "ryryr",
      "globe": "rggyg",
      "bloke": "ggggg",
    }
    wds = words.wordle(guesses)
    #for wd in wds:
    #  print(wd)
    next_guess = report_choose_best_small_large(Words(set(wds)), words.word_list())
    print(f"next_guess: {next_guess.guess}")
    if len(wds) == 1:
      print("final answer")
    elif len(wds) == 0:
      print("not in dictionary")
  
dir = "/Users/pquiring/github.com/powellquiring/wordle"

def popular_report():
  """
guess:table avg:19.38576779026217
guess:earth avg:19.307116104868914
guess:clear avg:19.04119850187266
guess:plate avg:18.153558052434455
guess:scale avg:18.1123595505618
guess:store avg:17.883895131086142
guess:trial avg:17.790262172284645
guess:learn avg:17.741573033707866
guess:heart avg:17.677902621722847
guess:share avg:17.408239700374533
guess:aline avg:17.071161048689138
guess:trade avg:17.04119850187266
guess:alone avg:16.078651685393258
guess:raise avg:14.812734082397004
guess:alter avg:14.50561797752809
guess:arise avg:14.146067415730338
  """
  ...

def all_report():
  """
  guess:trade avg:80.58077089649198
guess:react avg:79.70160242529234
guess:trail avg:79.15764400173235
guess:atone avg:78.77652663490689
guess:crane avg:78.68644434820268
guess:least avg:77.96751840623647
guess:alone avg:77.16024252923343
guess:leant avg:77.08921611087051
guess:learn avg:76.72195755738414
guess:aisle avg:76.08964919878736
guess:stale avg:75.32828064097012
guess:trace avg:73.9458640103941
guess:crate avg:72.80684278908619
guess:alert avg:71.51017756604591
guess:slate avg:71.28150714595063
guess:stare avg:71.0467734950195
guess:snare avg:71.0155911650065
guess:later avg:70.02728453876136
guess:saner avg:70.01775660459073
guess:alter avg:69.82546556951061
guess:arose avg:65.75963620614985
guess:irate avg:63.49198787353833
guess:arise avg:63.46860112602858
guess:raise avg:60.7444781290602
"""
  ...

def get_all_words():
  allwords = Words()
  # allwords.read_file(dir, "allwords.json")
  allwords.read_file(dir, "wordle.json")
  return allwords

def report_all_words():
  allwords = get_all_words()
  report(allwords)

def report_popular_words():
  popular = Words()
  popular.read_file(dir, "popular.json")
  report(popular)

def perf1():
  report_popular_words()

def perf2():
  report_all_words()

def edit_and_run_in_debugger():
  # create_wordle_json(dir, "main.18637ca1.js")
  #report_all_words()
  #report_popular_words()
  #report takes a long time
  #simulate(get_all_words(), "robin", "arise")
  #simulate(get_all_words(), "robin", "arise")
  #simulate(get_all_words(), "caulk", "arise")
  #simulate(get_all_words(), "caulk", "alert")
  #simulate(get_all_words(), "dodge", "arise")
  #simulate(get_all_words(), "swill", "arise")
  #simulate(get_all_words(), "swill", "raise")
  #simulate(get_all_words(), "tacit", "spoil")
  #simulate(get_all_words(), "tacit", "raise")
  # word_sets = {"popular":popular, "allwords": get_all_words()}

  word_sets = {"allwords": get_all_words()}
  play_wordle(word_sets, True)
  print("done")

if __name__ == '__main__':
  edit_and_run_in_debugger()
