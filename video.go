package main

import (
	"bufio"
	"flag"
	"fmt"
	"image"
	"io"
	"math"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func runVideo(args []string) {
	fs := flag.NewFlagSet("asciix video", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		mode       string
		width      int
		fps        float64
		forceColor bool
		noColor    bool
		loop       bool
		invert     bool
		threshold  int
	)
	fs.StringVar(&mode, "m", "blocks", "modo")
	fs.StringVar(&mode, "mode", "blocks", "modo")
	fs.IntVar(&width, "w", 0, "ancho en caracteres")
	fs.IntVar(&width, "width", 0, "ancho en caracteres")
	fs.Float64Var(&fps, "fps", 15, "fotogramas por segundo")
	fs.BoolVar(&forceColor, "color", false, "color")
	fs.BoolVar(&noColor, "no-color", false, "sin color")
	fs.BoolVar(&loop, "loop", false, "repetir")
	fs.BoolVar(&invert, "invert", false, "invertir")
	fs.IntVar(&threshold, "threshold", 128, "umbral")

	if err := parseInterspersed(fs, args); err != nil {
		os.Exit(2)
	}
	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "uso: asciix video [opciones] archivo")
		os.Exit(2)
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		fmt.Fprintln(os.Stderr, "asciix: se requiere ffmpeg para convertir vídeo")
		os.Exit(1)
	}
	video := fs.Arg(0)

	iw, ih, err := probeSize(video)
	if err != nil {
		fmt.Fprintln(os.Stderr, "asciix:", err)
		os.Exit(1)
	}
	tw, th := termSize()
	if width < 1 {
		width = tw
	}
	color := !noColor && (forceColor || isTerminal(os.Stdout))

	m := modeByName(mode)
	ratio := float64(ih) / float64(iw)
	h := int(math.Round(float64(width) * ratio * m.cellAspect()))
	if h < 1 {
		h = 1
	}
	sw, sh := width*m.Cols, h*m.Rows
	if sh > th*2 {
		sh = th * 2
		h = sh / m.Rows
	}

	ffargs := []string{"-hide_banner", "-loglevel", "error"}
	if loop {
		ffargs = append(ffargs, "-stream_loop", "-1")
	}
	ffargs = append(ffargs,
		"-i", video,
		"-vf", fmt.Sprintf("scale=%d:%d", sw, sh),
		"-r", strconv.FormatFloat(fps, 'f', -1, 64),
		"-f", "rawvideo", "-pix_fmt", "rgb24", "-",
	)
	cmd := exec.Command("ffmpeg", ffargs...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Fprintln(os.Stderr, "asciix:", err)
		os.Exit(1)
	}
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		fmt.Fprintln(os.Stderr, "asciix:", err)
		os.Exit(1)
	}

	out := bufio.NewWriter(os.Stdout)
	defer out.Flush()
	fmt.Fprint(out, "\x1b[?25l")
	defer fmt.Fprint(out, "\x1b[?25h\x1b[0m")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	done := make(chan struct{})
	go func() {
		<-sig
		cmd.Process.Kill()
		close(done)
	}()

	o := Options{Mode: mode, Width: width, Height: h, Color: color, Invert: invert, Threshold: threshold}
	frameSize := sw * sh * 3
	buf := make([]byte, frameSize)
	interval := time.Duration(float64(time.Second) / fps)

	for {
		if _, err := io.ReadFull(stdout, buf); err != nil {
			break
		}
		frame := &image.RGBA{Pix: make([]byte, sw*sh*4), Stride: sw * 4, Rect: image.Rect(0, 0, sw, sh)}
		for i, j := 0, 0; i < frameSize; i, j = i+3, j+4 {
			frame.Pix[j] = buf[i]
			frame.Pix[j+1] = buf[i+1]
			frame.Pix[j+2] = buf[i+2]
			frame.Pix[j+3] = 255
		}
		g := Convert(frame, o)
		out.WriteString("\x1b[2J\x1b[H")
		if color {
			out.WriteString(g.ANSI())
		} else {
			out.WriteString(g.Plain())
		}
		out.Flush()
		select {
		case <-done:
			cmd.Wait()
			return
		default:
		}
		time.Sleep(interval)
	}
	cmd.Wait()
}

func probeSize(path string) (int, int, error) {
	out, err := exec.Command("ffprobe", "-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=width,height",
		"-of", "csv=s=x:p=0", path).Output()
	if err != nil {
		return 0, 0, fmt.Errorf("ffprobe falló: %v", err)
	}
	fields := strings.Split(strings.TrimSpace(string(out)), "x")
	if len(fields) != 2 {
		return 0, 0, fmt.Errorf("no se pudo obtener el tamaño del vídeo")
	}
	w, _ := strconv.Atoi(fields[0])
	h, _ := strconv.Atoi(fields[1])
	if w == 0 || h == 0 {
		return 0, 0, fmt.Errorf("tamaño de vídeo inválido")
	}
	return w, h, nil
}
