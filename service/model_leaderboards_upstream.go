package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"golang.org/x/sync/errgroup"
)

const (
	arenaLeaderboardEndpoint = "https://datasets-server.huggingface.co/filter"
	arenaLeaderboardPageSize = 100
)

var arenaLeaderboardClient = &http.Client{Timeout: 10 * time.Second}

type arenaLeaderboard struct {
	config   string
	category string
}

type arenaLeaderboardPage struct {
	Total   int  `json:"num_rows_total"`
	Partial bool `json:"partial"`
	Rows    []struct {
		Row struct {
			ModelName   string   `json:"model_name"`
			Rank        int      `json:"rank"`
			IndexScore  *float64 `json:"index_score"`
			Rating      *float64 `json:"rating"`
			Category    string   `json:"category"`
			PublishDate string   `json:"leaderboard_publish_date"`
		} `json:"row"`
	} `json:"rows"`
}

func fetchModelLeaderboards(ctx context.Context) (map[string][]model.ModelLeaderboardEntry, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	sources := map[string]arenaLeaderboard{
		model.ModelLeaderboardAgent:  {config: "agent_capability", category: "overall"},
		model.ModelLeaderboardCoding: {config: "agent_capability", category: "code"},
		model.ModelLeaderboardChat:   {config: "text_style_control", category: "overall"},
	}

	var mu sync.Mutex
	boards := make(map[string][]model.ModelLeaderboardEntry, len(sources))
	group, ctx := errgroup.WithContext(ctx)
	for key, source := range sources {
		group.Go(func() error {
			entries, err := fetchArenaLeaderboard(ctx, arenaLeaderboardClient, arenaLeaderboardEndpoint, source)
			if err != nil {
				return fmt.Errorf("Arena.ai %s leaderboard: %w", key, err)
			}
			mu.Lock()
			boards[key] = entries
			mu.Unlock()
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		return nil, err
	}
	return boards, nil
}

func fetchArenaLeaderboard(ctx context.Context, client *http.Client, endpoint string, source arenaLeaderboard) ([]model.ModelLeaderboardEntry, error) {
	query := url.Values{
		"dataset": {"lmarena-ai/leaderboard-dataset"},
		"config":  {source.config},
		"split":   {"latest"},
		"where":   {fmt.Sprintf(`"category"='%s'`, source.category)},
		"orderby": {`"rank"`},
		"length":  {strconv.Itoa(arenaLeaderboardPageSize)},
	}
	fetchPage := func(ctx context.Context, offset int) (arenaLeaderboardPage, error) {
		params := maps.Clone(query)
		params.Set("offset", strconv.Itoa(offset))
		page, err := fetchArenaLeaderboardPage(ctx, client, endpoint+"?"+params.Encode())
		if err != nil {
			return page, fmt.Errorf("offset %d: %w", offset, err)
		}
		return page, nil
	}
	first, err := fetchPage(ctx, 0)
	if err != nil {
		return nil, err
	}
	total := first.Total
	if first.Partial || total <= 0 || len(first.Rows) != min(arenaLeaderboardPageSize, total) {
		return nil, errors.New("incomplete or inconsistent leaderboard at offset 0")
	}

	pages := make([]arenaLeaderboardPage, (total-1)/arenaLeaderboardPageSize+1)
	pages[0] = first
	group, pageCtx := errgroup.WithContext(ctx)
	group.SetLimit(4)
	for index := 1; index < len(pages); index++ {
		group.Go(func() error {
			page, err := fetchPage(pageCtx, index*arenaLeaderboardPageSize)
			pages[index] = page
			return err
		})
	}
	if err := group.Wait(); err != nil {
		return nil, err
	}

	entries := make([]model.ModelLeaderboardEntry, 0, total)
	models := make(map[string]struct{}, total)
	publishDate := first.Rows[0].Row.PublishDate
	for index, page := range pages {
		offset := index * arenaLeaderboardPageSize
		if page.Partial || page.Total != total ||
			len(page.Rows) != min(arenaLeaderboardPageSize, total-offset) {
			return nil, fmt.Errorf("incomplete or inconsistent leaderboard at offset %d", offset)
		}

		for _, result := range page.Rows {
			row := result.Row
			score := row.IndexScore
			if source.config == "text_style_control" {
				score = row.Rating
			}
			name := strings.TrimSpace(row.ModelName)
			if row.Category != source.category || row.PublishDate == "" || row.PublishDate != publishDate ||
				name == "" || row.Rank < 1 || score == nil ||
				(len(entries) > 0 && row.Rank < entries[len(entries)-1].Rank) {
				return nil, fmt.Errorf("invalid leaderboard entry at offset %d", len(entries))
			}
			if _, duplicate := models[name]; duplicate {
				return nil, fmt.Errorf("duplicate leaderboard model %q at offset %d", name, len(entries))
			}
			models[name] = struct{}{}
			entries = append(entries, model.ModelLeaderboardEntry{ModelName: name, Rank: row.Rank, Score: *score})
		}
	}
	return entries, nil
}

func fetchArenaLeaderboardPage(ctx context.Context, client *http.Client, endpoint string) (arenaLeaderboardPage, error) {
	var page arenaLeaderboardPage
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return page, err
	}
	for attempt := 0; attempt < 2; attempt++ {
		resp, requestErr := client.Do(req)
		if requestErr == nil {
			if resp.StatusCode == http.StatusOK {
				requestErr = common.DecodeJson(io.LimitReader(resp.Body, 2<<20), &page)
				resp.Body.Close()
				if requestErr == nil {
					return page, nil
				}
				var networkErr net.Error
				if !errors.As(requestErr, &networkErr) || !networkErr.Timeout() {
					return page, requestErr
				}
			} else {
				resp.Body.Close()
				requestErr = fmt.Errorf("unexpected HTTP status %d", resp.StatusCode)
				if resp.StatusCode < http.StatusInternalServerError {
					return page, requestErr
				}
			}
		}
		err = requestErr
		if ctx.Err() != nil {
			break
		}
		page = arenaLeaderboardPage{}
	}
	return page, err
}
