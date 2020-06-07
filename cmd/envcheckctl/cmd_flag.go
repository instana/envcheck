package main

import (
	"flag"
	"fmt"
	"io"
)

var (
	// ErrNoSubcommand occurs when too few arguments are supplied to the executable.
	ErrNoSubcommand = fmt.Errorf("no sub-command specified")
	// ErrUnknownSubcommand occurs when an unknown sub-command is specified.
	ErrUnknownSubcommand = fmt.Errorf("invalid sub-command specified")
)

// New creates a new CmdFlag for capturing sub-command flags and configurations.
func New(w io.Writer) *CmdFlag {
	return &CmdFlag{
		flagSets: make(map[string]*flag.FlagSet),
		configs:  make(map[string]*EnvcheckConfig),
		w:        w,
	}
}

// CmdFlag is a struct to capture an number of subcommands, config, and their flags.
type CmdFlag struct {
	flagSets map[string]*flag.FlagSet
	configs  map[string]*EnvcheckConfig
	w        io.Writer
}

// FlagSet creates a new flagset with the name and associated subCmd enum.
func (cf *CmdFlag) FlagSet(name string, subCmd int) (*flag.FlagSet, *EnvcheckConfig) {
	f := flag.NewFlagSet(name, flag.ContinueOnError)
	cfg := &EnvcheckConfig{Subcommand: subCmd}
	cf.flagSets[name] = f
	cf.configs[name] = cfg
	f.SetOutput(cf.w)
	return f, cfg
}

// Usage prints the usage for all commands.
func (cf *CmdFlag) Usage(cmd string) {
	cf.w.Write([]byte("Usage: " + cmd + " requires a subcommand (rev. " + Revision + ")\n"))
	for _, v := range cf.flagSets {
		cf.w.Write([]byte("\n"))
		v.Usage()
	}
	cf.w.Write([]byte("\n"))
}

// Parse extracts the relevant flag values for the appropriate sub-command.
func (cf *CmdFlag) Parse(args []string) (*EnvcheckConfig, error) {
	cmd := args[0]
	if len(args) < 2 {
		cf.Usage(cmd)
		return nil, ErrNoSubcommand
	}

	subCmd := args[1]
	p, ok := cf.flagSets[subCmd]
	if !ok {
		cf.Usage(cmd)
		return nil, ErrUnknownSubcommand
	}

	err := p.Parse(args[2:])
	return cf.configs[subCmd], err
}
