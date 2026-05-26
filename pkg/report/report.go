package report

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"

	"gitea.cervbox.synology.me/CervoSoft/cervo-mutant/pkg/engine"
)

func JSON(result engine.RunResult) ([]byte, error) {
	if result.SchemaVersion == "" {
		result.SchemaVersion = "1"
	}
	if result.Thresholds == nil {
		result.Thresholds = map[string]any{}
	}
	return json.MarshalIndent(result, "", "  ")
}

func Summary(result engine.RunResult) string {
	return fmt.Sprintf("Mutation score: %.2f%%\nKilled: %d\nSurvived: %d\nQuarantined: %d\nTimed out: %d\nCompile errors: %d\n",
		result.Summary.Score,
		result.Summary.Killed,
		result.Summary.Survived,
		result.Summary.Quarantined,
		result.Summary.TimedOut,
		result.Summary.CompileError,
	)
}

func Survivors(result engine.RunResult) string {
	var b strings.Builder
	for _, mutant := range result.Mutants {
		if mutant.Status != engine.StatusSurvived {
			continue
		}
		fmt.Fprintf(&b, "%s %s:%d %s %s -> %s\n", mutant.MutantID, mutant.Mutant.File, mutant.Mutant.Line, mutant.Mutant.Operator, mutant.Mutant.Original, mutant.Mutant.Mutated)
	}
	return b.String()
}

func HTML(result engine.RunResult) string {
	var b strings.Builder
	b.WriteString("<!doctype html><html><head><meta charset=\"utf-8\"><title>cervomut report</title></head><body>")
	b.WriteString("<h1>cervomut mutation report</h1>")
	b.WriteString("<pre>")
	b.WriteString(html.EscapeString(Summary(result)))
	b.WriteString("</pre><table><thead><tr><th>Status</th><th>Mutant</th><th>Diff</th></tr></thead><tbody>")
	for _, mutant := range result.Mutants {
		b.WriteString("<tr><td>")
		b.WriteString(html.EscapeString(string(mutant.Status)))
		b.WriteString("</td><td>")
		b.WriteString(html.EscapeString(mutant.MutantID))
		b.WriteString("</td><td><pre>")
		b.WriteString(html.EscapeString(mutant.Mutant.Diff))
		b.WriteString("</pre></td></tr>")
	}
	b.WriteString("</tbody></table></body></html>")
	return b.String()
}

type junitTestsuite struct {
	XMLName  xml.Name        `xml:"testsuite"`
	Name     string          `xml:"name,attr"`
	Tests    int             `xml:"tests,attr"`
	Failures int             `xml:"failures,attr"`
	Cases    []junitTestcase `xml:"testcase"`
}

type junitTestcase struct {
	Name    string        `xml:"name,attr"`
	Failure *junitFailure `xml:"failure,omitempty"`
}

type junitFailure struct {
	Message string `xml:"message,attr"`
	Text    string `xml:",chardata"`
}

func JUnit(result engine.RunResult) ([]byte, error) {
	suite := junitTestsuite{Name: "cervomut", Tests: len(result.Mutants)}
	for _, mutant := range result.Mutants {
		tc := junitTestcase{Name: mutant.MutantID}
		if mutant.Status == engine.StatusSurvived || mutant.Status == engine.StatusTimedOut {
			suite.Failures++
			tc.Failure = &junitFailure{Message: string(mutant.Status), Text: mutant.StatusReason}
		}
		suite.Cases = append(suite.Cases, tc)
	}
	return xml.MarshalIndent(suite, "", "  ")
}

func WriteAll(dir string, result engine.RunResult) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	jsonData, err := JSON(result)
	if err != nil {
		return err
	}
	junitData, err := JUnit(result)
	if err != nil {
		return err
	}
	files := map[string][]byte{
		"summary.txt":          []byte(Summary(result)),
		"mutation-report.json": jsonData,
		"junit.xml":            junitData,
		"index.html":           []byte(HTML(result)),
		"survivors.txt":        []byte(Survivors(result)),
	}
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			return err
		}
	}
	return nil
}
