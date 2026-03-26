package ui

import (
	"bufio"
	"fmt"
	"io"
	"os"
)

type rawState any

func Select(label string, items []string, out io.Writer) (int, error) {
	if len(items) == 0 {
		return -1, fmt.Errorf("no items to select")
	}

	if !isTerminal(os.Stdin.Fd()) {
		return fallbackSelect(label, items, os.Stdin, out)
	}

	oldState, err := makeRaw(os.Stdin.Fd())
	if err != nil {
		return fallbackSelect(label, items, os.Stdin, out)
	}
	defer restore(os.Stdin.Fd(), oldState)

	selected := 0
	reader := bufio.NewReader(os.Stdin)

	for {
		render(label, items, selected, out)

		b, err := reader.ReadByte()
		if err != nil {
			return -1, err
		}

		switch b {
		case 3:
			return -1, fmt.Errorf("selection cancelled")
		case 13, 10:
			fmt.Fprintln(out)
			return selected, nil
		case 'k':
			if selected > 0 {
				selected--
			}
		case 'j':
			if selected < len(items)-1 {
				selected++
			}
		case 27:
			next, _ := reader.ReadByte()
			if next != '[' {
				continue
			}
			arrow, _ := reader.ReadByte()
			switch arrow {
			case 'A':
				if selected > 0 {
					selected--
				}
			case 'B':
				if selected < len(items)-1 {
					selected++
				}
			}
		}
	}
}

func render(label string, items []string, selected int, out io.Writer) {
	writeCRLF(out, "\033[H\033[2J")
	writeCRLF(out, label)
	writeCRLF(out, "")
	for i, item := range items {
		prefix := "  "
		if i == selected {
			prefix = "> "
		}
		writeCRLF(out, prefix+item)
	}
	writeCRLF(out, "")
	writeCRLF(out, "Use up/down (or j/k), press Enter to confirm.")
}

func fallbackSelect(label string, items []string, in io.Reader, out io.Writer) (int, error) {
	fmt.Fprintln(out, label)
	for i, item := range items {
		fmt.Fprintf(out, "%d. %s\n", i+1, item)
	}
	fmt.Fprint(out, "> ")

	var choice int
	if _, err := fmt.Fscanln(in, &choice); err != nil {
		return -1, err
	}
	if choice < 1 || choice > len(items) {
		return -1, fmt.Errorf("invalid choice")
	}
	return choice - 1, nil
}

func writeCRLF(out io.Writer, text string) {
	fmt.Fprint(out, text, "\r\n")
}
