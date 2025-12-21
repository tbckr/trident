package threatminer_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/imroc/req/v3"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	tridenthttp "github.com/tbckr/trident/internal/http"
	"github.com/tbckr/trident/internal/services/threatminer"
)

func TestThreatMiner(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	client := req.C()
	httpmock.ActivateNonDefault(client.GetClient())
	defer httpmock.DeactivateAndReset()

	httpClient := tridenthttp.NewClientWithReqClient(logger, client)
	service := threatminer.NewService(httpClient, logger, nil)

	t.Run("LookupDomain", func(t *testing.T) {
		httpmock.RegisterResponder("GET", "https://api.threatminer.org/v2/domain.php?q=example.com&rt=1",
			httpmock.NewStringResponder(200, `{"status":"200","status_message":"Success","results":["1.2.3.4"]}`))

		result, err := service.LookupDomain(context.Background(), "example.com", 1)
		assert.NoError(t, err)
		assert.Equal(t, "200", result.Status)
		assert.Equal(t, []string{"1.2.3.4"}, result.Results)
	})

	t.Run("LookupIP", func(t *testing.T) {
		httpmock.RegisterResponder("GET", "https://api.threatminer.org/v2/host.php?q=1.2.3.4&rt=1",
			httpmock.NewStringResponder(200, `{"status":"200","status_message":"Success","results":["example.com"]}`))

		result, err := service.LookupIP(context.Background(), "1.2.3.4", 1)
		assert.NoError(t, err)
		assert.Equal(t, "200", result.Status)
		assert.Equal(t, []string{"example.com"}, result.Results)
	})

	t.Run("LookupHash", func(t *testing.T) {
		httpmock.RegisterResponder("GET", "https://api.threatminer.org/v2/sample.php?q=hash123&rt=1",
			httpmock.NewStringResponder(200, `{"status":"200","status_message":"Success","results":[{"sample_id":"123","file_name":"malware.exe"}]}`))

		result, err := service.LookupHash(context.Background(), "hash123", 1)
		assert.NoError(t, err)
		assert.Equal(t, "200", result.Status)
		assert.Equal(t, "malware.exe", result.Results[0].FileName)
	})

	t.Run("Context Cancellation", func(t *testing.T) {
		httpmock.RegisterResponder("GET", "https://api.threatminer.org/v2/domain.php?q=cancelled.com&rt=1",
			httpmock.NewStringResponder(200, `{"status":"200"}`).Delay(100*time.Millisecond))

		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(50 * time.Millisecond)
			cancel()
		}()

		_, err := service.LookupDomain(ctx, "cancelled.com", 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context canceled")
	})
}
