package runner

// Minimal YAML frontmatter parser for maxwell.md and SKILL.md.
//
// We deliberately avoid pulling in gopkg.in/yaml.v3 to keep the binary
// dependency-free for v0.1. The frontmatter shapes we accept are bounded
// (documented in spec §5.2 and §5.4); we parse exactly that shape.
//
// If the spec grows beyond what this parser handles, we'll switch to a
// real YAML library. Until then, every dependency-free release is a win.

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// parseFrontmatter parses the top-level maxwell.md frontmatter into Config.
func parseFrontmatter(s string) (*Config, error) {
	root, err := parseYAMLBlock(s, 0)
	if err != nil {
		return nil, err
	}
	cfg := &Config{}

	if v, ok := root["maxwell_spec_version"].(string); ok {
		cfg.MaxwellSpecVersion = v
	}
	if v, ok := root["verdict_model"].(string); ok {
		cfg.VerdictModel = v
	}
	if v, ok := root["self_judgment"].(string); ok {
		cfg.SelfJudgment = v
	}

	if rawGates, ok := root["gates"].([]interface{}); ok {
		for _, gv := range rawGates {
			m, ok := gv.(map[string]interface{})
			if !ok {
				return nil, errors.New("gates: each entry must be a map")
			}
			ref := GateRef{}
			if name, ok := m["name"].(string); ok {
				ref.Name = name
			}
			if sev, ok := m["severity"].(string); ok {
				ref.Severity = Severity(sev)
			}
			if ref.Name == "" {
				return nil, errors.New("gates: name is required")
			}
			cfg.Gates = append(cfg.Gates, ref)
		}
	}

	cfg.Hooks = map[string][]HookConfig{}
	if rawHooks, ok := root["hooks"].(map[string]interface{}); ok {
		for ev, rawList := range rawHooks {
			list, ok := rawList.([]interface{})
			if !ok {
				continue
			}
			for _, item := range list {
				m, ok := item.(map[string]interface{})
				if !ok {
					continue
				}
				h := HookConfig{}
				if v, ok := m["matcher"].(string); ok {
					h.Matcher = v
				}
				if v, ok := m["command"].(string); ok {
					h.Command = v
				}
				if v, ok := m["timeoutSec"].(int); ok {
					h.TimeoutSec = v
				}
				cfg.Hooks[ev] = append(cfg.Hooks[ev], h)
			}
		}
	}

	if rawRunner, ok := root["runner"].(map[string]interface{}); ok {
		if v, ok := rawRunner["timeout_ms"].(int); ok {
			cfg.Runner.TimeoutMS = v
		}
		if v, ok := rawRunner["flake_budget_pct"].(float64); ok {
			cfg.Runner.FlakeBudgetPct = v
		}
		if v, ok := rawRunner["max_reflections_per_gate"].(int); ok {
			cfg.Runner.MaxReflectionsPerGate = v
		}
	}

	if rawWS, ok := root["workspace"].(map[string]interface{}); ok {
		if v, ok := rawWS["isolation"].(string); ok {
			cfg.Workspace.Isolation = v
		}
		if v, ok := rawWS["cleanup"].(string); ok {
			cfg.Workspace.Cleanup = v
		}
		cfg.Workspace.ProdCodeGlobs = stringList(rawWS["prod_code_globs"])
		cfg.Workspace.TestGlobs = stringList(rawWS["test_globs"])
	}

	if rawObs, ok := root["observability"].(map[string]interface{}); ok {
		if v, ok := rawObs["transcripts"].(string); ok {
			cfg.Observability.Transcripts = v
		}
		if v, ok := rawObs["cot_preservation"].(string); ok {
			cfg.Observability.CoTPreservation = v
		}
		if v, ok := rawObs["eval_awareness_recording"].(string); ok {
			cfg.Observability.EvalAwarenessRecording = v
		}
	}

	if rawPerm, ok := root["permissions"].(map[string]interface{}); ok {
		if v, ok := rawPerm["mode"].(string); ok {
			cfg.Permissions.Mode = v
		}
		cfg.Permissions.Allow = stringList(rawPerm["allow"])
		cfg.Permissions.Deny = stringList(rawPerm["deny"])
	}

	return cfg, nil
}

// parseGateFrontmatter parses SKILL.md frontmatter into a Gate.
func parseGateFrontmatter(s string) (Gate, error) {
	root, err := parseYAMLBlock(s, 0)
	if err != nil {
		return Gate{}, err
	}
	g := Gate{}
	if v, ok := root["name"].(string); ok {
		g.Name = v
	}
	if v, ok := root["description"].(string); ok {
		g.Description = v
	}
	if v, ok := root["severity"].(string); ok {
		g.Severity = Severity(v)
	}
	if v, ok := root["maxwell_spec_version"].(string); ok {
		g.MaxwellSpecVersion = v
	}
	g.Assumptions = stringList(root["assumptions"])
	if g.Name == "" {
		return Gate{}, errors.New("SKILL.md frontmatter: name is required")
	}
	if g.Description == "" {
		return Gate{}, errors.New("SKILL.md frontmatter: description is required")
	}
	if g.MaxwellSpecVersion == "" {
		return Gate{}, errors.New("SKILL.md frontmatter: maxwell_spec_version is required")
	}
	return g, nil
}

func stringList(v interface{}) []string {
	if v == nil {
		return nil
	}
	rawList, ok := v.([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, 0, len(rawList))
	for _, x := range rawList {
		if s, ok := x.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// parseYAMLBlock is a tiny recursive parser for the YAML subset we use:
// scalars (string, int, float, bool), block mappings, and block sequences.
// Indent is space-only. No flow style. No anchors. No tags.
//
// The function returns a map[string]interface{} for the top-level block.
//
// This parser is deliberately small and forgiving for hand-authored YAML.
// For machine-generated YAML or complex shapes, swap in a real library.
func parseYAMLBlock(s string, _ int) (map[string]interface{}, error) {
	lines := splitLinesPreserve(s)
	idx := 0
	return parseMap(lines, &idx, 0)
}

func parseMap(lines []string, idx *int, indent int) (map[string]interface{}, error) {
	out := map[string]interface{}{}
	for *idx < len(lines) {
		raw := lines[*idx]
		line := stripComment(raw)
		if strings.TrimSpace(line) == "" {
			*idx++
			continue
		}
		curIndent := leadingSpaces(line)
		if curIndent < indent {
			return out, nil
		}
		if curIndent > indent {
			return nil, fmt.Errorf("yaml: unexpected indent at line %d (%q)", *idx+1, raw)
		}
		trimmed := strings.TrimSpace(line)
		// Sequence under a mapping is illegal at the top of a mapping line.
		if strings.HasPrefix(trimmed, "- ") || trimmed == "-" {
			return nil, fmt.Errorf("yaml: unexpected sequence entry at line %d under a mapping", *idx+1)
		}
		colon := strings.Index(trimmed, ":")
		if colon < 0 {
			return nil, fmt.Errorf("yaml: missing colon at line %d (%q)", *idx+1, raw)
		}
		key := strings.TrimSpace(trimmed[:colon])
		valStr := strings.TrimSpace(trimmed[colon+1:])
		*idx++
		if valStr != "" {
			out[key] = decodeScalar(valStr)
			continue
		}
		// Block value follows: either map or list, identified by the next
		// non-blank line's indentation and whether it starts with "- ".
		next := peekNonBlank(lines, *idx)
		if next == -1 || leadingSpaces(stripComment(lines[next])) <= curIndent {
			out[key] = nil
			continue
		}
		nextIndent := leadingSpaces(stripComment(lines[next]))
		nextTrim := strings.TrimSpace(stripComment(lines[next]))
		if strings.HasPrefix(nextTrim, "- ") || nextTrim == "-" {
			seq, err := parseSeq(lines, idx, nextIndent)
			if err != nil {
				return nil, err
			}
			out[key] = seq
		} else {
			child, err := parseMap(lines, idx, nextIndent)
			if err != nil {
				return nil, err
			}
			out[key] = child
		}
	}
	return out, nil
}

func parseSeq(lines []string, idx *int, indent int) ([]interface{}, error) {
	var out []interface{}
	for *idx < len(lines) {
		raw := lines[*idx]
		line := stripComment(raw)
		if strings.TrimSpace(line) == "" {
			*idx++
			continue
		}
		curIndent := leadingSpaces(line)
		if curIndent < indent {
			return out, nil
		}
		if curIndent > indent {
			return nil, fmt.Errorf("yaml: unexpected indent in sequence at line %d (%q)", *idx+1, raw)
		}
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "- ") && trimmed != "-" {
			return out, nil
		}
		// Strip the "- " prefix to get either a scalar or a same-line k:v.
		body := strings.TrimPrefix(trimmed, "-")
		body = strings.TrimSpace(body)
		*idx++
		if body == "" {
			// Block map under "-": parse the following block at indent+2.
			next := peekNonBlank(lines, *idx)
			if next < 0 {
				out = append(out, nil)
				continue
			}
			child, err := parseMap(lines, idx, leadingSpaces(stripComment(lines[next])))
			if err != nil {
				return nil, err
			}
			out = append(out, child)
			continue
		}
		// Inline form: either scalar ("- value") or mapping ("- key: value").
		if strings.Contains(body, ":") && !looksLikeStringScalar(body) {
			// Single-line mapping entry; feed back as an indented mapping.
			fakeKey, fakeVal, ok := splitKVOnce(body)
			if ok {
				m := map[string]interface{}{}
				if fakeVal == "" {
					// Possibly a multi-line map; parse following lines indented further.
					next := peekNonBlank(lines, *idx)
					if next >= 0 && leadingSpaces(stripComment(lines[next])) > indent {
						child, err := parseMap(lines, idx, leadingSpaces(stripComment(lines[next])))
						if err != nil {
							return nil, err
						}
						for k, v := range child {
							m[k] = v
						}
					}
					m[fakeKey] = nil
				} else {
					m[fakeKey] = decodeScalar(fakeVal)
					// Continue absorbing further keys at indent+2 of this list entry.
					next := peekNonBlank(lines, *idx)
					if next >= 0 && leadingSpaces(stripComment(lines[next])) > indent {
						extraIndent := leadingSpaces(stripComment(lines[next]))
						extra, err := parseMap(lines, idx, extraIndent)
						if err != nil {
							return nil, err
						}
						for k, v := range extra {
							m[k] = v
						}
					}
				}
				out = append(out, m)
				continue
			}
		}
		out = append(out, decodeScalar(body))
	}
	return out, nil
}

func splitKVOnce(s string) (string, string, bool) {
	idx := strings.Index(s, ":")
	if idx < 0 {
		return s, "", false
	}
	key := strings.TrimSpace(s[:idx])
	val := strings.TrimSpace(s[idx+1:])
	return key, val, true
}

func looksLikeStringScalar(s string) bool {
	return strings.HasPrefix(s, `"`) || strings.HasPrefix(s, `'`)
}

func decodeScalar(s string) interface{} {
	if s == "" {
		return ""
	}
	if (strings.HasPrefix(s, `"`) && strings.HasSuffix(s, `"`)) ||
		(strings.HasPrefix(s, `'`) && strings.HasSuffix(s, `'`)) {
		return s[1 : len(s)-1]
	}
	switch strings.ToLower(s) {
	case "true":
		return true
	case "false":
		return false
	case "null", "~":
		return nil
	}
	if i, err := strconv.Atoi(s); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}
	return s
}

func leadingSpaces(s string) int {
	n := 0
	for _, r := range s {
		if r == ' ' {
			n++
			continue
		}
		break
	}
	return n
}

func stripComment(s string) string {
	// A '#' that is not inside quotes starts a line comment.
	inSingle, inDouble := false, false
	for i, r := range s {
		switch r {
		case '\'':
			if !inDouble {
				inSingle = !inSingle
			}
		case '"':
			if !inSingle {
				inDouble = !inDouble
			}
		case '#':
			if !inSingle && !inDouble {
				return s[:i]
			}
		}
	}
	return s
}

func splitLinesPreserve(s string) []string {
	return strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
}

func peekNonBlank(lines []string, from int) int {
	for i := from; i < len(lines); i++ {
		if strings.TrimSpace(stripComment(lines[i])) != "" {
			return i
		}
	}
	return -1
}
