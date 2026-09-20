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
	// CellAspect is the terminal cell width/height ratio (0 means 0.5).
	CellAspect float64
	// ImgAspect is the glyph advance / line-height ratio used when exporting
	// to an image (0 means 0.5).
	ImgAspect float64
	// Font is the preferred export font for this style ("" keeps the default).
	Font string
}

func (m Mode) cellAspect() float64 {
	if m.CellAspect > 0 {
		return m.CellAspect
	}
	return 0.5
}

func (m Mode) imgAspect() float64 {
	if m.ImgAspect > 0 {
		return m.ImgAspect
	}
	return m.cellAspect()
}

const asciiRamp = " .'`^\",:;Il!i><~+_-?][}{1)(|\\/tfjrxnuvczXYUJCLQ0OZmwqpdbkhao*#MW&8%B@$"
const shortRamp = " .:-=+*#%@"
const shadeRamp = " ░▒▓█"

// Ideographic (full-width) space starts the CJK ramps so the grid keeps its
// width even in the darkest cells.
const chineseRamp = "\u3000·丶丿一乙十人八力口土大女子山工小川己心手文日月中木水火田目石米耳肉自舌舟虫行衣言贝走足车金门雨面食马高鱼鸟龙龘"
const japaneseRamp = "\u3000・、。ぁぃぅぇぉあいうえおかきくけこさしすせそたちつてとなにぬねのはひふへほまみむめもやゆよらりるれろわをん漢龘"

// Nerd Font (Material Design + others) glyphs ordered roughly by visual
// weight: tiny dot, medium dot, small shapes, halves, filled shapes, solid.
const nerdRamp = " \U000F09DF\U000F09DE\U000F0A15\U000F1A0A\U000F0246\U000F1395\U000F04CE\U000F0764\U000F02D8\U000F02D1\U000F1853\U000F0F62\U000F0638\U000F0025"

var modes = []Mode{
	{Name: "ascii", Desc: "classic 70-level ASCII", Cols: 1, Rows: 1, Kind: "ramp", Ramp: asciiRamp},
	{Name: "short", Desc: "10-level ASCII (cleaner)", Cols: 1, Rows: 1, Kind: "ramp", Ramp: shortRamp},
	{Name: "shade", Desc: "shading blocks ░▒▓█", Cols: 1, Rows: 1, Kind: "ramp", Ramp: shadeRamp},
	{Name: "blocks", Desc: "half blocks ▀▄█ with real color", Cols: 1, Rows: 2, Kind: "half"},
	{Name: "quad", Desc: "quadrants ▖▗▘▝ 2x2", Cols: 2, Rows: 2, Kind: "quad"},
	{Name: "braille", Desc: "braille ⠿ 2x4, highest density", Cols: 2, Rows: 4, Kind: "braille"},
	{
		Name: "chinese", Desc: "Chinese characters (Noto Sans CJK)", Cols: 1, Rows: 1, Kind: "ramp",
		Ramp: chineseRamp, CellAspect: 1.0, ImgAspect: 0.69, Font: "Noto Sans CJK SC",
	},
	{
		Name: "japanese", Desc: "Japanese kana + kanji (Noto Sans CJK)", Cols: 1, Rows: 1, Kind: "ramp",
		Ramp: japaneseRamp, CellAspect: 1.0, ImgAspect: 0.69, Font: "Noto Sans CJK JP",
	},
	{
		Name: "nerd", Desc: "Nerd Font icons (Hack Nerd Font)", Cols: 1, Rows: 1, Kind: "ramp",
		Ramp: nerdRamp, CellAspect: 0.5, ImgAspect: 0.5, Font: "Hack Nerd Font",
	},
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
