package api

import (
	"encoding/json"
	"strings"

	"nofx/prediction/types"
)

func redactPreviewWireJSON(wire string) string {
	if wire == "" {
		return ""
	}
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(wire), &obj); err != nil {
		if len(wire) > 32 {
			return wire[:16] + "…[redacted]"
		}
		return "[redacted]"
	}
	if sig, ok := obj["signature"].(string); ok && len(sig) > 16 {
		obj["signature"] = sig[:10] + "…[redacted]"
	}
	b, _ := json.Marshal(obj)
	return string(b)
}

func redactExecutionOutcomes(execs []types.ExecutionOutcome) []types.ExecutionOutcome {
	if len(execs) == 0 {
		return execs
	}
	out := make([]types.ExecutionOutcome, len(execs))
	copy(out, execs)
	for i := range out {
		if out[i].PreviewWire != "" {
			out[i].PreviewWire = redactPreviewWireJSON(out[i].PreviewWire)
		}
	}
	return out
}

func redactPreviewWireField(wire string) string {
	return redactPreviewWireJSON(wire)
}

func includeFullPreviewWire(cQuery string) bool {
	return strings.EqualFold(strings.TrimSpace(cQuery), "true")
}
