package flagr

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/checkr/flagr/swagger_gen/models"
	"go.uber.org/zap"

	"github.com/RussellLuo/caddy-ext/flagr/evaluator"
)

var (
	regexpFullVar  = regexp.MustCompile(`^\{http\.request\..+\}$`)
	regexpShortVar = regexp.MustCompile(`^\{(\w+)\.(.+)\}$`)
)

func init() {
	caddy.RegisterModule(Flagr{})
}

type Evaluator interface {
	PostEvaluationBatch(ctx context.Context, req *models.EvaluationBatchRequest) (*models.EvaluationBatchResponse, error)
}

type ContextValue struct {
	Value      interface{}
	IsCaddyVar bool
	Converters []Converter `json:"-"`
}

// Flagr implements a handler for applying Feature Flags for HTTP requests
// by using checkr/flagr.
type Flagr struct {
	// The address of the flagr server.
	URL string `json:"url,omitempty"`

	// Which evaluator to use.
	// Supported options: "local" or "remote".
	Evaluator string `json:"evaluator,omitempty"`
	// The refresh interval of the internal eval cache (only used for the "local" evaluator).
	RefreshInterval string `json:"refresh_interval,omitempty"`

	// The unique ID from the entity, which is used to deterministically at
	// random to evaluate the flag result. Must be a Caddy variable.
	EntityID string `json:"entity_id,omitempty"`

	// The context parameters (key-value pairs) from the entity, which is used
	// to match the constraints.
	EntityContext map[string]interface{} `json:"entity_context,omitempty"`

	// A list of flag keys to look up.
	FlagKeys []string `json:"flag_keys,omitempty"`

	// Which element of the request to bind the evaluated variant keys.
	// Supported options: "header.NAME" or "query.NAME".
	BindVariantKeysTo string `json:"bind_variant_keys_to,omitempty"`

	logger         *zap.Logger
	evaluator      Evaluator
	entityIDVar    string
	entityContext  map[string]ContextValue
	bindToLocation string
	bindToName     string
}

// CaddyModule returns the Caddy module information.
func (Flagr) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.flagr",
		New: func() caddy.Module { return new(Flagr) }, // return a singleton.
	}
}

// Provision implements caddy.Provisioner.
func (f *Flagr) Provision(ctx caddy.Context) (err error) {
	f.logger = ctx.Logger(f)
	return f.provision()
}

func (f *Flagr) provision() (err error) {
	if f.URL == "" {
		return fmt.Errorf("empty url")
	}

	refreshInterval := 10 * time.Second
	if f.RefreshInterval != "" {
		d, err := time.ParseDuration(f.RefreshInterval)
		if err != nil {
			return err
		}
		refreshInterval = d
	}

	if f.Evaluator == "" {
		f.Evaluator = "local"
	}
	switch f.Evaluator {
	case "local":
		f.evaluator, err = evaluator.NewLocal(refreshInterval, f.URL)
		if err != nil {
			return err
		}
	case "remote":
		f.evaluator = evaluator.NewRemote(f.URL)
	default:
		return fmt.Errorf("unsupported evaluator %q", f.Evaluator)
	}

	f.entityIDVar, err = parseVar(f.EntityID)
	if err != nil {
		return err
	}

	f.entityContext = make(map[string]ContextValue)
	for k, v := range f.EntityContext {
		cv := ContextValue{Value: v}

		// Handle string values specially.
		if s, ok := v.(string); ok {
			parts := strings.Split(s, "|")
			val := parts[0]

			if p, err := parseVar(val); err == nil {
				cv.Value = p
				cv.IsCaddyVar = true
			} else {
				cv.Value = val
			}

			for _, name := range parts[1:] {
				c, err := GetConverter(name)
				if err != nil {
					return err
				}
				cv.Converters = append(cv.Converters, c)
			}
		}

		f.entityContext[k] = cv
	}

	if f.BindVariantKeysTo == "" {
		f.BindVariantKeysTo = "header.X-Flagr-Variant"
	}

	parts := strings.SplitN(f.BindVariantKeysTo, ".", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid bind_variant_key_to")
	}
	f.bindToLocation, f.bindToName = parts[0], parts[1]

	return nil
}

// Cleanup cleans up the resources made by rl during provisioning.
func (f *Flagr) Cleanup() error {
	return nil
}

// Validate implements caddy.Validator.
func (f *Flagr) Validate() error {
	if f.entityIDVar == "" {
		return fmt.Errorf("invalid entity_id")
	}
	if len(f.entityContext) == 0 {
		return fmt.Errorf("invalid entity_context")
	}
	if len(f.FlagKeys) == 0 {
		return fmt.Errorf("empty flag_keys")
	}

	if f.bindToLocation != "header" && f.bindToLocation != "query" {
		return fmt.Errorf("invalid location %q from bind_variant_key_to", f.bindToLocation)
	}
	if f.bindToName == "" {
		return fmt.Errorf("emtpy name from bind_variant_key_to")
	}

	return nil
}

// ServeHTTP implements caddyhttp.MiddlewareHandler.
func (f *Flagr) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	repl := r.Context().Value(caddy.ReplacerCtxKey).(*caddy.Replacer)
	entityID := repl.ReplaceAll(f.entityIDVar, "")
	if entityID == "" {
		f.logger.Info("entityID is evaluated to be empty",
			zap.Any("entityIDVar", f.entityIDVar),
			zap.Any("originalEntityContext", f.EntityContext),
		)
		return next.ServeHTTP(w, r)
	}

	entityContext, err := evalEntityContext(f.entityContext, repl)
	if err != nil {
		f.logger.Error("failed to evaluate the entity context",
			zap.String("entityID", entityID),
			zap.Any("originalEntityContext", f.EntityContext),
			zap.Error(err),
		)
		return next.ServeHTTP(w, r)
	}

	f.logger.Debug("ready to evaluate the request entity by flagr",
		zap.String("entityID", entityID),
		zap.Any("entityContext", entityContext),
	)

	resp, err := f.evaluator.PostEvaluationBatch(context.Background(), &models.EvaluationBatchRequest{
		Entities: []*models.EvaluationEntity{
			{
				EntityID:      entityID,
				EntityContext: entityContext,
			},
		},
		FlagKeys: f.FlagKeys,
	})
	if err != nil {
		f.logger.Error("failed to evaluate the request entity by flagr",
			zap.String("entityID", entityID),
			zap.Any("originalEntityContext", f.EntityContext),
			zap.Error(err),
		)
		return next.ServeHTTP(w, r)
	}

	for _, er := range resp.EvaluationResults {
		if er.VariantKey != "" {
			variant := er.FlagKey + "." + er.VariantKey
			switch f.bindToLocation {
			case "header":
				r.Header.Add(f.bindToName, variant)
			case "query":
				r.URL.Query().Add(f.bindToName, variant)
			}
		}
	}

	return next.ServeHTTP(w, r)
}

func evalEntityContext(entityCtx map[string]ContextValue, repl *caddy.Replacer) (interface{}, error) {
	out := make(map[string]interface{})
	for k, cv := range entityCtx {
		v := cv.Value
		if cv.IsCaddyVar {
			// Use evaluated values for placeholders.
			v = repl.ReplaceAll(cv.Value.(string), "")
		}
		out[k] = v

		// If v is of type string, convert it by a list of converters, if any.
		if s, ok := v.(string); ok {
			for _, c := range cv.Converters {
				r, err := c(s)
				if err != nil {
					return nil, err
				}
				out[k] = r
			}
		}
	}
	return out, nil
}

// parseVar transforms shorthand variables into Caddy-style placeholders.
// Copied from ratelimit/ratelimit.go.
func parseVar(s string) (v string, err error) {
	if regexpFullVar.MatchString(s) {
		// If the variable is already a fully-qualified Caddy placeholder,
		// return it as is.
		return s, nil
	}

	result := regexpShortVar.FindStringSubmatch(s)
	if len(result) != 3 {
		return "", fmt.Errorf("invalid key variable: %q", s)
	}
	location, name := result[1], result[2]

	switch location {
	case "path":
		v = fmt.Sprintf("{http.request.uri.path.%s}", name)
	case "query":
		v = fmt.Sprintf("{http.request.uri.query.%s}", name)
	case "header":
		v = fmt.Sprintf("{http.request.header.%s}", name)
	case "cookie":
		v = fmt.Sprintf("{http.request.cookie.%s}", name)
	case "body":
		v = fmt.Sprintf("{http.request.body.%s}", name)
	default:
		err = fmt.Errorf("unrecognized key variable: %q", s)
	}

	return
}

// Interface guards
var (
	_ caddy.Provisioner           = (*Flagr)(nil)
	_ caddy.CleanerUpper          = (*Flagr)(nil)
	_ caddy.Validator             = (*Flagr)(nil)
	_ caddyhttp.MiddlewareHandler = (*Flagr)(nil)
)
