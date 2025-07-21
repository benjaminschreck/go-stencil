package stencil

import (
	"regexp"
	"strings"
)

// TokenType represents the type of a template token
type TokenType int

const (
	TokenText TokenType = iota
	TokenVariable
	TokenIf
	TokenElse
	TokenElsif
	TokenUnless
	TokenFor
	TokenEnd
	TokenPageBreak
	TokenInclude
)

// Token represents a parsed template token
type Token struct {
	Type  TokenType
	Value string
}

var (
	// Regular expression to match template tokens
	tokenRegex = regexp.MustCompile(`\{\{([^}]*)\}\}`)
)

// Tokenize parses a template string into tokens
func Tokenize(input string) []Token {
	var tokens []Token
	lastEnd := 0

	logger := GetLogger()
	if logger.IsDebugMode() {
		logger.WithField("input_length", len(input)).Debug("Starting tokenization")
	}

	matches := tokenRegex.FindAllStringSubmatchIndex(input, -1)
	
	for _, match := range matches {
		// Add any text before this token
		if match[0] > lastEnd {
			tokens = append(tokens, Token{
				Type:  TokenText,
				Value: input[lastEnd:match[0]],
			})
		}

		// Extract the content between {{ and }}
		content := strings.TrimSpace(input[match[2]:match[3]])
		
		// Skip empty tokens
		if content == "" {
			tokens = append(tokens, Token{
				Type:  TokenText,
				Value: input[match[0]:match[1]],
			})
		} else {
			// Parse the token type
			token := parseToken(content)
			if logger.IsDebugMode() {
				logger.WithFields(Fields{
					"type":    token.Type,
					"content": content,
				}).Debug("Found token")
			}
			tokens = append(tokens, token)
		}

		lastEnd = match[1]
	}

	// Add any remaining text
	if lastEnd < len(input) {
		tokens = append(tokens, Token{
			Type:  TokenText,
			Value: input[lastEnd:],
		})
	}

	if logger.IsDebugMode() {
		logger.WithField("token_count", len(tokens)).Debug("Tokenization complete")
	}

	return tokens
}

// parseToken determines the type of token from its content
func parseToken(content string) Token {
	// Split the content into words
	parts := strings.Fields(content)
	if len(parts) == 0 {
		return Token{Type: TokenText, Value: "{{" + content + "}}"}
	}

	keyword := parts[0]
	
	switch keyword {
	case "if":
		return Token{
			Type:  TokenIf,
			Value: strings.TrimSpace(strings.TrimPrefix(content, "if")),
		}
	case "else":
		return Token{
			Type:  TokenElse,
			Value: "",
		}
	case "elsif", "elseif", "elif":
		return Token{
			Type:  TokenElsif,
			Value: strings.TrimSpace(strings.TrimPrefix(content, keyword)),
		}
	case "unless":
		return Token{
			Type:  TokenUnless,
			Value: strings.TrimSpace(strings.TrimPrefix(content, "unless")),
		}
	case "for":
		return Token{
			Type:  TokenFor,
			Value: strings.TrimSpace(strings.TrimPrefix(content, "for")),
		}
	case "end":
		return Token{
			Type:  TokenEnd,
			Value: "",
		}
	case "pageBreak":
		return Token{
			Type:  TokenPageBreak,
			Value: "",
		}
	case "include":
		return Token{
			Type:  TokenInclude,
			Value: strings.TrimSpace(strings.TrimPrefix(content, "include")),
		}
	default:
		// It's a variable or expression
		return Token{
			Type:  TokenVariable,
			Value: content,
		}
	}
}

// FindTemplateTokens finds all template tokens in a string
// This is a utility function for debugging and analysis
func FindTemplateTokens(input string) []string {
	matches := tokenRegex.FindAllString(input, -1)
	if matches == nil {
		return []string{}
	}
	return matches
}