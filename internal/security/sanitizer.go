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

// Sanitize scans a text, redacts sensitive keys into safe aliases, and returns sanitized text and audit
func (s *Sanitizer) Sanitize(input string) (string, *SanitizeAudit) {
	if strings.TrimSpace(input) == "" {
		return input, &SanitizeAudit{
			RedactedCount: 0,
			Records:       []RedactedRecord{},
			Message:       "No secrets detected",
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	result := input
	aliasMap := make(map[string]string)
	aliasCounter := make(map[string]int)
	records := make([]RedactedRecord, 0)

	maskPreview := func(raw string) string {
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

	getAlias := func(raw, prefix, cat string) string {
		if alias, exists := aliasMap[raw]; exists {
			return alias
		}
		aliasCounter[prefix]++
		alias := fmt.Sprintf("[ALIASED_%s_%d]", prefix, aliasCounter[prefix])
		aliasMap[raw] = alias

		records = append(records, RedactedRecord{
			Category:      cat,
			Alias:         alias,
			MaskedPreview: maskPreview(raw),
		})
		return alias
	}

	// PASS 1: Custom Censored Keywords & User Rules (First Line of Defense)
	customMatchesCount := 0
	for _, ck := range s.customKeywords {
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

		result = ck.Compiled.ReplaceAllStringFunc(result, func(m string) string {
			if strings.HasPrefix(m, "[ALIASED_") {
				return m
			}
			customMatchesCount++
			return getAlias(m, prefix, cat)
		})
	}

	// PASS 2: Built-in Deep Threat Patterns
	// 1. Process Database URIs first (special multi-group)
	dbRegex := regexp.MustCompile(`(?i)((postgres|postgresql|mysql|mongodb|mongodb\+srv|redis|rediss|amqp|couchdb):\/\/)([^:\s\/]+):([^@\s\/]+)@`)
	result = dbRegex.ReplaceAllStringFunc(result, func(m string) string {
		sub := dbRegex.FindStringSubmatch(m)
		if len(sub) == 5 {
			scheme := sub[1]
			user := sub[3]
			pass := sub[4]
			aliasedUser := getAlias(user, "DB_USER", "db_credential")
			aliasedPass := getAlias(pass, "DB_PASSWORD", "db_credential")
			return fmt.Sprintf("%s%s:%s@", scheme, aliasedUser, aliasedPass)
		}
		return m
	})

	// 2. Process Env & Config assignments
	envRegex := regexp.MustCompile(`(?i)\b([A-Za-z0-9_]*(KEY|SECRET|PASSWORD|PASSWD|TOKEN|AUTH|CREDENTIAL|PRIVATE))\b\s*([:=])\s*["']?([^"'\r\n\s]{6,})["']?`)
	result = envRegex.ReplaceAllStringFunc(result, func(m string) string {
		sub := envRegex.FindStringSubmatch(m)
		if len(sub) >= 5 {
			varKey := sub[1]
			delim := sub[3]
			val := sub[4]

			// Don't mask already aliased keys or common false positives
			if strings.HasPrefix(val, "[ALIASED_") || val == "true" || val == "false" || val == "null" {
				return m
			}

			alias := getAlias(val, "CONFIG_SECRET", "env_secret")
			return fmt.Sprintf("%s%s \"%s\"", varKey, delim, alias)
		}
		return m
	})

	// 3. Process all direct pattern matches
	for _, p := range s.patterns {
		if p.aliasPrefix == "DATABASE_URI_CREDENTIAL" || p.aliasPrefix == "CONFIG_SECRET" {
			continue // Already processed with submatches
		}

		if p.aliasPrefix == "BEARER_TOKEN" {
			result = p.regex.ReplaceAllStringFunc(result, func(m string) string {
				parts := strings.SplitN(m, " ", 2)
				if len(parts) == 2 {
					token := strings.TrimSpace(parts[1])
					if strings.HasPrefix(token, "[ALIASED_") {
						return m
					}
					alias := getAlias(token, "BEARER_TOKEN", p.category)
					return fmt.Sprintf("%s %s", parts[0], alias)
				}
				return m
			})
			continue
		}

		result = p.regex.ReplaceAllStringFunc(result, func(m string) string {
			if strings.HasPrefix(m, "[ALIASED_") {
				return m
			}
			return getAlias(m, p.aliasPrefix, p.category)
		})
	}

	builtInMatches := len(records) - customMatchesCount
	if builtInMatches < 0 {
		builtInMatches = 0
	}

	audit := &SanitizeAudit{
		RedactedCount:       len(records),
		CustomMatches:       customMatchesCount,
		BuiltInMatches:      builtInMatches,
		Records:             records,
		DoubleSecurityLevel: "ACTIVE_ENFORCED",
		Message: func() string {
			if len(records) == 0 {
				return "Zero secrets detected. Payload is clean and double-security verified."
			}
			return fmt.Sprintf("Double Security Enforced: Censored %d secret(s) (%d custom keyword match, %d built-in pattern match).",
				len(records), customMatchesCount, builtInMatches)
		}(),
	}

	if len(records) > 0 {
		preview := result
		if len(preview) > 350 {
			preview = preview[:350] + "..."
		}
		audit.SafePreview = preview
	}

	return result, audit
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
