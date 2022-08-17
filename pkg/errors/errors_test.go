package errors

import (
	"errors"
	"fmt"
	"io"
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		err  string
		want error
	}{
		{"", fmt.Errorf("")},
		{"foo", fmt.Errorf("foo")},
		{"foo", New("foo")},
		{"with format %v", errors.New("with format %v")},
	}
	for _, tt := range tests {

		got := New(tt.err)
		if got.Error() != tt.want.Error() {
			t.Errorf("New.Error(): got: %q, want %q", got, tt.want)
		}

	}
}

func TestWrap(t *testing.T) {
	testCases := []struct {
		err     error
		message string
		want    string
	}{

		{io.EOF, "read error", "read error"},
		{Wrap(io.EOF, "read error"), "client error", "client error"},
	}
	for _, tt := range testCases {
		got := Wrap(tt.err, tt.message).Error()
		if got != tt.want {
			t.Errorf("Wrap(%v, %q): got: %v, want %v", tt.err, tt.message, got, tt.want)
		}
	}

	t.Run("WrapNik", func(t *testing.T) {
		got := Wrap(nil, "hello")
		if got != nil {
			t.Errorf("want nil but got %#v", got)
		}
	})
}
func TestWrapfNil(t *testing.T) {
	got := Wrapf(nil, "no error")
	if got != nil {
		t.Errorf("Wrapf(nil, \"no error\"): got %#v, expected nil", got)
	}
}

func TestWrapf(t *testing.T) {
	tests := []struct {
		err     error
		message string
		want    string
	}{
		{io.EOF, "read error", "read error"},
		{Wrapf(io.EOF, "read error without format specifiers"), "client error", "client error"},
		{Wrapf(io.EOF, "read error with %d format specifier", 1), "client error", "client error"},
	}

	for _, tt := range tests {
		got := Wrapf(tt.err, tt.message).Error()
		if got != tt.want {
			t.Errorf("Wrapf(%v, %q): got: %v, want %v", tt.err, tt.message, got, tt.want)
		}
	}
}
