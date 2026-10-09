package gologger

import (
	"regexp"
	"strings"
)

// Credentials reach the logs from two directions: cloud SDKs echo
// their own request URLs into error strings (Aliyun puts the whole signed
// query, including AccessKeyId and Signature, into url.Error), and RedC itself
// logs generated instance passwords. Those logs are persisted to
// task-result/*/logs/*.log, so anything printed here is a durable copy of a
// live credential.
//
// Redact rewrites the well-known credential shapes before they are formatted.
// It is deliberately conservative: it only touches values that sit behind a
// recognisable key name or match a known access-key prefix, so ordinary log
// lines are left byte-identical.

// keyValueSecret matches `key = value`, `key: value`, `"key":"value"` and
// `key=value`, for key names that denote credentials.
var keyValueSecret = regexp.MustCompile(
	`(?i)([a-z0-9_.\-]*(?:access[_-]?key[_-]?(?:id|secret)?|secret[_-]?(?:id|key)?|` +
		`client[_-]?secret|api[_-]?key|apikey|auth[_-]?token|access[_-]?token|` +
		`instance[_-]?password|admin[_-]?password|passwd|password|token))` +
		`("?\s*[:=]\s*"?)` +
		`([^"'\s&,;}\]]{4,})`,
)

// signatureValue matches the signed-query parameters Aliyun/AWS embed in URLs.
var signatureValue = regexp.MustCompile(
	`(?i)(X-Amz-(?:Credential|Signature|Security-Token)|Signature|SignatureNonce)(=)` +
		`([^&"'\s]{4,})`,
)

// bareKey matches provider access-key IDs that are not behind a key name (for
// example when a secret is pasted into a userdata script).
var bareKey = regexp.MustCompile(
	`\b(?:AKIA|ASIA|AGPA|AIDA|AROA)[0-9A-Z]{12,}\b|` +
		`\bLTAI[0-9A-Za-z]{8,}\b|` +
		`\bAKID[0-9A-Za-z]{10,}\b|` +
		`\bAKLT[0-9A-Za-z+/=]{20,}`,
)

// redactionPlaceholder replaces every matched secret value. Values are never
// partially preserved: a masked suffix is still a credential disclosure.
const redactionPlaceholder = "***"

// Redact returns s with credential values replaced by a fixed placeholder.
func Redact(s string) string {
	if s == "" {
		return s
	}
	// Cheap pre-filter: nothing to do for the overwhelming majority of lines.
	if !strings.ContainsAny(s, "=:") && !bareKey.MatchString(s) {
		return s
	}
	s = keyValueSecret.ReplaceAllString(s, "${1}${2}"+redactionPlaceholder)
	s = signatureValue.ReplaceAllString(s, "${1}${2}"+redactionPlaceholder)
	s = bareKey.ReplaceAllString(s, redactionPlaceholder)
	return s
}
