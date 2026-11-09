package security

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
)

// RedactedRecord holds details about a sanitized secret
type RedactedRecord struct {
	Category      string `json:"category"`       // e.g., "ai_key", "server_key", "db_credential", "private_key", "env_secret", "custom_keyword"
	Alias         string `json:"alias"`          // e.g., "[ALIASED_OPENAI_KEY_1]", "[ALIASED_CUSTOM_KEY_1]"
	MaskedPreview string `json:"masked_preview"` // e.g., "sk-proj-••••••••abcd"
}

// SanitizeAudit contains the audit trail of sanitization
type SanitizeAudit struct {
	RedactedCount       int              `json:"redacted_count"`
	CustomMatches       int              `json:"custom_matches"`
	BuiltInMatches      int              `json:"built_in_matches"`
	Records             []RedactedRecord `json:"records"`
	Message             string           `json:"message"`
	SafePreview         string           `json:"safe_preview,omitempty"`
	DoubleSecurityLevel string           `json:"double_security_level"` // "ACTIVE_ENFORCED"
}

type patternDef struct {
	category    string
	aliasPrefix string
	regex       *regexp.Regexp
}

// CustomKeywordRule represents a runtime configurable censored keyword or regex
type CustomKeywordRule struct {
	ID          uint           `json:"id"`
	Keyword     string         `json:"keyword"`
	Category    string         `json:"category"`
	Description string         `json:"description"`
	IsRegex     bool           `json:"is_regex"`
	IsActive    bool           `json:"is_active"`
	Compiled    *regexp.Regexp `json:"-"`
}

// Sanitizer provides high-performance, deterministic censorship of sensitive keys
type Sanitizer struct {
	mu             sync.RWMutex
	patterns       []patternDef
	customKeywords []CustomKeywordRule
}

// GlobalSanitizer is the singleton instance applied globally across all AI operations
var GlobalSanitizer *Sanitizer

func init() {
	GlobalSanitizer = NewSanitizer()
}

// NewSanitizer creates and compiles all secret detection rules
func NewSanitizer() *Sanitizer {
	s := &Sanitizer{
		patterns:       make([]patternDef, 0),
		customKeywords: make([]CustomKeywordRule, 0),
	}

	// 1. Private Keys (RSA, OpenSSH, EC, PGP)
	s.addPattern("private_key", "PRIVATE_KEY_BLOCK", regexp.MustCompile(`(?s)-----BEGIN [A-Z ]*PRIVATE KEY-----.*?-----END [A-Z ]*PRIVATE KEY-----`))

	// 2. AI Provider Keys
	s.addPattern("ai_key", "OPENAI_KEY", regexp.MustCompile(`\bsk-proj-[a-zA-Z0-9_\-]{20,}\b`))
	s.addPattern("ai_key", "OPENAI_KEY", regexp.MustCompile(`\bsk-[a-zA-Z0-9]{30,}\b`))
	s.addPattern("ai_key", "ANTHROPIC_KEY", regexp.MustCompile(`\bsk-ant-[a-zA-Z0-9_\-]{20,}\b`))
	s.addPattern("ai_key", "GEMINI_GOOGLE_KEY", regexp.MustCompile(`\bAIza[0-9A-Za-z\-_]{30,45}\b`))
	s.addPattern("ai_key", "HUGGINGFACE_KEY", regexp.MustCompile(`\bhf_[a-zA-Z0-9]{30,}\b`))
	s.addPattern("ai_key", "GROQ_KEY", regexp.MustCompile(`\bgsk_[a-zA-Z0-9]{25,}\b`))
	s.addPattern("ai_key", "OPENROUTER_KEY", regexp.MustCompile(`\bsk-or-v1-[a-zA-Z0-9]{35,}\b`))
	s.addPattern("ai_key", "GENERIC_API_KEY", regexp.MustCompile(`\b[a-zA-Z0-9_-]{32,64}\b`)) // For high-entropy keys

	// 3. Cloud & Server Credentials
	s.addPattern("server_key", "AWS_ACCESS_KEY", regexp.MustCompile(`\b(AKIA|ABIA|ACCA|ASIA)[0-9A-Z]{16}\b`))
	s.addPattern("server_key", "GITHUB_TOKEN", regexp.MustCompile(`\bgh[pousr]_[A-Za-z0-9_]{30,}\b`))
	s.addPattern("server_key", "GITHUB_PAT", regexp.MustCompile(`\bgithub_pat_[A-Za-z0-9_]{40,}\b`))
	s.addPattern("server_key", "GITLAB_TOKEN", regexp.MustCompile(`\bglpat-[0-9a-zA-Z\-_]{15,}\b`))
	s.addPattern("server_key", "SLACK_TOKEN", regexp.MustCompile(`\bxox[baprs]-[0-9a-zA-Z]{10,}\b`))
	s.addPattern("server_key", "SLACK_WEBHOOK", regexp.MustCompile(`https:\/\/hooks\.slack\.com\/services\/[A-Za-z0-9_\/\-]+`))
	s.addPattern("server_key", "STRIPE_KEY", regexp.MustCompile(`\b(sk|pk|rk)_(live|test)_[0-9a-zA-Z]{20,}\b`))

	// 4. JWT & Bearer Tokens
	s.addPattern("token", "JWT_TOKEN", regexp.MustCompile(`\beyJ[a-zA-Z0-9_\-]{10,}\.eyJ[a-zA-Z0-9_\-]{10,}\.[a-zA-Z0-9_\-]+\b`))
	s.addPattern("token", "BEARER_TOKEN", regexp.MustCompile(`(?i)\bBearer\s+([a-zA-Z0-9\-_\.]{20,})\b`))

	// 5. Database Connection URLs with user:password
	s.addPattern("db_credential", "DATABASE_URI_CREDENTIAL", regexp.MustCompile(`(?i)((postgres|postgresql|mysql|mongodb|mongodb\+srv|redis|rediss|amqp|couchdb):\/\/)([^:\s\/]+):([^@\s\/]+)@`))

	// 6. Generic Environment & Config Assignments (e.g. FOO_KEY: "secret", DB_PASS=secret)
	s.addPattern("env_secret", "CONFIG_SECRET", regexp.MustCompile(`(?i)\b([A-Za-z0-9_]*(KEY|SECRET|PASSWORD|PASSWD|TOKEN|AUTH|CREDENTIAL|PRIVATE))\b\s*([:=])\s*["']?([^"'\r\n\s]{6,})["']?`))

	return s
}

func (s *Sanitizer) addPattern(category, aliasPrefix string, re *regexp.Regexp) {
	s.patterns = append(s.patterns, patternDef{
		category:    category,
		aliasPrefix: aliasPrefix,
		regex:       re,
	})
}

// SetCustomKeywords updates the custom censored keywords list atomically
func (s *Sanitizer) SetCustomKeywords(rules []CustomKeywordRule) {
	s.mu.Lock()
	defer s.mu.Unlock()

	compiledRules := make([]CustomKeywordRule, 0, len(rules))
	for _, r := range rules {
		if !r.IsActive || strings.TrimSpace(r.Keyword) == "" {
			compiledRules = append(compiledRules, r)
			continue
		}
		rule := r
		if rule.IsRegex {
			if re, err := regexp.Compile(rule.Keyword); err == nil {
				rule.Compiled = re
			}
		} else {
			// Literal case-insensitive word or phrase regex
			escaped := regexp.QuoteMeta(rule.Keyword)
			if re, err := regexp.Compile("(?i)" + escaped); err == nil {
				rule.Compiled = re
			}
		}
		compiledRules = append(compiledRules, rule)
	}
	s.customKeywords = compiledRules
}

// GetCustomKeywords returns a copy of current custom keyword rules
func (s *Sanitizer) GetCustomKeywords() []CustomKeywordRule {
	s.mu.RLock()
	defer s.mu.RUnlock()

	res := make([]CustomKeywordRule, len(s.customKeywords))
	copy(res, s.customKeywords)
	return res
}

const aliasedMarker = "[ALIASED_"

var (
	dbURIRegex = regexp.MustCompile(`(?i)((postgres|postgresql|mysql|mongodb|mongodb\+srv|redis|rediss|amqp|couchdb):\/\/)([^:\s\/]+):([^@\s\/]+)@`)
	envRegex   = regexp.MustCompile(`(?i)\b([A-Za-z0-9_]*(KEY|SECRET|PASSWORD|PASSWD|TOKEN|AUTH|CREDENTIAL|PRIVATE))\b\s*([:=])\s*["']?([^"'\r\n\s]{6,})["']?`)
)

// sanitizeRun = keadaan satu kali Sanitize: teks hasil, peta alias, dan catatan audit.
type sanitizeRun struct {
	result        string
	aliasMap      map[string]string
	aliasCounter  map[string]int
	records       []RedactedRecord
	customMatches int
}

func maskPreview(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if len(trimmed) <= 8 {
		return "••••••••"
	}
	prefix := 4
	suffix := 4
	if len(trimmed) < 14 {
		prefix = 2
		suffix = 2
	}
	return fmt.Sprintf("%s••••••••%s", trimmed[:prefix], trimmed[len(trimmed)-suffix:])
}

func (r *sanitizeRun) getAlias(raw, prefix, cat string) string {
	if alias, exists := r.aliasMap[raw]; exists {
		return alias
	}
	r.aliasCounter[prefix]++
	alias := fmt.Sprintf("[ALIASED_%s_%d]", prefix, r.aliasCounter[prefix])
	r.aliasMap[raw] = alias

	r.records = append(r.records, RedactedRecord{
		Category:      cat,
		Alias:         alias,
		MaskedPreview: maskPreview(raw),
	})
	return alias
}

// customKeywordsPass: Custom Censored Keywords & User Rules (First Line of Defense).
func (r *sanitizeRun) customKeywordsPass(keywords []CustomKeywordRule) {
	for _, ck := range keywords {
		if !ck.IsActive || ck.Compiled == nil {
			continue
		}
		prefix := "CUSTOM_KEY"
		if ck.Category != "" && ck.Category != "custom" {
			prefix = strings.ToUpper(ck.Category)
		}
		cat := "custom_keyword"
		if ck.Category != "" {
			cat = ck.Category
		}

		r.result = ck.Compiled.ReplaceAllStringFunc(r.result, func(m string) string {
			if strings.HasPrefix(m, aliasedMarker) {
				return m
			}
			r.customMatches++
			return r.getAlias(m, prefix, cat)
		})
	}
}

// dbURIPass: Database URIs first (special multi-group).
func (r *sanitizeRun) dbURIPass() {
	r.result = dbURIRegex.ReplaceAllStringFunc(r.result, func(m string) string {
		sub := dbURIRegex.FindStringSubmatch(m)
		if len(sub) != 5 {
			return m
		}
		aliasedUser := r.getAlias(sub[3], "DB_USER", "db_credential")
		aliasedPass := r.getAlias(sub[4], "DB_PASSWORD", "db_credential")
		return fmt.Sprintf("%s%s:%s@", sub[1], aliasedUser, aliasedPass)
	})
}

// envPass: Env & Config assignments.
func (r *sanitizeRun) envPass() {
	r.result = envRegex.ReplaceAllStringFunc(r.result, func(m string) string {
		sub := envRegex.FindStringSubmatch(m)
		if len(sub) < 5 {
			return m
		}
		varKey, delim, val := sub[1], sub[3], sub[4]

		// Don't mask already aliased keys or common false positives
		if strings.HasPrefix(val, aliasedMarker) || val == "true" || val == "false" || val == "null" {
			return m
		}
		alias := r.getAlias(val, "CONFIG_SECRET", "env_secret")
		return fmt.Sprintf("%s%s \"%s\"", varKey, delim, alias)
	})
}

func (r *sanitizeRun) bearerPass(p patternDef) {
	r.result = p.regex.ReplaceAllStringFunc(r.result, func(m string) string {
		parts := strings.SplitN(m, " ", 2)
		if len(parts) != 2 {
			return m
		}
		token := strings.TrimSpace(parts[1])
		if strings.HasPrefix(token, aliasedMarker) {
			return m
		}
		return fmt.Sprintf("%s %s", parts[0], r.getAlias(token, "BEARER_TOKEN", p.category))
	})
}

// patternsPass: all direct pattern matches.
func (r *sanitizeRun) patternsPass(patterns []patternDef) {
	for _, p := range patterns {
		switch p.aliasPrefix {
		case "DATABASE_URI_CREDENTIAL", "CONFIG_SECRET":
			continue // Already processed with submatches
		case "BEARER_TOKEN":
			r.bearerPass(p)
			continue
		}
		r.result = p.regex.ReplaceAllStringFunc(r.result, func(m string) string {
			if strings.HasPrefix(m, aliasedMarker) {
				return m
			}
			return r.getAlias(m, p.aliasPrefix, p.category)
		})
	}
}

func (r *sanitizeRun) audit() *SanitizeAudit {
	builtInMatches := len(r.records) - r.customMatches
	if builtInMatches < 0 {
		builtInMatches = 0
	}
	audit := &SanitizeAudit{
		RedactedCount:       len(r.records),
		CustomMatches:       r.customMatches,
		BuiltInMatches:      builtInMatches,
		Records:             r.records,
		DoubleSecurityLevel: "ACTIVE_ENFORCED",
		Message:             "Zero secrets detected. Payload is clean and double-security verified.",
	}
	if len(r.records) == 0 {
		return audit
	}
	audit.Message = fmt.Sprintf("Double Security Enforced: Censored %d secret(s) (%d custom keyword match, %d built-in pattern match).",
		len(r.records), r.customMatches, builtInMatches)
	preview := r.result
	if len(preview) > 350 {
		preview = preview[:350] + "..."
	}
	audit.SafePreview = preview
	return audit
}

// Sanitize scans a text, redacts sensitive keys into safe aliases, and returns sanitized text and audit
func (s *Sanitizer) Sanitize(input string) (string, *SanitizeAudit) {
	if strings.TrimSpace(input) == "" {
		return input, &SanitizeAudit{
			RedactedCount: 0,
			Records:       []RedactedRecord{},
			Message:       "No secrets detected",
		}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	run := &sanitizeRun{
		result:       input,
		aliasMap:     make(map[string]string),
		aliasCounter: make(map[string]int),
		records:      make([]RedactedRecord, 0),
	}
	run.customKeywordsPass(s.customKeywords)
	run.dbURIPass()
	run.envPass()
	run.patternsPass(s.patterns)
	return run.result, run.audit()
}

// SanitizeText returns only the sanitized text string
func SanitizeText(input string) string {
	clean, _ := GlobalSanitizer.Sanitize(input)
	return clean
}

// GetBuiltInPatterns returns the list of active built-in pattern descriptions
func (s *Sanitizer) GetBuiltInPatterns() []string {
	return []string{
		"OpenAI API Keys (sk-..., sk-proj-...)",
		"Anthropic API Keys (sk-ant-...)",
		"Google Gemini / Cloud Keys (AIza...)",
		"HuggingFace Access Tokens (hf_...)",
		"Groq & OpenRouter Keys (gsk_..., sk-or-v1-...)",
		"AWS Access Keys (AKIA..., ASIA...)",
		"GitHub & GitLab Tokens (ghp_..., github_pat_..., glpat-...)",
		"Slack Bot Tokens & Webhooks (xox..., hooks.slack.com)",
		"Stripe Secret & Live Keys (sk_live_..., rk_live_...)",
		"JWT & Bearer Tokens (eyJ..., Bearer ...)",
		"Database URIs with Passwords (postgres, mysql, mongodb, redis, amqp)",
		"Environment & Config Secret Assignments (KEY, SECRET, PASS, TOKEN)",
		"Private Key Envelopes (RSA, OpenSSH, EC, PGP)",
	}
}
