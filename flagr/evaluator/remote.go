package evaluator

import (
	"context"
	"sync"

	"github.com/checkr/flagr/swagger_gen/models"
	"github.com/checkr/goflagr"
)

var (
	onceRemote      sync.Once
	singletonRemote *Remote
)

type Remote struct {
	apiClient *goflagr.APIClient
}

func NewRemote(url string) *Remote {
	// Initialize the singleton once.
	onceRemote.Do(func() {
		singletonRemote = &Remote{
			apiClient: goflagr.NewAPIClient(&goflagr.Configuration{
				BasePath:      url,
				DefaultHeader: make(map[string]string),
				UserAgent:     "Caddy/go",
			}),
		}
	})
	return singletonRemote
}

func (r *Remote) PostEvaluationBatch(ctx context.Context, req *models.EvaluationBatchRequest) (*models.EvaluationBatchResponse, error) {
	resp, _, err := r.apiClient.EvaluationApi.PostEvaluationBatch(ctx, toGoFlagrRequest(req))
	if err != nil {
		return nil, err
	}
	return fromGoFlagrResponse(resp), nil
}

func toGoFlagrRequest(req *models.EvaluationBatchRequest) (result goflagr.EvaluationBatchRequest) {
	for _, e := range req.Entities {
		result.Entities = append(result.Entities, goflagr.EvaluationEntity{
			EntityID:      e.EntityID,
			EntityContext: &e.EntityContext,
		})
	}
	result.FlagKeys = req.FlagKeys
	return
}

func fromGoFlagrResponse(resp goflagr.EvaluationBatchResponse) *models.EvaluationBatchResponse {
	result := &models.EvaluationBatchResponse{}
	for _, r := range resp.EvaluationResults {
		result.EvaluationResults = append(result.EvaluationResults, &models.EvalResult{
			FlagID:            r.FlagID,
			FlagKey:           r.FlagKey,
			FlagSnapshotID:    r.FlagSnapshotID,
			SegmentID:         r.SegmentID,
			VariantID:         r.VariantID,
			VariantKey:        r.VariantKey,
			VariantAttachment: *r.VariantAttachment,
			EvalContext: &models.EvalContext{
				EntityID:      r.EvalContext.EntityID,
				EntityType:    r.EvalContext.EntityType,
				EntityContext: *r.EvalContext.EntityContext,
				EnableDebug:   r.EvalContext.EnableDebug,
				FlagID:        r.EvalContext.FlagID,
				FlagKey:       r.EvalContext.FlagKey,
			},
			Timestamp: r.Timestamp,
			EvalDebugLog: &models.EvalDebugLog{
				Msg:              r.EvalDebugLog.Msg,
				SegmentDebugLogs: convertSegmentDebugLogs(r.EvalDebugLog.SegmentDebugLogs),
			},
		})
	}
	return result
}

func convertSegmentDebugLogs(logs []goflagr.SegmentDebugLog) (result []*models.SegmentDebugLog) {
	for _, l := range logs {
		result = append(result, &models.SegmentDebugLog{
			Msg:       l.Msg,
			SegmentID: l.SegmentID,
		})
	}
	return
}
