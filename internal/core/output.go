package core

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/fatih/color"
)

type Event struct {
	Type    string `json:"event"`
	Path    string `json:"path,omitempty"`
	Reason  string `json:"reason,omitempty"`
	Detail  string `json:"detail,omitempty"`
	Success bool   `json:"success,omitempty"`
}

type Summary struct {
	Type         string `json:"event"`
	FilesCreated int    `json:"files_created"`
	FilesSkipped int    `json:"files_skipped"`
	Errors       int    `json:"errors"`
}

type Renderer interface {
	FileCreated(path string)
	FileSkipped(path string, reason string)
	Error(err error)
	Summary(created, skipped, errors int)
}

type HumanRenderer struct{ Verbose bool }

func (h *HumanRenderer) FileCreated(path string) {
	fmt.Printf("    %s %s\n", color.GreenString("+"), path)
}
func (h *HumanRenderer) FileSkipped(path string, reason string) {
	fmt.Printf("    %s %s (%s)\n", color.YellowString("~"), path, reason)
}
func (h *HumanRenderer) Error(err error) {
	fmt.Printf("    %s %v\n", color.RedString("ERR"), err)
}
func (h *HumanRenderer) Summary(created, skipped, errors int) {
	fmt.Println()
	bold := color.New(color.Bold)
	bold.Println("  Results:")
	fmt.Printf("    %s %d files created\n", color.GreenString("OK"), created)
	if skipped > 0 {
		fmt.Printf("    %s %d files skipped\n", color.YellowString("WARN"), skipped)
	}
	if errors > 0 {
		fmt.Printf("    %s %d errors\n", color.RedString("FAIL"), errors)
	}
	fmt.Println()
}

type JSONRenderer struct{ Writer io.Writer }

func (j *JSONRenderer) FileCreated(path string) {
	j.emit(Event{Type: "file_created", Path: path, Success: true})
}
func (j *JSONRenderer) FileSkipped(path string, reason string) {
	j.emit(Event{Type: "file_skipped", Path: path, Reason: reason})
}
func (j *JSONRenderer) Error(err error) {
	j.emit(Event{Type: "error", Detail: err.Error()})
}
func (j *JSONRenderer) Summary(created, skipped, errors int) {
	j.emit(Summary{Type: "complete", FilesCreated: created, FilesSkipped: skipped, Errors: errors})
}
func (j *JSONRenderer) emit(v any) {
	b, _ := json.Marshal(v)
	fmt.Fprintln(j.Writer, string(b))
}

type NDJSONRenderer struct{ Writer io.Writer }

func (n *NDJSONRenderer) FileCreated(path string) {
	n.emit(Event{Type: "file_created", Path: path, Success: true})
}
func (n *NDJSONRenderer) FileSkipped(path string, reason string) {
	n.emit(Event{Type: "file_skipped", Path: path, Reason: reason})
}
func (n *NDJSONRenderer) Error(err error) {
	n.emit(Event{Type: "error", Detail: err.Error()})
}
func (n *NDJSONRenderer) Summary(created, skipped, errors int) {
	n.emit(Summary{Type: "complete", FilesCreated: created, FilesSkipped: skipped, Errors: errors})
}
func (n *NDJSONRenderer) emit(v any) {
	b, _ := json.Marshal(v)
	fmt.Fprintln(n.Writer, string(b))
}

func NewRenderer(format string) Renderer {
	switch format {
	case "json":
		return &JSONRenderer{Writer: os.Stdout}
	case "ndjson":
		return &NDJSONRenderer{Writer: os.Stdout}
	default:
		return &HumanRenderer{Verbose: true}
	}
}
