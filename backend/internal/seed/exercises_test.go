package seed

import (
	"strings"
	"testing"
)

// The library is inserted in slice order and next_exercise_id is a foreign key, so every
// follow-up must come first; every exercise must name the pages it is based on.
func TestExerciseLibrary(t *testing.T) {
	levels := map[string]bool{"beginner": true, "intermediate": true, "advanced": true}
	disciplines := map[string]bool{"dressage": true, "jumping": true, "groundwork": true, "lunge": true}
	seen := map[int]string{}
	for _, e := range exerciseLibrary {
		if e.n < 1 || e.n > 99 {
			t.Errorf("%s: n %d outside 1-99", e.title, e.n)
		}
		if other, dup := seen[e.n]; dup {
			t.Errorf("%s: n %d already used by %s", e.title, e.n, other)
		}
		if !disciplines[e.discipline] || !levels[e.level] {
			t.Errorf("%s: discipline %q, level %q", e.title, e.discipline, e.level)
		}
		if e.next != 0 {
			next, ok := seen[e.next]
			if !ok {
				t.Errorf("%s: follow-up %d is not inserted before it", e.title, e.next)
			} else if nd := disciplineOf(e.next); nd != e.discipline {
				t.Errorf("%s: follow-up %s is %s", e.title, next, nd)
			}
		}
		if len(e.steps) < 3 || len(e.tags) == 0 {
			t.Errorf("%s: %d steps, %d tags", e.title, len(e.steps), len(e.tags))
		}
		for _, tag := range e.tags {
			if tag != strings.ToLower(tag) || strings.ContainsFunc(tag, func(r rune) bool { return r > 'z' || r < 'a' }) {
				t.Errorf("%s: tag %q is not a lower-case ASCII key", e.title, tag)
			}
		}
		if len(e.sources) == 0 {
			t.Errorf("%s: no source", e.title)
		}
		for _, s := range e.sources {
			if !strings.HasPrefix(s, "https://") {
				t.Errorf("%s: source %q", e.title, s)
			}
		}
		seen[e.n] = e.title
	}
}

func disciplineOf(n int) string {
	for _, e := range exerciseLibrary {
		if e.n == n {
			return e.discipline
		}
	}
	return ""
}
