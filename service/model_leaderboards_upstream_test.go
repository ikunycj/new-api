package service

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchArenaLeaderboard(t *testing.T) {
	tests := []struct {
		name          string
		source        arenaLeaderboard
		total         int
		expectedScore float64
	}{
		{"agent", arenaLeaderboard{"agent_capability", "overall"}, 1, 8.1},
		{"coding", arenaLeaderboard{"agent_capability", "code"}, 1, 8.1},
		{"chat", arenaLeaderboard{"text_style_control", "overall"}, 201, 1200},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var requests atomic.Int32
			lastPageRequested := make(chan struct{}, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				query := r.URL.Query()
				assert.Equal(t, "lmarena-ai/leaderboard-dataset", query.Get("dataset"))
				assert.Equal(t, test.source.config, query.Get("config"))
				assert.Equal(t, "latest", query.Get("split"))
				assert.Equal(t, fmt.Sprintf(`"category"='%s'`, test.source.category), query.Get("where"))
				assert.Equal(t, `"rank"`, query.Get("orderby"))
				assert.Equal(t, "100", query.Get("length"))
				offset, err := strconv.Atoi(query.Get("offset"))
				requests.Add(1)
				if !assert.NoError(t, err) || offset < 0 || offset%100 != 0 || offset >= test.total {
					http.Error(w, "unexpected offset", http.StatusBadRequest)
					return
				}
				// The first continuation cannot finish until the final page is
				// requested. A serial traversal would exhaust the context.
				switch offset {
				case 100:
					select {
					case <-lastPageRequested:
					case <-r.Context().Done():
						return
					}
				case 200:
					lastPageRequested <- struct{}{}
				}
				rows := make([]map[string]any, 0, min(100, test.total-offset))
				for i := offset; i < min(offset+100, test.total); i++ {
					rows = append(rows, map[string]any{"row": map[string]any{
						"model_name":               fmt.Sprintf("model-%d", i+1),
						"rank":                     i + 1,
						"index_score":              8.1,
						"rating":                   1200,
						"category":                 test.source.category,
						"leaderboard_publish_date": "2026-11-22",
					}})
				}
				response, err := common.Marshal(map[string]any{"rows": rows, "num_rows_total": test.total, "partial": false})
				if !assert.NoError(t, err) {
					http.Error(w, "invalid fixture", http.StatusInternalServerError)
					return
				}
				_, _ = w.Write(response)
			}))
			defer server.Close()

			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			rows, err := fetchArenaLeaderboard(ctx, server.Client(), server.URL, test.source)
			require.NoError(t, err)
			require.Len(t, rows, test.total)
			assert.Equal(t, model.ModelLeaderboardEntry{ModelName: "model-1", Rank: 1, Score: test.expectedScore}, rows[0])
			assert.Equal(t, model.ModelLeaderboardEntry{
				ModelName: fmt.Sprintf("model-%d", test.total),
				Rank:      test.total,
				Score:     test.expectedScore,
			}, rows[len(rows)-1])
			assert.Equal(t, (test.total+99)/100, int(requests.Load()))
		})
	}
}

func TestFetchArenaLeaderboardRejectsPartialResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"rows":[{"row":{"model_name":"model-1","rank":1,"index_score":8.1,"category":"overall","leaderboard_publish_date":"2026-11-22"}}],"num_rows_total":1,"partial":true}`))
	}))
	defer server.Close()

	rows, err := fetchArenaLeaderboard(t.Context(), server.Client(), server.URL, arenaLeaderboard{"agent_capability", "overall"})
	require.ErrorContains(t, err, "incomplete or inconsistent leaderboard")
	assert.Nil(t, rows)
}
