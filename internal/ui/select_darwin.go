//go:build darwin

package ui

import (
	"syscall"
	"unsafe"
)

func isTerminal(fd uintptr) bool {
	_, err := tcget(fd)
	return err == nil
}

func makeRaw(fd uintptr) (rawState, error) {
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

func restore(fd uintptr, state rawState) error {
	if state == nil {
		return nil
	}
	return tcset(fd, state.(*syscall.Termios))
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
