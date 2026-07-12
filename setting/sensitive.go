package setting

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
)

const (
	MaxSensitiveOutputRegexRules      = 64
	MaxSensitiveOutputRegexRuleBytes  = 512
	MaxSensitiveOutputRegexTotalBytes = 16 << 10
)

var CheckSensitiveEnabled = true
var CheckSensitiveOnPromptEnabled = true
var CheckSensitiveOnOutputEnabled = false
var ConversationLogEnabled = false

// StopOnSensitiveEnabled 如果检测到敏感词，是否立刻停止生成，否则替换敏感词
var StopOnSensitiveEnabled = true

// StreamCacheQueueLength 流模式缓存队列长度，0表示无缓存
var StreamCacheQueueLength = 0

// SensitiveWords 敏感词
var SensitiveWords = []string{
	"test_sensitive",
}

type CompiledSensitiveOutputRegexRule struct {
	Pattern string
	Regex   *regexp.Regexp
}

var (
	sensitiveOutputRegexRulesMu  sync.RWMutex
	SensitiveOutputRegexRules    []string
	sensitiveOutputRegexSnapshot atomic.Value
)

func init() {
	sensitiveOutputRegexSnapshot.Store([]CompiledSensitiveOutputRegexRule{})
}

func SensitiveWordsToString() string {
	return strings.Join(SensitiveWords, "\n")
}

func SensitiveWordsFromString(s string) {
	SensitiveWords = []string{}
	sw := strings.Split(s, "\n")
	for _, w := range sw {
		w = strings.TrimSpace(w)
		if w != "" {
			SensitiveWords = append(SensitiveWords, w)
		}
	}
}

func SensitiveOutputRegexRulesToString() string {
	sensitiveOutputRegexRulesMu.RLock()
	defer sensitiveOutputRegexRulesMu.RUnlock()
	return strings.Join(SensitiveOutputRegexRules, "\n")
}

func ValidateSensitiveOutputRegexRulesString(s string) error {
	_, _, err := compileSensitiveOutputRegexRules(s)
	return err
}

func SensitiveOutputRegexRulesFromString(s string) error {
	rules, compiled, err := compileSensitiveOutputRegexRules(s)
	if err != nil {
		return err
	}
	sensitiveOutputRegexRulesMu.Lock()
	SensitiveOutputRegexRules = rules
	sensitiveOutputRegexRulesMu.Unlock()
	sensitiveOutputRegexSnapshot.Store(compiled)
	return nil
}

func GetCompiledSensitiveOutputRegexRules() []CompiledSensitiveOutputRegexRule {
	return sensitiveOutputRegexSnapshot.Load().([]CompiledSensitiveOutputRegexRule)
}

func compileSensitiveOutputRegexRules(s string) ([]string, []CompiledSensitiveOutputRegexRule, error) {
	if len(s) > MaxSensitiveOutputRegexTotalBytes {
		return nil, nil, fmt.Errorf("sensitive output regex configuration exceeds %d bytes", MaxSensitiveOutputRegexTotalBytes)
	}
	rules := make([]string, 0)
	compiled := make([]CompiledSensitiveOutputRegexRule, 0)
	seen := make(map[string]struct{})
	for lineNumber, raw := range strings.Split(s, "\n") {
		rule := strings.TrimSpace(raw)
		if rule == "" {
			continue
		}
		if len(rule) > MaxSensitiveOutputRegexRuleBytes {
			return nil, nil, fmt.Errorf("sensitive output regex rule %d exceeds %d bytes", lineNumber+1, MaxSensitiveOutputRegexRuleBytes)
		}
		if _, exists := seen[rule]; exists {
			continue
		}
		if len(rules) >= MaxSensitiveOutputRegexRules {
			return nil, nil, fmt.Errorf("sensitive output regex configuration exceeds %d rules", MaxSensitiveOutputRegexRules)
		}
		re, err := regexp.Compile(rule)
		if err != nil {
			return nil, nil, fmt.Errorf("sensitive output regex rule %d is invalid: %w", lineNumber+1, err)
		}
		seen[rule] = struct{}{}
		rules = append(rules, rule)
		compiled = append(compiled, CompiledSensitiveOutputRegexRule{Pattern: rule, Regex: re})
	}
	return rules, compiled, nil
}

func ShouldCheckPromptSensitive() bool {
	return CheckSensitiveEnabled && CheckSensitiveOnPromptEnabled
}

func ShouldCheckOutputSensitive() bool {
	return CheckSensitiveEnabled && CheckSensitiveOnOutputEnabled
}
