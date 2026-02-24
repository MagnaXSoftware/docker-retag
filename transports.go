package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type basicAuthTransport struct {
	Wrapped  http.RoundTripper
	URL      string
	Username string
	Password string
}

func (b *basicAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if strings.HasPrefix(req.URL.String(), b.URL) {
		if b.Username != "" || b.Password != "" {
			req.SetBasicAuth(b.Username, b.Password)
		}
	}
	resp, err := b.Wrapped.RoundTrip(req)
	return resp, err
}

type tokenAuthTransport struct {
	Wrapped  http.RoundTripper
	Username string
	Password string
}

func (t *tokenAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.Wrapped.RoundTrip(req)
	if err != nil {
		return resp, err
	}
	if resp != nil && resp.StatusCode == http.StatusUnauthorized {
		// we might need to auth and restart
		challenges := parseAuthHeader(resp.Header)
		var bearerChallenge *authChallenge
		for _, challenge := range challenges {
			if challenge.Scheme == "bearer" {
				bearerChallenge = challenge
				break
			}
		}
		if bearerChallenge == nil {
			return resp, err
		}
		// we have a bearerChallenge
		_ = resp.Body.Close()
		resp, err = t.authAndRetry(&bearerAuthChallenge{
			Realm:   bearerChallenge.Parameters["realm"],
			Service: bearerChallenge.Parameters["service"],
			Scope:   bearerChallenge.Parameters["scope"],
		}, req)

	}
	return resp, err
}

type authToken struct {
	Token string `json:"token"`
}

func (t *tokenAuthTransport) auth(challenge *bearerAuthChallenge) (string, *http.Response, error) {
	realmUrl, err := url.Parse(challenge.Realm)
	if err != nil {
		return "", nil, err
	}

	q := realmUrl.Query()
	q.Set("service", challenge.Service)
	if challenge.Scope != "" {
		q.Set("scope", challenge.Scope)
	}
	realmUrl.RawQuery = q.Encode()

	authRequest, err := http.NewRequest("GET", realmUrl.String(), nil)
	if err != nil {
		return "", nil, err
	}

	if t.Username != "" || t.Password != "" {
		authRequest.SetBasicAuth(t.Username, t.Password)
	}

	client := http.Client{
		Transport: t.Wrapped,
	}

	response, err := client.Do(authRequest)
	if err != nil {
		return "", nil, err
	}

	if response.StatusCode != http.StatusOK {
		return "", response, err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(response.Body)

	var authToken authToken
	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(&authToken)
	if err != nil {
		return "", nil, err
	}

	return authToken.Token, nil, nil
}

func (t *tokenAuthTransport) authAndRetry(challenge *bearerAuthChallenge, req *http.Request) (*http.Response, error) {
	token, authResp, err := t.auth(challenge)
	if err != nil {
		return authResp, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	resp, err := t.Wrapped.RoundTrip(req)
	return resp, err
}

type authChallenge struct {
	Scheme     string
	Parameters map[string]string
}

type bearerAuthChallenge struct {
	Realm   string
	Service string
	Scope   string
}

func parseAuthHeader(header http.Header) []*authChallenge {
	var challenges []*authChallenge
	for _, h := range header[http.CanonicalHeaderKey("WWW-Authenticate")] {
		v, p := parseValueAndParams(h)
		if v != "" {
			challenges = append(challenges, &authChallenge{Scheme: v, Parameters: p})
		}
	}
	return challenges
}

func parseValueAndParams(header string) (value string, params map[string]string) {
	params = make(map[string]string)
	value, s := expectToken(header)
	if value == "" {
		return
	}
	value = strings.ToLower(value)
	s = "," + skipSpace(s)
	for strings.HasPrefix(s, ",") {
		var pkey string
		pkey, s = expectToken(skipSpace(s[1:]))
		if pkey == "" {
			return
		}
		if !strings.HasPrefix(s, "=") {
			return
		}
		var pvalue string
		pvalue, s = expectTokenOrQuoted(s[1:])
		if pvalue == "" {
			return
		}
		pkey = strings.ToLower(pkey)
		params[pkey] = pvalue
		s = skipSpace(s)
	}
	return
}

// Octet types from RFC 2616.
type octetType byte

var octetTypes [256]octetType

const (
	isToken octetType = 1 << iota
	isSpace
)

func init() {
	// OCTET      = <any 8-bit sequence of data>
	// CHAR       = <any US-ASCII character (octets 0 - 127)>
	// CTL        = <any US-ASCII control character (octets 0 - 31) and DEL (127)>
	// CR         = <US-ASCII CR, carriage return (13)>
	// LF         = <US-ASCII LF, linefeed (10)>
	// SP         = <US-ASCII SP, space (32)>
	// HT         = <US-ASCII HT, horizontal-tab (9)>
	// <">        = <US-ASCII double-quote mark (34)>
	// CRLF       = CR LF
	// LWS        = [CRLF] 1*( SP | HT )
	// TEXT       = <any OCTET except CTLs, but including LWS>
	// separators = "(" | ")" | "<" | ">" | "@" | "," | ";" | ":" | "\" | <">
	//              | "/" | "[" | "]" | "?" | "=" | "{" | "}" | SP | HT
	// token      = 1*<any CHAR except CTLs or separators>
	// qdtext     = <any TEXT except <">>

	for c := range 256 {
		var t octetType
		isCtl := c <= 31 || c == 127
		isChar := 0 <= c && c <= 127
		isSeparator := strings.ContainsRune(" \t\"(),/:;<=>?@[]\\{}", rune(c))
		if strings.ContainsRune(" \t\r\n", rune(c)) {
			t |= isSpace
		}
		if isChar && !isCtl && !isSeparator {
			t |= isToken
		}
		octetTypes[c] = t
	}
}

// skipSpace returns a slice of s starting at the first non-space character.
func skipSpace(s string) (rest string) {
	i := 0
	for ; i < len(s); i++ {
		if octetTypes[s[i]]&isSpace == 0 {
			break
		}
	}
	return s[i:]
}

// expectToken splits s into a token and everything after (the rest).
func expectToken(s string) (token, rest string) {
	i := 0
	for ; i < len(s); i++ {
		if octetTypes[s[i]]&isToken == 0 {
			break
		}
	}
	return s[:i], s[i:]
}

// expectTokenOrQuoted splits s into an optionally-quoted token and everything after (the rest).
func expectTokenOrQuoted(s string) (value string, rest string) {
	if !strings.HasPrefix(s, "\"") {
		return expectToken(s)
	}
	s = s[1:]
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '"':
			return s[:i], s[i+1:]
		case '\\':
			p := make([]byte, len(s)-1)
			j := copy(p, s[:i])
			escape := true
			for i += i; i < len(s); i++ {
				b := s[i]
				switch {
				case escape:
					escape = false
					p[j] = b
					j++
				case b == '\\':
					escape = true
				case b == '"':
					return string(p[:j]), s[i+1:]
				default:
					p[j] = b
					j++
				}
			}
			return "", ""
		}
	}
	return "", ""
}
