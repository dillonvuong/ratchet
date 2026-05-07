package runner

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// LoadConfig reads ratchet.md, parses front matter and prompt body,
// loads the referenced skill bundles, and returns a typed Config.
// Spec §5.2, §6.1.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("missing ratchet.md at %s", path)
		}
		return nil, fmt.Errorf("read ratchet.md: %w", err)
	}

	frontmatter, body, err := splitFrontmatter(string(data))
	if err != nil {
		return nil, fmt.Errorf("parse ratchet.md: %w", err)
	}

	cfg, err := parseFrontmatter(frontmatter)
	if err != nil {
		return nil, fmt.Errorf("parse ratchet.md frontmatter: %w", err)
	}
	cfg.PromptBody = strings.TrimSpace(body)
	if cfg.PromptBody == "" {
		return nil, errors.New("ratchet.md body is empty (doctrine prompt is required)")
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	cfg.ConfigPath = abs

	repoRoot := filepath.Dir(abs)
	for _, ref := range cfg.Gates {
		gate, err := LoadSkill(repoRoot, ref)
		if err != nil {
			return nil, err
		}
		cfg.LoadedGates = append(cfg.LoadedGates, gate)
	}

	return cfg, nil
}

// Validate enforces the spec preconditions for a usable config. Spec §6.2.
func (c *Config) Validate(runnerSpecVersion string) error {
	if c.RatchetSpecVersion == "" {
		return errors.New("ratchet.md frontmatter: ratchet_spec_version is required")
	}
	if !specVersionsCompatible(c.RatchetSpecVersion, runnerSpecVersion) {
		return fmt.Errorf("ratchet.md spec version %q is not compatible with runner %q",
			c.RatchetSpecVersion, runnerSpecVersion)
	}
	if c.VerdictModel != "" && c.VerdictModel != "hard" && c.VerdictModel != "advisory" {
		return fmt.Errorf("ratchet.md frontmatter: verdict_model must be 'hard' or 'advisory' (got %q)", c.VerdictModel)
	}
	if c.SelfJudgment == "permitted" {
		return errors.New("ratchet.md frontmatter: self_judgment 'permitted' is reserved for future versions; v0.1 forbids LLM self-judgment for primary verdicts")
	}
	if len(c.Gates) == 0 {
		return errors.New("ratchet.md frontmatter: at least one gate must be declared under gates:")
	}
	for _, gate := range c.LoadedGates {
		if !specVersionsCompatible(gate.RatchetSpecVersion, runnerSpecVersion) {
			return fmt.Errorf("gate %q is pinned to spec %q which is not compatible with runner %q",
				gate.Name, gate.RatchetSpecVersion, runnerSpecVersion)
		}
		if len(gate.Scripts) == 0 {
			return fmt.Errorf("gate %q has no scripts under .skills/%s/scripts/", gate.Name, gate.Name)
		}
	}
	return nil
}

// FindGate returns the Gate with the given name from the loaded set.
func (c *Config) FindGate(name string) (Gate, error) {
	for _, g := range c.LoadedGates {
		if g.Name == name {
			return g, nil
		}
	}
	return Gate{}, fmt.Errorf("gate %q not declared in ratchet.md", name)
}

// LoadSkill walks .skills/<name>/ and returns a populated Gate.
// Spec §5.4.
func LoadSkill(repoRoot string, ref GateRef) (Gate, error) {
	dir := filepath.Join(repoRoot, ".skills", ref.Name)
	skillMD := filepath.Join(dir, "SKILL.md")
	data, err := os.ReadFile(skillMD)
	if err != nil {
		return Gate{}, fmt.Errorf("load skill %q: %w", ref.Name, err)
	}

	front, _, err := splitFrontmatter(string(data))
	if err != nil {
		return Gate{}, fmt.Errorf("parse SKILL.md for %q: %w", ref.Name, err)
	}

	gate, err := parseGateFrontmatter(front)
	if err != nil {
		return Gate{}, err
	}
	if gate.Name != ref.Name {
		return Gate{}, fmt.Errorf("SKILL.md name %q does not match directory %q", gate.Name, ref.Name)
	}
	if ref.Severity != "" {
		gate.Severity = ref.Severity
	}
	if gate.Severity == "" {
		gate.Severity = SeverityP1
	}
	gate.SkillRoot = dir

	scriptsDir := filepath.Join(dir, "scripts")
	entries, err := os.ReadDir(scriptsDir)
	if err != nil {
		return Gate{}, fmt.Errorf("read scripts dir for %q: %w", ref.Name, err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if isExecutableName(entry.Name()) {
			gate.Scripts = append(gate.Scripts, filepath.Join(scriptsDir, entry.Name()))
		}
	}
	sort.Strings(gate.Scripts)
	return gate, nil
}

// splitFrontmatter splits a Markdown file with leading YAML front matter.
// The file MUST start with `---` and contain a closing `---` on its own line.
// Returns frontmatter (without delimiters) and body.
func splitFrontmatter(s string) (front, body string, err error) {
	s = strings.TrimLeft(s, "\uFEFF") // strip UTF-8 BOM if present
	if !strings.HasPrefix(s, "---\n") && !strings.HasPrefix(s, "---\r\n") {
		return "", "", errors.New("missing leading --- delimiter")
	}
	rest := strings.TrimPrefix(s, "---\n")
	rest = strings.TrimPrefix(rest, "---\r\n")
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return "", "", errors.New("missing closing --- delimiter")
	}
	front = rest[:idx]
	body = rest[idx:]
	body = strings.TrimPrefix(body, "\n---")
	body = strings.TrimPrefix(body, "\r")
	body = strings.TrimPrefix(body, "\n")
	return front, body, nil
}

func isExecutableName(name string) bool {
	// On Unix the bit on disk decides; on Windows we accept by extension.
	switch filepath.Ext(name) {
	case ".sh", ".bash", ".py", ".ps1", ".cmd", ".bat", ".exe", "":
		return true
	}
	return false
}

// specVersionsCompatible returns true iff the gate-pinned spec version is
// compatible with the runner's spec version using minor-version-tolerant
// comparison. Spec §4.2.
func specVersionsCompatible(gatePin, runnerVersion string) bool {
	gp := strings.SplitN(gatePin, ".", 3)
	rv := strings.SplitN(runnerVersion, ".", 3)
	if len(gp) < 2 || len(rv) < 2 {
		return gatePin == runnerVersion
	}
	return gp[0] == rv[0] && gp[1] == rv[1]
}
