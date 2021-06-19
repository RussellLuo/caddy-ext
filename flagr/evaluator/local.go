package evaluator

import (
	"context"
	"sync"
	"time"

	"github.com/checkr/flagr/pkg/config"
	"github.com/checkr/flagr/pkg/handler"
	"github.com/checkr/flagr/swagger_gen/models"
	"github.com/checkr/flagr/swagger_gen/restapi/operations/evaluation"
)

var (
	onceLocal      sync.Once
	singletonLocal *Local
)

type Local struct {
	cache *handler.EvalCache
	eval  handler.Eval
}

func NewLocal(interval time.Duration, url string) (*Local, error) {
	var err error
	onceLocal.Do(func() {
		// Change the global configuration of Flagr once.
		config.Config.EvalCacheRefreshInterval = interval
		config.Config.EvalOnlyMode = true
		config.Config.DBDriver = "json_http"
		// The URL for exporting JSON format of the eval cache dump,
		// see https://checkr.github.io/flagr/api_docs/#operation/getExportEvalCacheJSON
		config.Config.DBConnectionStr = url + "/export/eval_cache/json"

		// Start singletonEvalCache once.
		defer func() {
			// EvalCache.Start() may panic if it fails.
			if r := recover(); r != nil {
				if e, ok := r.(error); ok {
					err = e
				}
			}
		}()
		cache := handler.GetEvalCache()
		cache.Start()

		singletonLocal = &Local{
			cache: cache,
			eval:  handler.NewEval(),
		}
	})
	return singletonLocal, err
}

func (l *Local) PostEvaluationBatch(ctx context.Context, req *models.EvaluationBatchRequest) (*models.EvaluationBatchResponse, error) {
	resp := l.eval.PostEvaluationBatch(evaluation.PostEvaluationBatchParams{Body: req})
	ok := resp.(*evaluation.PostEvaluationBatchOK)
	return ok.Payload, nil
}
