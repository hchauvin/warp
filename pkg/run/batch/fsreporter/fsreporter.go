// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

package fsreporter

import (
	"encoding/json"
	"fmt"
	"github.com/hchauvin/warp/pkg/run/batch"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type FsReporter struct {
	Path   string
	Report Report
	mut    sync.Mutex
}

type Report struct {
	EnvironmentSetupResults []batch.EnvironmentSetupResult
	Results                 []batch.CommandResult
}

func New(path string) (*FsReporter, error) {
	if err := os.RemoveAll(path); err != nil {
		return nil, err
	}
	return &FsReporter{
		Path: path,
	}, nil
}

func (reporter *FsReporter) EnvironmentSetupResult(result *batch.EnvironmentSetupResult) {
	reporter.mut.Lock()
	defer reporter.mut.Unlock()
	reporter.Report.EnvironmentSetupResults = append(reporter.Report.EnvironmentSetupResults, *result)
}

func (reporter *FsReporter) CommandOutput(info *batch.CommandInfo) (io.WriteCloser, error) {
	path := reporter.commandOutputPath(info)

	if err := os.MkdirAll(filepath.Dir(path), 0777); err != nil {
		return nil, err
	}

	return os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)
}

func (reporter *FsReporter) CommandResult(result *batch.CommandResult) {
	reporter.mut.Lock()
	defer reporter.mut.Unlock()
	reporter.Report.Results = append(reporter.Report.Results, *result)
}

func (reporter *FsReporter) Finalize() error {
	path := filepath.Join(reporter.Path, "report.json")

	if err := os.MkdirAll(filepath.Dir(path), 0777); err != nil {
		return err
	}

	b, err := json.MarshalIndent(reporter.Report, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, b, 0777)
}

func (reporter *FsReporter) commandOutputPath(info *batch.CommandInfo) string {
	return filepath.Join(reporter.Path, "log", fmt.Sprintf("%s.%d.txt", commandNameToPath(info.Name), info.Tries))
}

const allowedRunes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_"

func commandNameToPath(commandName string) string {
	var baseName strings.Builder
	var tags []string
	var curTag strings.Builder
	var inTag bool
	for _, c := range commandName {
		if c == '[' {
			inTag = true
		} else if c == ']' {
			tags = append(tags, curTag.String())
			curTag.Reset()
			inTag = false
		} else if inTag {
			curTag.WriteRune(c)
		} else if strings.ContainsRune(allowedRunes, c) {
			baseName.WriteRune(c)
		} else {
			baseName.WriteRune('_')
		}
	}

	var path strings.Builder
	for _, tag := range tags {
		path.WriteString(tag)
		path.WriteRune(os.PathSeparator)
	}
	path.WriteString(baseName.String())

	return path.String()
}
