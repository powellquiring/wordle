import w

def test_long():
  allwords = w.Words()
  allwords.read_file(w.dir, "allwords.json")
  guesses = w.simulate(allwords, "caulk", "arise")
  assert guesses["caulk"] == "ggggg"
  assert len(guesses) == 4

def test_two():
  (words := w.Words()).add("aabbb")
  words.add("aaccc")
  possible = words.wordle({"axbcxx": "grgrrr"})
  assert len(possible) == 1

def test_one():
  (words := w.Words()).add("input")
  possible = words.wordle({"alert": "rrrrg"})
  assert len(possible) == 1

  (words := w.Words()).add("range")
  possible = words.wordle({"alter": "rrryr"})
  assert len(possible) == 0

  (words := w.Words()).add("alert")
  possible = words.wordle({"alter": "rrryr"})
  assert len(possible) == 0

  (words := w.Words()).add("abcde")
  possible = words.wordle({"xxxxx": "nnnnn"})
  assert len(possible) == 1

  (words := w.Words()).add("abcde")
  possible = words.wordle({"axxxx": "gnnnn"})
  assert len(possible) == 1

  (words := w.Words()).add("abcde")
  possible = words.wordle({"axxxx": "ynnnn"})
  assert len(possible) == 0


