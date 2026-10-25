package training

import (
	"reflect"
	"testing"
)

func TestLevelClass(t *testing.T) {
	for in, want := range map[string]string{
		"E": LevelBeginner, "a": LevelBeginner, "Klasse A": LevelBeginner, "A*": LevelIntermediate,
		"L": LevelIntermediate, "L** Dressur": LevelAdvanced, "M": LevelAdvanced, "S***": LevelAdvanced,
		"Basis": LevelBeginner, "": "", "L bei Anna": LevelIntermediate, "Anna": "", "Kl.": "",
	} {
		if got := LevelClass(in); got != want {
			t.Errorf("LevelClass(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestExerciseLibraries(t *testing.T) {
	cases := []struct {
		a          Activity
		discipline string
		want       []string
	}{
		{ActivityHall, "dressage", []string{"dressage", "cavaletti"}},
		{ActivityHall, "", []string{"dressage", "cavaletti"}},
		{ActivityHall, "eventing", []string{"dressage", "cavaletti"}},
		{ActivityArena, "dressage", []string{"dressage", "cavaletti"}},
		{ActivityArena, "jumping", []string{"jumping", "cavaletti", "dressage"}},
		{ActivityHall, "jumping", []string{"jumping", "cavaletti", "dressage"}},
		{ActivityHall, "western", []string{"western", "cavaletti", "dressage"}},
		{ActivityArena, "western", []string{"western", "cavaletti", "dressage"}},
		{ActivityJumping, "leisure", []string{"jumping", "cavaletti"}},
		{ActivityJumping, "jumping", []string{"jumping", "cavaletti"}},
		{ActivityLunge, "dressage", []string{"lunge", "cavaletti"}},
		{ActivityGroundwork, "", []string{"groundwork"}},
		{ActivityHack, "dressage", []string{"hack"}},
		{ActivityHack, "western", []string{"hack"}},
		{ActivityWalker, "", nil},
		{ActivityRest, "", nil},
		{"swimming", "dressage", nil},
	}
	for _, c := range cases {
		got := ExerciseLibraries(c.a, c.discipline)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("ExerciseLibraries(%s, %q) = %v, want %v", c.a, c.discipline, got, c.want)
		}
	}
	// The result is a fresh slice: a caller may change it.
	a := ExerciseLibraries(ActivityHall, "dressage")
	a[0] = "x"
	if ExerciseLibraries(ActivityHall, "dressage")[0] != "dressage" {
		t.Error("ExerciseLibraries shares its result")
	}
}

func TestHomeLibraries(t *testing.T) {
	for in, want := range map[string][]string{
		"dressage": {"dressage"}, "": {"dressage"}, "leisure": {"dressage"}, "eventing": {"dressage"},
		"jumping": {"jumping", "dressage"}, "western": {"western"}, "young_horse": {"dressage"},
	} {
		if got := HomeLibraries(in); !reflect.DeepEqual(got, want) {
			t.Errorf("HomeLibraries(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestSessionUsedExercises(t *testing.T) {
	s := Session{ExerciseID: "a", Exercises: []string{"b", "a", "", "c"}}
	if got := s.UsedExercises(); !reflect.DeepEqual(got, []string{"a", "b", "c"}) {
		t.Errorf("UsedExercises = %v", got)
	}
	if got := (Session{}).UsedExercises(); len(got) != 0 {
		t.Errorf("empty session uses %v", got)
	}
}
