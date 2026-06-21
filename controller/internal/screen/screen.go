package screen

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
)

type Screen struct {
	screen tcell.Screen
}

func NewScreen() (*Screen, error) {
	scr, err := tcell.NewScreen()
	if err != nil {
		return nil, err
	}
	if err := scr.Init(); err != nil {
		return nil, err
	}
	scr.SetStyle(tcell.StyleDefault)
	scr.Clear()
	scr.HideCursor()

	return &Screen{screen: scr}, nil
}

func (s *Screen) Finalize() {
	s.screen.Fini()
}

// KeyEvent is a minimal, tcell-free representation of a keypress so callers
// outside this package don't need to import tcell.
type KeyEvent struct {
	Rune rune
	Quit bool // Ctrl+C or Ctrl+D was pressed
}

// NextKey blocks until the next keyboard event and returns it. This must be
// the ONLY place that reads input. Never read os.Stdin directly elsewhere
// while tcell owns the terminal — the two readers will race for bytes,
// which is what was causing stray/garbled characters to show up.
func (s *Screen) NextKey() KeyEvent {
	for {
		switch ev := s.screen.PollEvent().(type) {
		case *tcell.EventKey:
			switch ev.Key() {
			case tcell.KeyCtrlC, tcell.KeyCtrlD:
				return KeyEvent{Quit: true}
			case tcell.KeyRune:
				return KeyEvent{Rune: ev.Rune()}
			}
		case *tcell.EventResize:
			// Keeps the layout correct/fixed if the terminal is resized.
			s.screen.Sync()
		}
	}
}

func (s *Screen) Refresh(roll int, pitch int, yaw int, throttle int, arm bool) {
	s.drawScreen(roll, pitch, yaw, throttle, arm)
}

func (s *Screen) drawScreen(
	roll int,
	pitch int,
	yaw int,
	throttle int,
	arm bool,
) {
	s.screen.Clear()

	lines := []struct {
		text string
		col  tcell.Color
	}{
		{"╔══════════════════════════════════════════════════════════════╗", tcell.ColorYellow},
		{"║                      DRONE CONTROLLER                        ║", tcell.ColorYellow},
		{"╚══════════════════════════════════════════════════════════════╝", tcell.ColorYellow},
		{"", tcell.ColorWhite},
		{"  COMMANDS                                                      ", tcell.ColorWhite},
		{"  [E] Emergency stop.                                           ", tcell.ColorRed},
		{"", tcell.ColorWhite},
		{"  REALTIME MOVEMENT  (hold key)                                 ", tcell.ColorWhite},
		{"  [I/K]  Forward / Backward  (Roll axis)                        ", tcell.ColorGreen},
		{"  [J/L]  Yaw left / right    (Pitch axis)                       ", tcell.ColorGreen},
		// {"  [Q/E]  Strafe left / right (Throttle axis)", tcell.ColorGreen}, //! TODO: Add strafe support to the drone and then re-enable this line.
		{"  [Q/A]  Up / Down           (Yaw axis)                         ", tcell.ColorGreen},
		{"", tcell.ColorWhite},
		{"  [Ctrl+C / Ctrl+D] Quit                                        ", tcell.ColorGray},
		{"", tcell.ColorWhite},
		{"────────────────────────────────────────────────────────────────", tcell.ColorDarkGray},
		{" DRONE VALUES                                                   ", tcell.ColorWhite},
		{"", tcell.ColorWhite},
		{fmt.Sprintf(" Roll: %+4d Pitch: %+4d Throttle: %+4d Yaw: %+4d Arm: %t", roll, pitch, throttle, yaw, arm), tcell.ColorLightCyan},
		{"────────────────────────────────────────────────────────────────", tcell.ColorDarkGray},
	}

	for row, line := range lines {
		s.drawText(2, row+1, line.text, line.col)
	}

	s.screen.Show()
}

func (s *Screen) drawText(x, y int, text string, color tcell.Color) {
	style := tcell.StyleDefault.Foreground(color)
	col := x
	for _, ch := range text {
		s.screen.SetContent(col, y, ch, nil, style)
		col++
	}
}
