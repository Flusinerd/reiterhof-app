package training

import "testing"

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

func TestExerciseLibrary(t *testing.T) {
	cases := []struct {
		a          Activity
		discipline string
		want       string
	}{
		{ActivityHall, "dressage", "dressage"}, {ActivityArena, "jumping", "jumping"}, {ActivityJumping, "leisure", "jumping"},
		{ActivityLunge, "dressage", "lunge"}, {ActivityGroundwork, "", "groundwork"}, {ActivityHack, "dressage", ""},
		{ActivityWalker, "", ""}, {ActivityRest, "", ""},
	}
	for _, c := range cases {
		if got := ExerciseLibrary(c.a, c.discipline); got != c.want {
			t.Errorf("ExerciseLibrary(%s, %s) = %q, want %q", c.a, c.discipline, got, c.want)
		}
	}
}
