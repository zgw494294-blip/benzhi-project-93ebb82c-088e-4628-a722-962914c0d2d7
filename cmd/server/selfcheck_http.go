package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"benzhi-project-93ebb82c-088e-4628-a722-962914c0d2d7/internal/domain"
)

type selfcheckClient struct {
	base   string
	client *http.Client
	seq    int
}

type resultEnvelope struct {
	Value      domain.Aggregate `json:"value"`
	Idempotent bool             `json:"idempotent"`
}

func prepareSelfcheckData() (string, func(), error) {
	dir, err := os.MkdirTemp("", "narration-selfcheck-*")
	if err != nil {
		return "", func() {}, fmt.Errorf("创建自检目录: %w", err)
	}
	return dir, func() { _ = os.RemoveAll(dir) }, nil
}

func runSelfcheck(base string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	c := &selfcheckClient{base: base, client: &http.Client{Timeout: 4 * time.Second}}
	if err := c.waitHealthy(ctx); err != nil {
		return err
	}
	agg, err := c.create(ctx)
	if err != nil {
		return err
	}
	productionID := agg.Production.ID
	agg, err = c.postMutation(ctx, "/api/productions/"+productionID+"/segments", map[string]any{
		"expectedRevision": agg.Production.Revision, "idempotencyKey": c.key("scene"), "id": "scene-1", "kind": "SCENE", "start_ms": 0, "end_ms": 30000, "label": "灯塔外景",
	})
	if err != nil {
		return err
	}
	agg, err = c.postMutation(ctx, "/api/productions/"+productionID+"/segments", map[string]any{
		"expectedRevision": agg.Production.Revision, "idempotencyKey": c.key("dialogue"), "id": "dialogue-1", "scene_id": "scene-1", "kind": "DIALOGUE", "start_ms": 9000, "end_ms": 13000, "label": "人物对白",
	})
	if err != nil {
		return err
	}
	var windows struct {
		Windows []struct {
			StartMS int64 `json:"start_ms"`
			EndMS   int64 `json:"end_ms"`
		} `json:"windows"`
		Conflicts []any `json:"conflicts"`
	}
	if err := c.getJSON(ctx, "/api/productions/"+productionID+"/windows", &windows); err != nil {
		return err
	}
	if len(windows.Windows) != 2 || len(windows.Conflicts) != 0 {
		return fmt.Errorf("候选窗口结果不符合预期: %+v", windows)
	}
	agg, err = c.postMutation(ctx, "/api/productions/"+productionID+"/timeline/finalize", meta(agg, c.key("timeline")))
	if err != nil {
		return err
	}
	agg, err = c.saveCue(ctx, agg, "雾从海面慢慢散开，灯塔的光掠过礁石。")
	if err != nil {
		return err
	}
	agg, err = c.saveCue(ctx, agg, "晨雾从海面慢慢散开，灯塔的光掠过黑色礁石。")
	if err != nil {
		return err
	}
	var diff struct {
		Added   string `json:"added"`
		Removed string `json:"removed"`
	}
	if err := c.getJSON(ctx, "/api/productions/"+productionID+"/cues/cue-1/diff?from=1&to=2", &diff); err != nil {
		return err
	}
	if diff.Added == "" && diff.Removed == "" {
		return fmt.Errorf("相邻提示版本差异为空")
	}
	agg, err = c.postMutation(ctx, "/api/productions/"+productionID+"/validation", meta(agg, c.key("validation-1")))
	if err != nil {
		return err
	}
	agg, err = c.rehearse(ctx, agg, "take-1")
	if err != nil {
		return err
	}
	firstTakeHash := agg.Rehearsals[len(agg.Rehearsals)-1].CueVersionSetHash
	agg, err = c.postMutation(ctx, "/api/productions/"+productionID+"/reviews", map[string]any{
		"expectedRevision": agg.Production.Revision, "idempotencyKey": c.key("revise"), "id": "decision-revise", "cue_id": "cue-1", "action": "REVISE", "reason": "开头需要明确晨雾，便于听众建立时段", "reviewer": "顾审校",
	})
	if err != nil {
		return err
	}
	if agg.Production.State != domain.StateRevising {
		return fmt.Errorf("退修后状态为 %s", agg.Production.State)
	}
	agg, err = c.saveCue(ctx, agg, "清晨，雾从海面慢慢散开，灯塔的光掠过黑色礁石。")
	if err != nil {
		return err
	}
	if agg.Rehearsals[0].InvalidatedAt == nil || len(agg.Decisions) != 0 {
		return fmt.Errorf("改稿未使旧排演与旧审校失效")
	}
	agg, err = c.postMutation(ctx, "/api/productions/"+productionID+"/validation", meta(agg, c.key("validation-2")))
	if err != nil {
		return err
	}
	agg, err = c.rehearse(ctx, agg, "take-2")
	if err != nil {
		return err
	}
	if agg.Rehearsals[len(agg.Rehearsals)-1].CueVersionSetHash == firstTakeHash {
		return fmt.Errorf("改稿后的排演版本集合摘要未变化")
	}
	agg, err = c.postMutation(ctx, "/api/productions/"+productionID+"/reviews", map[string]any{
		"expectedRevision": agg.Production.Revision, "idempotencyKey": c.key("accept"), "id": "decision-accept", "cue_id": "cue-1", "action": "ACCEPT", "reviewer": "顾审校",
	})
	if err != nil {
		return err
	}
	agg, err = c.postMutation(ctx, "/api/productions/"+productionID+"/approve", meta(agg, c.key("approve")))
	if err != nil {
		return err
	}
	agg, err = c.postMutation(ctx, "/api/productions/"+productionID+"/release", map[string]any{
		"expectedRevision": agg.Production.Revision, "idempotencyKey": c.key("release"), "released_by": "顾审校",
	})
	if err != nil {
		return err
	}
	if agg.Production.State != domain.StateReleased || agg.Release == nil || !agg.Release.VerifyHash() {
		return fmt.Errorf("发布状态或摘要无效")
	}
	jsonData, err := c.getBytes(ctx, "/api/productions/"+productionID+"/release.json")
	if err != nil {
		return err
	}
	var snapshot domain.ReleaseSnapshot
	if err := json.Unmarshal(jsonData, &snapshot); err != nil {
		return fmt.Errorf("发布 JSON 无效: %w", err)
	}
	if !snapshot.VerifyHash() || snapshot.ContentHash != agg.Release.ContentHash {
		return fmt.Errorf("下载 JSON 摘要不一致")
	}
	vtt, err := c.getBytes(ctx, "/api/productions/"+productionID+"/release.vtt")
	if err != nil {
		return err
	}
	if !strings.HasPrefix(string(vtt), "WEBVTT\n\n") || !strings.Contains(string(vtt), snapshot.ApprovedCues[0].Text) {
		return fmt.Errorf("WebVTT 导出内容不完整")
	}
	return nil
}

func (c *selfcheckClient) key(prefix string) string {
	c.seq++
	return fmt.Sprintf("selfcheck-%s-%d", prefix, c.seq)
}

func meta(agg domain.Aggregate, key string) map[string]any {
	return map[string]any{"expectedRevision": agg.Production.Revision, "idempotencyKey": key}
}

func (c *selfcheckClient) waitHealthy(ctx context.Context) error {
	deadline := time.NewTicker(25 * time.Millisecond)
	defer deadline.Stop()
	for {
		var result map[string]any
		if err := c.getJSON(ctx, "/healthz", &result); err == nil && result["status"] == "ok" {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("等待服务健康: %w", ctx.Err())
		case <-deadline.C:
		}
	}
}

func (c *selfcheckClient) create(ctx context.Context) (domain.Aggregate, error) {
	body := map[string]any{
		"id": "selfcheck-production", "title": "自检影片", "language": "zh-CN", "duration_ms": 30000, "frame_rate": 25,
		"participants":   []map[string]string{{"name": "林编剧", "role": "WRITER"}, {"name": "周排演", "role": "PERFORMER"}, {"name": "顾审校", "role": "REVIEWER"}},
		"idempotencyKey": c.key("create"),
	}
	return c.postMutation(ctx, "/api/productions", body)
}

func (c *selfcheckClient) saveCue(ctx context.Context, agg domain.Aggregate, text string) (domain.Aggregate, error) {
	return c.postMutation(ctx, "/api/productions/"+agg.Production.ID+"/cues", map[string]any{
		"expectedRevision": agg.Production.Revision, "idempotencyKey": c.key("cue"), "cue_id": "cue-1", "window_start_ms": 1000, "window_end_ms": 8000,
		"text": text, "intent": "平静交代环境", "planned_chars_per_second": 6, "pause_budget_ms": 250,
	})
}

func (c *selfcheckClient) rehearse(ctx context.Context, agg domain.Aggregate, id string) (domain.Aggregate, error) {
	cue := agg.LatestCues()[0]
	return c.postMutation(ctx, "/api/productions/"+agg.Production.ID+"/rehearsals", map[string]any{
		"expectedRevision": agg.Production.Revision, "idempotencyKey": c.key("take"), "id": id,
		"measurements": []map[string]any{{"cue_id": cue.ID, "cue_version": cue.Version, "actual_start_ms": 1250, "actual_end_ms": 7750, "spoken_duration_ms": 4200, "pause_ms": 300}}, "findings": []any{},
	})
}

func (c *selfcheckClient) postMutation(ctx context.Context, path string, body any) (domain.Aggregate, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return domain.Aggregate{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+path, bytes.NewReader(data))
	if err != nil {
		return domain.Aggregate{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	response, err := c.client.Do(req)
	if err != nil {
		return domain.Aggregate{}, fmt.Errorf("POST %s: %w", path, err)
	}
	defer response.Body.Close()
	responseData, err := io.ReadAll(response.Body)
	if err != nil {
		return domain.Aggregate{}, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return domain.Aggregate{}, fmt.Errorf("POST %s 返回 %d: %s", path, response.StatusCode, responseData)
	}
	var envelope resultEnvelope
	if err := json.Unmarshal(responseData, &envelope); err != nil {
		return domain.Aggregate{}, fmt.Errorf("解析 POST %s: %w", path, err)
	}
	return envelope.Value, nil
}

func (c *selfcheckClient) getJSON(ctx context.Context, path string, dst any) error {
	data, err := c.getBytes(ctx, path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, dst); err != nil {
		return fmt.Errorf("解析 GET %s: %w", path, err)
	}
	return nil
}

func (c *selfcheckClient) getBytes(ctx context.Context, path string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+path, nil)
	if err != nil {
		return nil, err
	}
	response, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", path, err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s 返回 %d: %s", path, response.StatusCode, data)
	}
	return data, nil
}
