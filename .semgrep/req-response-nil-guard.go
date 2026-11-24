// Test fixture for the trident-req-response-nil-guard rule, run by
// `semgrep --test .semgrep/`. Not compiled: Go tooling skips dot directories.
// `// ruleid:` marks the line after it as an expected finding, `// ok:` as one
// that must not match.
package fixtures

import "github.com/imroc/req/v3"

func embeddedFieldWithoutGuard(resp *req.Response) int {
	// ruleid: trident-req-response-nil-guard
	return resp.Response.StatusCode
}

func embeddedHeaderWithoutGuard(resp *req.Response) string {
	// ruleid: trident-req-response-nil-guard
	return resp.Response.Header.Get("Location")
}

func embeddedFieldInsideGuard(resp *req.Response) int {
	if resp.Response != nil {
		// ok: trident-req-response-nil-guard
		return resp.Response.StatusCode
	}
	return 0
}

func embeddedFieldInElseOfNilCheck(resp *req.Response) int {
	if resp.Response == nil {
		return 0
	} else {
		// ok: trident-req-response-nil-guard
		return resp.Response.StatusCode
	}
}

// A service reads the promoted field after err == nil: the embedded response
// is always set there, so the rule leaves it alone outside hooks.
func promotedFieldAfterSuccess(c *req.Client) int {
	resp, err := c.R().Get("https://example.com/")
	if err != nil {
		return 0
	}
	// ok: trident-req-response-nil-guard
	return resp.StatusCode
}

func hookWithoutGuard(c *req.Client) {
	c.OnAfterResponse(func(_ *req.Client, resp *req.Response) error {
		// ruleid: trident-req-response-nil-guard
		if resp.StatusCode >= 400 {
			return nil
		}
		// ruleid: trident-req-response-nil-guard
		_ = resp.Header.Get("Content-Type")
		return nil
	})
}

func hookWithNamedClientParameter(c *req.Client) {
	c.OnAfterResponse(func(client *req.Client, resp *req.Response) error {
		_ = client
		// ruleid: trident-req-response-nil-guard
		_ = resp.StatusCode
		return nil
	})
}

func hookWithEarlyReturn(c *req.Client) {
	c.OnAfterResponse(func(_ *req.Client, resp *req.Response) error {
		if resp.Response == nil {
			return nil
		}
		// ok: trident-req-response-nil-guard
		_ = resp.StatusCode
		// ok: trident-req-response-nil-guard
		_ = resp.Header
		return nil
	})
}

func hookInsideGuard(c *req.Client) {
	c.OnAfterResponse(func(_ *req.Client, resp *req.Response) error {
		if resp.Response != nil {
			// ok: trident-req-response-nil-guard
			_ = resp.StatusCode
		}
		return nil
	})
}
