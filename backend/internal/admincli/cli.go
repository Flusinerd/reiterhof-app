// Package admincli implements the operator command line tool stallfunk-admin
// (cmd/admin): creating the first stable and admins, invite codes, user and horse
// maintenance, a test mail, a weather refresh and migration status.
//
// Every command is a function of an Env (database pool, config, reader and writers), so
// tests call Run directly with a seeded database and buffers instead of executing a binary.
// The commands reuse the packages of the API (privacy.DeleteAccount, auth.CreateInvite,
// auth.NewMailer, weather.Service, db.Migrate) so that they behave exactly like the API.
package admincli

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Flusinerd/reiterhof-app/backend/internal/auth"
	"github.com/Flusinerd/reiterhof-app/backend/internal/config"
	"github.com/Flusinerd/reiterhof-app/backend/internal/httpx"
	"github.com/Flusinerd/reiterhof-app/backend/internal/weather"
)

// Env is everything a command needs. Only Out is required; the rest has defaults.
type Env struct {
	// In is read for confirmation prompts; Out receives results, Err messages and hints.
	In       io.Reader
	Out, Err io.Writer

	// Pool is the database. When nil, Open is called on first use (commands like
	// "mail test" and "help" never connect).
	Pool *pgxpool.Pool
	Open func(ctx context.Context) (*pgxpool.Pool, error)

	Config config.Config
	Now    func() time.Time
	Log    *slog.Logger

	// Migrations is the embedded migration set (migrations.FS).
	Migrations fs.FS

	// Overrides for tests; nil means the real implementation.
	Mailer  auth.Mailer
	Fetcher weather.Fetcher
	Notify  httpx.Notifier

	// Set by the global flags; every subcommand also accepts them.
	JSON bool
	Yes  bool

	stdin    *bufio.Reader
	stdinSrc io.Reader // the In that stdin wraps
}

// ErrUsage marks wrong arguments; the caller prints the message and exits with status 2.
var ErrUsage = errors.New("usage error")

// ErrAborted is returned when the operator declines a confirmation prompt.
var ErrAborted = errors.New("aborted")

type usageError string

func (u usageError) Error() string        { return string(u) }
func (u usageError) Is(target error) bool { return target == ErrUsage }

func usagef(format string, a ...any) error {
	return usageError(fmt.Sprintf(format, a...))
}

func (e *Env) now() time.Time {
	if e.Now != nil {
		return e.Now()
	}
	return time.Now()
}

func (e *Env) log() *slog.Logger {
	if e.Log != nil {
		return e.Log
	}
	return slog.New(slog.NewTextHandler(e.Err, nil))
}

func (e *Env) pool(ctx context.Context) (*pgxpool.Pool, error) {
	if e.Pool != nil {
		return e.Pool, nil
	}
	if e.Open == nil {
		return nil, errors.New("no database configured")
	}
	p, err := e.Open(ctx)
	if err != nil {
		return nil, err
	}
	e.Pool = p
	return p, nil
}

// confirm asks the operator to type y/yes unless --yes was given.
func (e *Env) confirm(prompt string) error {
	if e.Yes {
		return nil
	}
	if e.stdin == nil || e.stdinSrc != e.In {
		e.stdinSrc = e.In
		in := e.In
		if in == nil {
			in = strings.NewReader("")
		}
		e.stdin = bufio.NewReader(in)
	}
	fmt.Fprintf(e.Err, "%s [y/N]: ", prompt)
	line, err := e.stdin.ReadString('\n')
	answer := strings.ToLower(strings.TrimSpace(line))
	if answer == "y" || answer == "yes" {
		return nil
	}
	if err != nil && answer == "" {
		fmt.Fprintln(e.Err, "\nno confirmation given (use --yes to skip the prompt)")
	}
	return ErrAborted
}

// flags creates a flag set that also accepts the global --json and --yes flags.
func (e *Env) flags(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	// Errors are returned to the caller; only the usage (on -h and on bad flags) is printed.
	fs.SetOutput(io.Discard)
	fs.Usage = func() {
		fmt.Fprintf(e.Err, "Usage: stallfunk-admin %s [flags]\n", name)
		fs.SetOutput(e.Err)
		fs.PrintDefaults()
		fs.SetOutput(io.Discard)
	}
	fs.BoolVar(&e.JSON, "json", e.JSON, "print machine-readable JSON (list commands)")
	fs.BoolVar(&e.Yes, "yes", e.Yes, "do not ask for confirmation")
	return fs
}

// errHelp is returned by parse after -h/--help; Run turns it into success.
var errHelp = errors.New("help requested")

// parse parses flags that may appear before, between or after the positional arguments
// (the standard flag package stops at the first positional). It requires between min
// and max positionals.
func parse(fs *flag.FlagSet, args []string, min, max int) ([]string, error) {
	var pos []string
	for {
		if err := fs.Parse(args); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return nil, errHelp
			}
			return nil, usagef("%v", err)
		}
		if fs.NArg() == 0 {
			break
		}
		pos = append(pos, fs.Arg(0))
		args = fs.Args()[1:]
	}
	if len(pos) < min || len(pos) > max {
		if max == 0 {
			return nil, usagef("%s takes no arguments (got %q)", fs.Name(), pos)
		}
		return nil, usagef("%s: expected %d..%d arguments, got %d", fs.Name(), min, max, len(pos))
	}
	return pos, nil
}

// setFlags returns the names of the flags that were given explicitly.
func setFlags(fs *flag.FlagSet) map[string]bool {
	set := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { set[f.Name] = true })
	return set
}

type command struct {
	usage string
	run   func(ctx context.Context, e *Env, args []string) error
}

// commands maps "group" -> "action" -> command.
func commands() map[string]map[string]command {
	return map[string]map[string]command{
		"stable": {
			"list":   {"stable list", runStableList},
			"create": {"stable create --name NAME [--farm F] [--city C] [--lat N --lng N] [--timezone TZ] [--reminder-time HH:MM] [--geofence-radius M]", runStableCreate},
			"update": {"stable update ID [--name ...] (same flags as create)", runStableUpdate},
		},
		"user": {
			"list":    {"user list [--stable ID]", runUserList},
			"create":  {"user create --email E --name N [--stable ID] [--admin]", runUserCreate},
			"promote": {"user promote EMAIL|ID", runUserPromote},
			"demote":  {"user demote EMAIL|ID [--yes]", runUserDemote},
			"move":    {"user move EMAIL|ID [--stable ID] [--yes]", runUserMove},
			"delete":  {"user delete EMAIL|ID [--yes]", runUserDelete},
		},
		"invite": {
			"create": {"invite create [--stable ID] [--days 7] [--max-uses 10]", runInviteCreate},
			"list":   {"invite list [--stable ID]", runInviteList},
		},
		"horse": {
			"list":     {"horse list [--stable ID]", runHorseList},
			"transfer": {"horse transfer ID|NAME --to EMAIL|ID [--stable ID] [--yes]", runHorseTransfer},
		},
		"sessions": {
			"revoke": {"sessions revoke EMAIL|ID", runSessionsRevoke},
		},
		"mail": {
			"test": {"mail test --to ADDRESS", runMailTest},
		},
		"weather": {
			"refresh": {"weather refresh", runWeatherRefresh},
		},
		"migrate": {
			"status": {"migrate status", runMigrateStatus},
			"up":     {"migrate up", runMigrateUp},
		},
	}
}

// Usage prints the command overview.
func Usage(w io.Writer) {
	fmt.Fprint(w, `stallfunk-admin: operator tool for the Stallfunk backend

Usage: stallfunk-admin [--env-file FILE] [--json] [--yes] COMMAND [ARGS]

Configuration comes from the environment (REITERHOF_DATABASE_URL, REITERHOF_SMTP_*, ...).
--env-file reads a systemd EnvironmentFile (default /etc/reiterhof/api.env if it exists);
variables that are already set in the environment win.

Commands:
`)
	cmds := commands()
	groups := make([]string, 0, len(cmds))
	for g := range cmds {
		groups = append(groups, g)
	}
	sort.Strings(groups)
	for _, g := range groups {
		actions := make([]string, 0, len(cmds[g]))
		for a := range cmds[g] {
			actions = append(actions, a)
		}
		sort.Strings(actions)
		for _, a := range actions {
			fmt.Fprintf(w, "  %s\n", cmds[g][a].usage)
		}
	}
	fmt.Fprint(w, `
Global flags (also accepted after the command):
  --json   machine-readable output for list commands
  --yes    skip the confirmation of destructive commands (user delete/demote/move, horse transfer)

With exactly one stable in the database, --stable defaults to it; with several it must be given
(the list commands then show all stables). ID is the UUID, see the list commands.
`)
}

// Run executes one command line (without the program name and the --env-file flag,
// which the caller consumed to set up the environment). The returned error is already
// worded for the operator; errors.Is(err, ErrUsage) means a wrong command line.
func Run(ctx context.Context, e *Env, args []string) error {
	if e.Out == nil || e.Err == nil {
		return errors.New("admincli: Env needs Out and Err")
	}
	// Global flags before the command word.
	gfs := e.flags("stallfunk-admin")
	gfs.Usage = func() { Usage(e.Err) }
	if err := gfs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return usagef("%v", err)
	}
	args = gfs.Args()
	if len(args) == 0 || args[0] == "help" {
		Usage(e.Out)
		return nil
	}
	group, ok := commands()[args[0]]
	if !ok {
		return usagef("unknown command %q (see: stallfunk-admin help)", args[0])
	}
	if len(args) < 2 || strings.HasPrefix(args[1], "-") {
		return usagef("%s needs a subcommand: %s", args[0], strings.Join(sortedKeys(group), ", "))
	}
	cmd, ok := group[args[1]]
	if !ok {
		return usagef("unknown subcommand %q for %s (have: %s)", args[1], args[0], strings.Join(sortedKeys(group), ", "))
	}
	err := cmd.run(ctx, e, args[2:])
	if errors.Is(err, errHelp) {
		return nil
	}
	return err
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// --- output helpers ---------------------------------------------------------------

func writeTable(w io.Writer, header []string, rows [][]string) {
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, strings.Join(header, "\t"))
	for _, r := range rows {
		fmt.Fprintln(tw, strings.Join(r, "\t"))
	}
	_ = tw.Flush()
}

func fmtTime(t time.Time) string { return t.UTC().Format("2006-01-02 15:04") }

func fmtTimePtr(t *time.Time) string {
	if t == nil {
		return "-"
	}
	return fmtTime(*t)
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
