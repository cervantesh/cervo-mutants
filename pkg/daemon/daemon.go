package daemon

import (
	"bufio"
	"context"
	"encoding/json"
	"io"

	"gitea.cervbox.synology.me/CervoSoft/cervo-mutant/pkg/engine"
)

type Message struct {
	Type   string              `json:"type"`
	Job    engine.MutantJob    `json:"job,omitempty"`
	Result engine.MutantResult `json:"result,omitempty"`
	Error  string              `json:"error,omitempty"`
}

func ServeJSONLines(ctx context.Context, in io.Reader, out io.Writer, runner engine.Runner) error {
	scanner := bufio.NewScanner(in)
	enc := json.NewEncoder(out)
	for scanner.Scan() {
		var msg Message
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			_ = enc.Encode(Message{Type: "error", Error: err.Error()})
			continue
		}
		if msg.Type != "job" {
			_ = enc.Encode(Message{Type: "error", Error: "unsupported message type"})
			continue
		}
		result, err := runner.Run(ctx, msg.Job)
		if err != nil {
			_ = enc.Encode(Message{Type: "error", Error: err.Error()})
			continue
		}
		if err := enc.Encode(Message{Type: "result", Result: result}); err != nil {
			return err
		}
	}
	return scanner.Err()
}
