package main

// Mode describes an output style. Cols/Rows are the number of sub-pixels
// packed inside a single terminal character cell.
type Mode struct {
	Name string
	Desc string
	Cols int
	Rows int
	Kind string // ramp | half | quad | braille
	Ramp string
}

const asciiRamp = " .'`^\",:;Il!i><~+_-?][}{1)(|\\/tfjrxnuvczXYUJCLQ0OZmwqpdbkhao*#MW&8%B@$"
const shortRamp = " .:-=+*#%@"
const shadeRamp = " ░▒▓█"

var modes = []Mode{
	{Name: "ascii", Desc: "clásico, 1 carácter por celda", Cols: 1, Rows: 1, Kind: "ramp", Ramp: asciiRamp},
	{Name: "short", Desc: "ASCII de 10 niveles (más limpio)", Cols: 1, Rows: 1, Kind: "ramp", Ramp: shortRamp},
	{Name: "shade", Desc: "bloques de sombra ░▒▓█", Cols: 1, Rows: 1, Kind: "ramp", Ramp: shadeRamp},
	{Name: "blocks", Desc: "medio bloque ▀▄█ con color real", Cols: 1, Rows: 2, Kind: "half"},
	{Name: "quad", Desc: "cuadrantes ▖▗▘▝ 2x2", Cols: 2, Rows: 2, Kind: "quad"},
	{Name: "braille", Desc: "braille ⠿ 2x4, máxima densidad", Cols: 2, Rows: 4, Kind: "braille"},
}

func modeByName(name string) Mode {
	for _, m := range modes {
		if m.Name == name {
			return m
		}
	}
	return modes[0]
}

func modeNames() []string {
	out := make([]string, len(modes))
	for i, m := range modes {
		out[i] = m.Name
	}
	return out
}

// quadChars maps a 4-bit mask (TL=1, TR=2, BL=4, BR=8) to a quadrant block.
var quadChars = [16]rune{
	' ', '▘', '▝', '▀',
	'▖', '▌', '▞', '▛',
	'▗', '▚', '▐', '▜',
	'▄', '▙', '▟', '█',
}

// braille bit for a dot at (row 0..3, col 0..1).
var brailleBits = [4][2]uint8{
	{0x01, 0x08}, // dot 1, dot 4
	{0x02, 0x10}, // dot 2, dot 5
	{0x04, 0x20}, // dot 3, dot 6
	{0x40, 0x80}, // dot 7, dot 8
}
