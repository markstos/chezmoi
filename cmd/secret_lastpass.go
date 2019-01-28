package cmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"unicode"

	"github.com/spf13/cobra"
	"github.com/twpayne/chezmoi/lib/chezmoi"
	vfs "github.com/twpayne/go-vfs"
)

var lastpassCmd = &cobra.Command{
	Use:   "lastpass [args...]",
	Short: "Execute the LastPass CLI (lpass)",
	RunE:  makeRunE(config.runLastpassCmd),
}

var lastpassParseNoteRegexp = regexp.MustCompile(`\A([ A-Za-z]*):(.*)\z`)

type lastpassCmdConfig struct {
	Lpass string
}

var lastPassCache = make(map[string]interface{})

func init() {
	config.Lastpass.Lpass = "lpass"
	config.addTemplateFunc("lastpass", config.lastpassFunc)

	secretCmd.AddCommand(lastpassCmd)
}

func (c *Config) runLastpassCmd(fs vfs.FS, args []string) error {
	return c.exec(append([]string{c.Lastpass.Lpass}, args...))
}

func (c *Config) lastpassFunc(id string) interface{} {
	if data, ok := lastPassCache[id]; ok {
		return data
	}
	name := c.Lastpass.Lpass
	args := []string{"show", "-j", id}
	if c.Verbose {
		fmt.Printf("%s %s\n", name, strings.Join(args, " "))
	}
	output, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		chezmoi.ReturnTemplateFuncError(fmt.Errorf("lastpass: %s %s: %v\n%s", name, strings.Join(args, " "), err, output))
	}
	var data []map[string]interface{}
	if err := json.Unmarshal(output, &data); err != nil {
		chezmoi.ReturnTemplateFuncError(fmt.Errorf("lastpass: %s %s: %v\n%s", name, strings.Join(args, " "), err, output))
	}
	for _, d := range data {
		if note, ok := d["note"].(string); ok {
			d["note"] = lastpassParseNote(note)
		}
	}
	lastPassCache[id] = data
	return data
}

func lastpassParseNote(note string) map[string]string {
	result := make(map[string]string)
	s := bufio.NewScanner(bytes.NewBufferString(note))
	key := ""
	for s.Scan() {
		if m := lastpassParseNoteRegexp.FindStringSubmatch(s.Text()); m != nil {
			keyComponents := strings.Split(m[1], " ")
			firstComponentRunes := []rune(keyComponents[0])
			firstComponentRunes[0] = unicode.ToLower(firstComponentRunes[0])
			keyComponents[0] = string(firstComponentRunes)
			key = strings.Join(keyComponents, "")
			result[key] = m[2] + "\n"
		} else {
			result[key] += s.Text() + "\n"
		}
	}
	if err := s.Err(); err != nil {
		chezmoi.ReturnTemplateFuncError(fmt.Errorf("lastpassParseNote: %v", err))
	}
	return result
}
