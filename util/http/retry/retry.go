package retry

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/attic-labs/noms/d"
)

var (
	sleepFn = time.Sleep
)

func Request(url string, requestFn func() (*http.Response, error)) *http.Response {
	retries := []time.Duration{100 * time.Millisecond, 2 * time.Second, 5 * time.Second}

	getBody := func(resp *http.Response) string {
		b := &bytes.Buffer{}
		_, err := io.Copy(b, resp.Body)
		d.Chk.NoError(err)
		return b.String()
	}

	for i := 0; i <= len(retries); i++ {
		resp, err := requestFn()
		var body string
		var code int
		if err == nil {
			if class := code / 100; class != 4 && class != 5 {
				return resp
			}
			body = getBody(resp)
			code = resp.StatusCode
		}

		if i < len(retries) {
			dur := retries[i]
			fmt.Printf("Failed to fetch %s on attempt #%d, code %d, body %s, err %s. Trying again in %s.\n",
				url, i, code, body, err, dur.String())
			sleepFn(dur)
		} else {
			d.Chk.Fail(fmt.Sprintf("Failed to fetch %s on final attempt #%d, code %d, body %s, err %s. Goodbye.\n",
			url, len(retries), code, body, err))
		}
	}

	// We'll never get here
	d.Chk.Fail("Should not reach here")
	return nil
}
