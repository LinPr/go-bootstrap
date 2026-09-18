package otlplogfile

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

// base64ToHex decodes a base64-encoded string and returns its hex representation.
func base64ToHex(s string) (string, error) {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return "", fmt.Errorf("base64 decode: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// decodeSpanIDs walks the OTLP JSON structure and converts base64-encoded
// spanId fields in each log record to their hex representation.
func decodeSpanIDs(data map[string]any) {
	resourceLogs, ok := data["resourceLogs"].([]any)
	if !ok {
		return
	}
	for _, rl := range resourceLogs {
		rlMap, ok := rl.(map[string]any)
		if !ok {
			continue
		}
		scopeLogs, ok := rlMap["scopeLogs"].([]any)
		if !ok {
			continue
		}
		for _, sl := range scopeLogs {
			slMap, ok := sl.(map[string]any)
			if !ok {
				continue
			}
			logRecords, ok := slMap["logRecords"].([]any)
			if !ok {
				continue
			}
			for _, lr := range logRecords {
				lrMap, ok := lr.(map[string]any)
				if !ok {
					continue
				}
				if traceId, ok := lrMap["traceId"].(string); ok {
					if hexID, err := base64ToHex(traceId); err == nil {
						lrMap["traceId"] = hexID
					}
				}
				if spanId, ok := lrMap["spanId"].(string); ok {
					if hexID, err := base64ToHex(spanId); err == nil {
						lrMap["spanId"] = hexID
					}
				}
			}
		}
	}
}
