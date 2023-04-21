package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
)

type options struct {
	token       string
	model       gptModel
	prompt      string
	interactive bool
}

func hasPipedInput() bool {
	if stat, err := os.Stdin.Stat(); err == nil {
		return (stat.Mode() & os.ModeCharDevice) == 0
	}
	return false
}

func main() {
	opts := options{
		model:       gpt3,
		prompt:      "bash",
		interactive: false,
	}

	flag.StringVar(&opts.prompt, "prompt", "", "Ask for help about topic (bash, php...)")
	flag.StringVar(&opts.prompt, "p", "", "Ask for help about topic (bash, php...)")

	flag.BoolVar(&opts.interactive, "interactive", false, "Start in interactive mode right away")
	flag.BoolVar(&opts.interactive, "i", false, "Start in interactive mode right away")

	var init bool
	flag.BoolVar(&init, "init", false, "Initialize configuration")

	flag.Parse()

	if init {
		if err := initializeConfig(); err != nil {
			panic(err)
		}
		os.Exit(0)
	} else if !hasConfigFile() {
		fmt.Println("Unable to find config file, please run with --init flag")
		os.Exit(1)
	}

	cfg := loadConfig()
	if cfg.Token == "" {
		path, _ := getConfigFilepath()
		fmt.Printf("Please configure your OpenAI token in %s\n", path)
		os.Exit(1)
	}
	opts.token = cfg.Token
	if cfg.Model != "" {
		opts.model = cfg.Model
	}

	var convo conversation
	if opts.prompt != "" {
		convo = conversation{
			message{
				Role:    roleSystem,
				Content: fmt.Sprintf("You are a helpful assistant that helps with %s.", opts.prompt),
			},
		}
	} else {
		convo = conversation{}
	}

	var questionBuilder strings.Builder
	questionBuilder.WriteString(
		strings.TrimSpace(strings.Join(flag.Args(), " ")))
	if hasPipedInput() {
		questionBuilder.WriteString("\n")
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			questionBuilder.WriteString(scanner.Text())
			questionBuilder.WriteString("\n")
		}
	}

	question := questionBuilder.String()
	if question == "" {
		opts.interactive = true
	} else {
		var err error
		convo, err = convo.Ask(question, opts)
		if err != nil {
			panic(err)
		}

		if !opts.interactive {
			code := convo.ParseCode()
			if len(code) > 1 {
				opts.interactive = true
			} else if !opts.interactive {
				// TODO: should be pushed to command line, or selection, or execution
				if len(code) != 0 {
					fmt.Println(code[0])
				} else {
					fmt.Println(convo.Last())
				}
			}
		}
	}

	if opts.interactive {
		chat(opts, convo)
	}
}
