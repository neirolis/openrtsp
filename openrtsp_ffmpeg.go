package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/mattn/go-shellwords"
	"github.com/sg3des/argum"
)

var args struct {
	Addr      string `argum:"pos,req" help:"rtsp stream address"`
	FrameRate int    `argum:"--framerate" help:"target fps" default:"5"`
	Quality   int    `argum:"--quality" help:"frames quality 1-31" default:"3"`
	Archive   string `argum:"--archive" help:"archive type: mp4 only supported now"`
}

func init() {
	log.SetFlags(log.Lshortfile)
	argum.MustParse(&args)
}

var openRTSPExec = "./openRTSP"
var openRTSPExecLine = "-V -n -v -t -c -b 10000000"
var ffmpegExec = "ffmpeg"
var ffmpegExecTmpl = "-hide_banner -loglevel level+info -y -i - -c:v mjpeg -huffman optimal -q:v {{.Quality}} -vf fps={{.FrameRate}},realtime -f image2pipe -"

func main() {

	openrtsp, err := startOpenRTSP()
	if err != nil {
		log.Fatal(err)
	}

	ffmpeg, err := startFFmpeg()
	if err != nil {
		log.Fatal(err)
	}

	// copy openrtsp stdout with stream to the ffmpeg stdin
	go io.Copy(ffmpeg.stdin, openrtsp.stdout)

	// copy output to the main process stdout and stderr
	go io.Copy(os.Stdout, ffmpeg.stdout)

	// read ffmpeg stderr
	go func() {
		s := bufio.NewScanner(ffmpeg.stderr)
		for s.Scan() {
			line := s.Text()

			if strings.Contains(line, "[warning]") {
				continue
			}

			fmt.Fprintln(os.Stderr, line)
		}
	}()

	// read openrtsp stderr
	s := bufio.NewScanner(openrtsp.stderr)
	var openrtspStderrLine string
	var openrtspStarted bool
	go func() {
		for s.Scan() {
			openrtspStderrLine = s.Text()

			if openrtspStarted {
				fmt.Fprintln(os.Stderr, openrtspStderrLine)
				continue
			}

			if !openrtspStarted && strings.Contains(openrtspStderrLine, "Data packets have begun arriving") {
				openrtspStarted = true
			}
		}
	}()

	go func() {
		if err := ffmpeg.cmd.Wait(); err != nil {
			log.Fatal(err)
		}
	}()

	if err := openrtsp.cmd.Wait(); err != nil {
		log.Fatal(openrtspStderrLine, err)
	}
}

type Proc struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser
}

func startFFmpeg() (*Proc, error) {
	// start ffmpeg
	tmpl, err := template.New("").Parse(ffmpegExecTmpl)
	if err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer([]byte{})
	if err := tmpl.Execute(buf, args); err != nil {
		return nil, err
	}

	execLine := buf.String()

	rtmipdir := os.Getenv("RTMIPDIR")
	// enable mp4 archive
	if arhiveDir, ok := os.LookupEnv("ARCHIVEDIR"); args.Archive == "mp4" && ok {
		execLine += " -f stream_segment -c:v copy -segment_format_options movflags=+frag_keyframe+empty_moov+faststart -segment_time 01:00:00 -strftime 1 " + filepath.Join(arhiveDir, "%Y-%m-%d-%H-%M-%S.mp4")
	}

	execArgs, err := shellwords.Parse(execLine)
	if err != nil {
		return nil, err
	}

	ffmpegExec := "ffmpeg"
	if rtmipdir != "" {
		ffmpegExec = filepath.Join(rtmipdir, ffmpegExec)
	}

	cmd := exec.Command(ffmpegExec, execArgs...)
	cmd.Dir = rtmipdir
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return &Proc{cmd, stdin, stdout, stderr}, nil
}

func startOpenRTSP() (*Proc, error) {
	execArgs, err := shellwords.Parse(openRTSPExecLine)
	if err != nil {
		return nil, err
	}
	execArgs = append(execArgs, args.Addr)

	cmd := exec.Command(openRTSPExec, execArgs...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return &Proc{cmd, stdin, stdout, stderr}, nil
}
