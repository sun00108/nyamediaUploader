package ui

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"syscall"
	"unsafe"
)

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

func isTerminal(fd uintptr) bool {
	_, err := tcget(fd)
	return err == nil
}

func makeRaw(fd uintptr) (*syscall.Termios, error) {
	state, err := tcget(fd)
	if err != nil {
		return nil, err
	}

	raw := *state
	raw.Iflag &^= syscall.IGNBRK | syscall.BRKINT | syscall.PARMRK | syscall.ISTRIP | syscall.INLCR | syscall.IGNCR | syscall.ICRNL | syscall.IXON
	raw.Oflag &^= syscall.OPOST
	raw.Lflag &^= syscall.ECHO | syscall.ECHONL | syscall.ICANON | syscall.ISIG | syscall.IEXTEN
	raw.Cflag &^= syscall.CSIZE | syscall.PARENB
	raw.Cflag |= syscall.CS8
	raw.Cc[syscall.VMIN] = 1
	raw.Cc[syscall.VTIME] = 0

	if err := tcset(fd, &raw); err != nil {
		return nil, err
	}
	return state, nil
}

func restore(fd uintptr, state *syscall.Termios) error {
	return tcset(fd, state)
}

func tcget(fd uintptr) (*syscall.Termios, error) {
	var value syscall.Termios
	_, _, errno := syscall.Syscall6(syscall.SYS_IOCTL, fd, uintptr(syscall.TIOCGETA), uintptr(unsafe.Pointer(&value)), 0, 0, 0)
	if errno != 0 {
		return nil, errno
	}
	return &value, nil
}

func tcset(fd uintptr, value *syscall.Termios) error {
	_, _, errno := syscall.Syscall6(syscall.SYS_IOCTL, fd, uintptr(syscall.TIOCSETA), uintptr(unsafe.Pointer(value)), 0, 0, 0)
	if errno != 0 {
		return errno
	}
	return nil
}
